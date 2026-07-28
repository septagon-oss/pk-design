package handoff

// Implements: REQ-011.
// Per: ADR-0076.
// Discipline: C-14.
// builder.go projects a normalized DTCG token set into the portable snapshot
// contract without introducing a renderer or provider dependency.
//
// ADR: ADR-0029 (file purpose declaration), ADR-0076 (layered design delivery).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"fmt"
	"maps"
	"strings"

	"github.com/septagon-oss/pk-design/pkg/tokens"
)

// SetSnapshotOptions supplies delivery identity and source ownership when a
// DTCG set is materialized for an external editor.
type SetSnapshotOptions struct {
	Profile    Profile
	Provenance Provenance
	Mode       string
	Origin     Origin
	// OriginForPath optionally resolves ownership per dotted Set path. When
	// set, it takes precedence over Origin.
	OriginForPath func(path string) (Origin, error)
	Metadata      map[string]string
}

// FromSet projects every token in set into one snapshot mode. Callers that
// compose multiple modes can append the resulting tokens and normalize once.
func FromSet(set tokens.Set, options SetSnapshotOptions) (Snapshot, error) {
	normalizedSet, err := set.Normalize()
	if err != nil {
		return Snapshot{}, fmt.Errorf("handoff from set: %w", err)
	}
	mode := strings.TrimSpace(options.Mode)
	if mode == "" {
		mode = DefaultMode
	}
	keys, err := normalizedSet.Keys()
	if err != nil {
		return Snapshot{}, fmt.Errorf("handoff from set keys: %w", err)
	}
	snapshot := Snapshot{
		SchemaVersion: SnapshotSchemaVersion,
		Profile:       options.Profile,
		Provenance:    options.Provenance,
		Tokens:        make([]Token, 0, len(keys)),
		Metadata:      maps.Clone(options.Metadata),
	}
	for _, path := range keys {
		token, ok, err := normalizedSet.Lookup(path)
		if err != nil {
			return Snapshot{}, fmt.Errorf("handoff from set token %q: %w", path, err)
		}
		if !ok {
			return Snapshot{}, fmt.Errorf("handoff from set token %q disappeared after normalization", path)
		}
		origin := options.Origin
		if options.OriginForPath != nil {
			origin, err = options.OriginForPath(path)
			if err != nil {
				return Snapshot{}, fmt.Errorf("handoff from set origin %q: %w", path, err)
			}
		}
		if origin.Pointer == "" {
			tokenPointer, pointerErr := PathFromSegments(strings.Split(path, ".")...)
			if pointerErr != nil {
				return Snapshot{}, fmt.Errorf("handoff from set pointer %q: %w", path, pointerErr)
			}
			origin.Pointer = "/tokens" + tokenPointer + "/$value"
		}
		tokenPointer, err := PathFromSegments(strings.Split(token.Path, ".")...)
		if err != nil {
			return Snapshot{}, fmt.Errorf("handoff from set token pointer %q: %w", token.Path, err)
		}
		snapshot.Tokens = append(snapshot.Tokens, Token{
			Path:        tokenPointer,
			Mode:        mode,
			Type:        token.Type,
			Value:       tokens.CopyValue(token.Value),
			Description: token.Description,
			Origin:      origin,
		})
	}
	return snapshot.Normalize()
}
