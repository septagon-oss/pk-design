package handoff

// Implements: REQ-011.
// Per: ADR-0076.
// Discipline: C-14.
// diff.go computes governed changes from an imported snapshot and applies them
// atomically with digest, expected-value, profile, and ownership checks.
//
// ADR: ADR-0029 (file purpose declaration), ADR-0076 (layered design delivery).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/septagon-oss/pk-design/pkg/tokens"
)

// Diff compares an imported base snapshot with edited managed state and returns
// the smallest deterministic change set. External adapters should reconstruct
// edited from managed provider values and retain the base token origins.
func Diff(base, edited Snapshot, provenance Provenance) (ChangeSet, error) {
	normalizedBase, err := base.Normalize()
	if err != nil {
		return ChangeSet{}, fmt.Errorf("handoff diff base: %w", err)
	}
	normalizedEdited, err := edited.Normalize()
	if err != nil {
		return ChangeSet{}, fmt.Errorf("handoff diff edited: %w", err)
	}
	if !sameProfile(normalizedBase.Profile, normalizedEdited.Profile) {
		return ChangeSet{}, ConflictError{
			Code:    ConflictProfile,
			Message: "edited snapshot targets a different profile",
		}
	}
	provenance, err = normalizeProvenance(provenance)
	if err != nil {
		return ChangeSet{}, fmt.Errorf("handoff diff provenance: %w", err)
	}
	baseDigest, err := normalizedBase.Digest()
	if err != nil {
		return ChangeSet{}, err
	}

	baseTokens := indexTokens(normalizedBase.Tokens)
	editedTokens := indexTokens(normalizedEdited.Tokens)
	keys := make([]string, 0, len(baseTokens)+len(editedTokens))
	for key := range baseTokens {
		keys = append(keys, key)
	}
	for key := range editedTokens {
		if _, exists := baseTokens[key]; !exists {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)

	changes := make([]Change, 0)
	for _, key := range keys {
		before, hadBefore := baseTokens[key]
		after, hasAfter := editedTokens[key]
		switch {
		case !hadBefore && hasAfter:
			if !after.Origin.Writable {
				return ChangeSet{}, ownershipConflict(key, "new token origin is read-only")
			}
			afterCopy := copyToken(after)
			changes = append(changes, Change{Operation: OperationAdd, After: &afterCopy})
		case hadBefore && !hasAfter:
			if !before.Origin.Writable {
				return ChangeSet{}, ownershipConflict(key, "read-only token cannot be removed")
			}
			beforeCopy := copyToken(before)
			changes = append(changes, Change{Operation: OperationRemove, Before: &beforeCopy})
		case hadBefore && hasAfter:
			if !sameOrigin(before.Origin, after.Origin) {
				return ChangeSet{}, ownershipConflict(key, "external edit changed token origin")
			}
			if sameToken(before, after) {
				continue
			}
			if !before.Origin.Writable {
				return ChangeSet{}, ownershipConflict(key, "read-only token cannot be updated")
			}
			beforeCopy := copyToken(before)
			afterCopy := copyToken(after)
			changes = append(changes, Change{
				Operation: OperationUpdate,
				Before:    &beforeCopy,
				After:     &afterCopy,
			})
		}
	}
	if len(changes) == 0 {
		return ChangeSet{}, fmt.Errorf("handoff diff: managed token state has no changes")
	}
	result := ChangeSet{
		SchemaVersion: ChangeSetSchemaVersion,
		Profile:       normalizedBase.Profile,
		BaseDigest:    baseDigest,
		Provenance:    provenance,
		Changes:       changes,
	}
	if err := result.Validate(); err != nil {
		return ChangeSet{}, err
	}
	return result, nil
}

// Apply applies a complete change set to the current snapshot in memory.
// Nothing is mutated on failure. Source-specific receivers turn the returned
// snapshot into a write plan, run their compilers and validators, then commit
// those writes atomically.
func Apply(current Snapshot, changes ChangeSet) (Snapshot, error) {
	normalizedCurrent, err := current.Normalize()
	if err != nil {
		return Snapshot{}, fmt.Errorf("handoff apply current: %w", err)
	}
	if err := changes.Validate(); err != nil {
		return Snapshot{}, err
	}
	if !sameProfile(normalizedCurrent.Profile, changes.Profile) {
		return Snapshot{}, ConflictError{
			Code:    ConflictProfile,
			Message: "change set targets a different profile",
		}
	}
	currentDigest, err := normalizedCurrent.Digest()
	if err != nil {
		return Snapshot{}, err
	}
	if currentDigest != changes.BaseDigest {
		return Snapshot{}, ConflictError{
			Code: ConflictBaseDigest,
			Message: fmt.Sprintf(
				"current digest %s does not match change base %s",
				currentDigest,
				changes.BaseDigest,
			),
		}
	}

	next := indexTokens(normalizedCurrent.Tokens)
	for _, change := range changes.Changes {
		key := changeCoordinate(change)
		switch change.Operation {
		case OperationAdd:
			if _, exists := next[key]; exists {
				return Snapshot{}, ConflictError{
					Code:       ConflictTokenExists,
					Coordinate: key,
					Message:    "token already exists",
				}
			}
			next[key] = copyToken(*change.After)
		case OperationUpdate:
			existing, exists := next[key]
			if !exists {
				return Snapshot{}, ConflictError{
					Code:       ConflictTokenMissing,
					Coordinate: key,
					Message:    "token no longer exists",
				}
			}
			if !sameToken(existing, *change.Before) {
				return Snapshot{}, ConflictError{
					Code:       ConflictTokenMismatch,
					Coordinate: key,
					Message:    "current token does not match expected previous state",
				}
			}
			if !existing.Origin.Writable || !sameOrigin(existing.Origin, change.After.Origin) {
				return Snapshot{}, ownershipConflict(key, "token origin does not permit this update")
			}
			next[key] = copyToken(*change.After)
		case OperationRemove:
			existing, exists := next[key]
			if !exists {
				return Snapshot{}, ConflictError{
					Code:       ConflictTokenMissing,
					Coordinate: key,
					Message:    "token no longer exists",
				}
			}
			if !sameToken(existing, *change.Before) {
				return Snapshot{}, ConflictError{
					Code:       ConflictTokenMismatch,
					Coordinate: key,
					Message:    "current token does not match expected previous state",
				}
			}
			if !existing.Origin.Writable {
				return Snapshot{}, ownershipConflict(key, "token origin does not permit removal")
			}
			delete(next, key)
		}
	}
	tokensOut := make([]Token, 0, len(next))
	for _, token := range next {
		tokensOut = append(tokensOut, copyToken(token))
	}
	slices.SortFunc(tokensOut, compareTokens)
	result := Snapshot{
		SchemaVersion: SnapshotSchemaVersion,
		Profile:       normalizedCurrent.Profile,
		Provenance:    changes.Provenance,
		Tokens:        tokensOut,
		Metadata:      normalizedCurrent.Metadata,
	}
	return result.Normalize()
}

func indexTokens(values []Token) map[string]Token {
	out := make(map[string]Token, len(values))
	for _, token := range values {
		out[coordinate(token)] = copyToken(token)
	}
	return out
}

func changeCoordinate(change Change) string {
	if change.After != nil {
		return coordinate(*change.After)
	}
	return coordinate(*change.Before)
}

func copyToken(token Token) Token {
	token.Value = tokens.CopyValue(token.Value)
	return token
}

func sameToken(left, right Token) bool {
	if left.Path != right.Path ||
		left.Mode != right.Mode ||
		left.Type != right.Type ||
		left.Description != right.Description ||
		!sameOrigin(left.Origin, right.Origin) {
		return false
	}
	leftJSON, leftErr := json.Marshal(left.Value)
	rightJSON, rightErr := json.Marshal(right.Value)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

func ownershipConflict(coordinate, message string) ConflictError {
	return ConflictError{
		Code:       ConflictOwnership,
		Coordinate: coordinate,
		Message:    message,
	}
}
