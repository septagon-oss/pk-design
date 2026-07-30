package handoff

// Implements: REQ-011.
// Per: ADR-0076.
// Discipline: C-14.
// normalize.go validates and canonicalizes provider-neutral handoff documents,
// and computes stable content digests independent of generation time.
//
// ADR: ADR-0029 (file purpose declaration), ADR-0076 (layered design delivery).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/septagon-oss/pk-design/pkg/tokens"
)

var profileIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[./-][a-z0-9]+)*$`)

// Normalize validates a snapshot and returns a deep, deterministically sorted
// copy. The receiver can retain the result without sharing caller-owned maps or
// composite token values.
func (s Snapshot) Normalize() (Snapshot, error) {
	if s.SchemaVersion != SnapshotSchemaVersion {
		return Snapshot{}, fmt.Errorf(
			"handoff snapshot: unsupported schemaVersion %q (want %q)",
			s.SchemaVersion,
			SnapshotSchemaVersion,
		)
	}
	profile, err := normalizeProfile(s.Profile)
	if err != nil {
		return Snapshot{}, fmt.Errorf("handoff snapshot profile: %w", err)
	}
	provenance, err := normalizeProvenance(s.Provenance)
	if err != nil {
		return Snapshot{}, fmt.Errorf("handoff snapshot provenance: %w", err)
	}
	if len(s.Tokens) == 0 {
		return Snapshot{}, fmt.Errorf("handoff snapshot: at least one token is required")
	}

	normalized := Snapshot{
		SchemaVersion: SnapshotSchemaVersion,
		Profile:       profile,
		Provenance:    provenance,
		Tokens:        make([]Token, 0, len(s.Tokens)),
		Metadata:      maps.Clone(s.Metadata),
	}
	seen := make(map[string]struct{}, len(s.Tokens))
	for index, token := range s.Tokens {
		candidate, err := normalizeToken(token)
		if err != nil {
			return Snapshot{}, fmt.Errorf("handoff snapshot token[%d]: %w", index, err)
		}
		key := coordinate(candidate)
		if _, duplicate := seen[key]; duplicate {
			return Snapshot{}, fmt.Errorf("handoff snapshot: duplicate token coordinate %q", key)
		}
		seen[key] = struct{}{}
		normalized.Tokens = append(normalized.Tokens, candidate)
	}
	slices.SortFunc(normalized.Tokens, compareTokens)
	modeRoots := map[string]*pathNode{}
	for _, token := range normalized.Tokens {
		root := modeRoots[token.Mode]
		if root == nil {
			root = &pathNode{}
			modeRoots[token.Mode] = root
		}
		segments, _ := PathSegments(token.Path)
		if ancestor, descendant := root.insert(segments, token.Path); ancestor != "" {
			return Snapshot{}, fmt.Errorf(
				"handoff snapshot mode %q: token path %q conflicts with descendant token %q",
				token.Mode,
				ancestor,
				descendant,
			)
		}
	}
	return normalized, nil
}

// Digest returns the SHA-256 digest of canonical snapshot content. Provenance
// timestamps are omitted so the same source graph always has the same digest.
func (s Snapshot) Digest() (string, error) {
	normalized, err := s.Normalize()
	if err != nil {
		return "", err
	}
	content := struct {
		SchemaVersion string            `json:"schemaVersion"`
		Profile       Profile           `json:"profile"`
		Tokens        []Token           `json:"tokens"`
		Metadata      map[string]string `json:"metadata,omitempty"`
	}{
		SchemaVersion: normalized.SchemaVersion,
		Profile:       normalized.Profile,
		Tokens:        normalized.Tokens,
		Metadata:      normalized.Metadata,
	}
	raw, err := json.Marshal(content)
	if err != nil {
		return "", fmt.Errorf("handoff snapshot digest: %w", err)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// Validate checks the complete change-set transaction.
func (c ChangeSet) Validate() error {
	if c.SchemaVersion != ChangeSetSchemaVersion {
		return fmt.Errorf(
			"handoff changes: unsupported schemaVersion %q (want %q)",
			c.SchemaVersion,
			ChangeSetSchemaVersion,
		)
	}
	if _, err := normalizeProfile(c.Profile); err != nil {
		return fmt.Errorf("handoff changes profile: %w", err)
	}
	if !validDigest(c.BaseDigest) {
		return fmt.Errorf("handoff changes: baseDigest must be a sha256 digest")
	}
	if _, err := normalizeProvenance(c.Provenance); err != nil {
		return fmt.Errorf("handoff changes provenance: %w", err)
	}
	if len(c.Changes) == 0 {
		return fmt.Errorf("handoff changes: at least one change is required")
	}
	seen := make(map[string]struct{}, len(c.Changes))
	for index, change := range c.Changes {
		key, err := validateChange(change)
		if err != nil {
			return fmt.Errorf("handoff changes change[%d]: %w", index, err)
		}
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("handoff changes: duplicate token coordinate %q", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func normalizeProfile(profile Profile) (Profile, error) {
	profile.ID = strings.TrimSpace(profile.ID)
	profile.Version = strings.TrimSpace(profile.Version)
	profile.ClientID = strings.TrimSpace(profile.ClientID)
	profile.ParentID = strings.TrimSpace(profile.ParentID)
	profile.ParentDigest = strings.TrimSpace(profile.ParentDigest)
	if !profileIDPattern.MatchString(profile.ID) {
		return Profile{}, fmt.Errorf("id %q is not canonical", profile.ID)
	}
	if profile.Version == "" {
		return Profile{}, fmt.Errorf("version is required")
	}
	switch profile.Kind {
	case ProfileOSS:
		if profile.ClientID != "" || profile.ParentID != "" || profile.ParentDigest != "" {
			return Profile{}, fmt.Errorf("OSS profile %q cannot declare client or parent fields", profile.ID)
		}
	case ProfilePro:
		if profile.ClientID != "" {
			return Profile{}, fmt.Errorf("pro profile %q cannot declare clientId", profile.ID)
		}
		if profile.ParentID == "" || !validDigest(profile.ParentDigest) {
			return Profile{}, fmt.Errorf("pro profile %q requires parentId and parentDigest", profile.ID)
		}
	case ProfileClient:
		if !profileIDPattern.MatchString(profile.ClientID) {
			return Profile{}, fmt.Errorf("client profile %q requires canonical clientId", profile.ID)
		}
		if profile.ParentID == "" || !validDigest(profile.ParentDigest) {
			return Profile{}, fmt.Errorf("client profile %q requires parentId and parentDigest", profile.ID)
		}
	default:
		return Profile{}, fmt.Errorf("profile %q has unsupported kind %q", profile.ID, profile.Kind)
	}
	return profile, nil
}

func normalizeProvenance(provenance Provenance) (Provenance, error) {
	provenance.Source = strings.TrimSpace(provenance.Source)
	provenance.Author = strings.TrimSpace(provenance.Author)
	provenance.Reason = strings.TrimSpace(provenance.Reason)
	if provenance.Source == "" {
		return Provenance{}, fmt.Errorf("source is required")
	}
	if provenance.GeneratedAt.IsZero() {
		return Provenance{}, fmt.Errorf("generatedAt is required")
	}
	return provenance, nil
}

func normalizeToken(token Token) (Token, error) {
	token.Path = strings.TrimSpace(token.Path)
	token.Mode = strings.TrimSpace(token.Mode)
	token.Description = strings.TrimSpace(token.Description)
	token.Origin.Owner = strings.TrimSpace(token.Origin.Owner)
	token.Origin.Layer = strings.TrimSpace(token.Origin.Layer)
	token.Origin.SourceURI = strings.TrimSpace(token.Origin.SourceURI)
	token.Origin.Pointer = strings.TrimSpace(token.Origin.Pointer)
	if token.Mode == "" {
		token.Mode = DefaultMode
	}
	if _, err := PathSegments(token.Path); err != nil {
		return Token{}, fmt.Errorf("path: %w", err)
	}
	if token.Value == nil {
		return Token{}, fmt.Errorf("token %q/%q value is required", token.Mode, token.Path)
	}
	if token.Origin.Owner == "" || token.Origin.Layer == "" || token.Origin.SourceURI == "" {
		return Token{}, fmt.Errorf("token %q/%q origin requires owner, layer, and sourceUri", token.Mode, token.Path)
	}
	if token.Origin.Pointer != "" {
		if _, err := PathSegments(token.Origin.Pointer); err != nil {
			return Token{}, fmt.Errorf("token %q/%q origin pointer: %w", token.Mode, token.Path, err)
		}
	}
	single := tokens.Set{
		Name:   "handoff",
		Values: map[string]tokens.Value{"token": token.Value},
	}
	if token.Type != "" {
		single.Types = map[string]tokens.Type{"token": token.Type}
	}
	validated, err := single.Normalize()
	if err != nil {
		return Token{}, fmt.Errorf("token %q/%q: %w", token.Mode, token.Path, err)
	}
	token.Value = tokens.CopyValue(validated.Values["token"])
	token.Type = validated.Types["token"]
	return token, nil
}

func validateChange(change Change) (string, error) {
	switch change.Operation {
	case OperationAdd:
		if change.Before != nil || change.After == nil {
			return "", fmt.Errorf("add requires after and forbids before")
		}
	case OperationUpdate:
		if change.Before == nil || change.After == nil {
			return "", fmt.Errorf("update requires before and after")
		}
	case OperationRemove:
		if change.Before == nil || change.After != nil {
			return "", fmt.Errorf("remove requires before and forbids after")
		}
	default:
		return "", fmt.Errorf("unsupported operation %q", change.Operation)
	}
	var before, after Token
	var err error
	if change.Before != nil {
		before, err = normalizeToken(*change.Before)
		if err != nil {
			return "", fmt.Errorf("before: %w", err)
		}
	}
	if change.After != nil {
		after, err = normalizeToken(*change.After)
		if err != nil {
			return "", fmt.Errorf("after: %w", err)
		}
	}
	if change.Before != nil && change.After != nil {
		if coordinate(before) != coordinate(after) {
			return "", fmt.Errorf("before and after coordinates differ")
		}
		if !sameOrigin(before.Origin, after.Origin) {
			return "", fmt.Errorf("update cannot change token origin")
		}
	}
	token := after
	if change.After == nil {
		token = before
	}
	if !token.Origin.Writable {
		return "", fmt.Errorf("token %q is read-only", coordinate(token))
	}
	return coordinate(token), nil
}

func compareTokens(left, right Token) int {
	if left.Mode < right.Mode {
		return -1
	}
	if left.Mode > right.Mode {
		return 1
	}
	if left.Path < right.Path {
		return -1
	}
	if left.Path > right.Path {
		return 1
	}
	return 0
}

func coordinate(token Token) string {
	return token.Mode + ":" + token.Path
}

func sameOrigin(left, right Origin) bool {
	return left == right
}

func sameProfile(left, right Profile) bool {
	return left == right
}

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

// PathFromSegments encodes exact DTCG group and token names as a canonical RFC
// 6901 JSON Pointer. Empty paths and empty segments are rejected because a
// managed token must have a concrete, addressable name.
func PathFromSegments(segments ...string) (string, error) {
	if len(segments) == 0 {
		return "", fmt.Errorf("an absolute token pointer requires at least one segment")
	}
	encoded := make([]string, len(segments))
	for index, segment := range segments {
		if segment == "" {
			return "", fmt.Errorf("token pointer segment[%d] is empty", index)
		}
		encoded[index] = strings.ReplaceAll(
			strings.ReplaceAll(segment, "~", "~0"),
			"/",
			"~1",
		)
	}
	return "/" + strings.Join(encoded, "/"), nil
}

// PathSegments decodes and validates a canonical absolute RFC 6901 JSON
// Pointer. It returns newly allocated exact DTCG path segments.
func PathSegments(pointer string) ([]string, error) {
	if pointer == "" || !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("token path %q is not an absolute JSON pointer", pointer)
	}
	raw := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	segments := make([]string, len(raw))
	for index, segment := range raw {
		var decoded strings.Builder
		for offset := 0; offset < len(segment); offset++ {
			if segment[offset] != '~' {
				decoded.WriteByte(segment[offset])
				continue
			}
			if offset+1 >= len(segment) {
				return nil, fmt.Errorf("token path %q has an incomplete escape", pointer)
			}
			offset++
			switch segment[offset] {
			case '0':
				decoded.WriteByte('~')
			case '1':
				decoded.WriteByte('/')
			default:
				return nil, fmt.Errorf("token path %q has invalid escape ~%c", pointer, segment[offset])
			}
		}
		if decoded.Len() == 0 {
			return nil, fmt.Errorf("token path %q contains an empty segment", pointer)
		}
		segments[index] = decoded.String()
	}
	canonical, err := PathFromSegments(segments...)
	if err != nil {
		return nil, err
	}
	if canonical != pointer {
		return nil, fmt.Errorf("token path %q is not canonical (want %q)", pointer, canonical)
	}
	return segments, nil
}

type pathNode struct {
	token    string
	children map[string]*pathNode
}

func (n *pathNode) insert(segments []string, pointer string) (ancestor, descendant string) {
	cursor := n
	for _, segment := range segments {
		if cursor.token != "" {
			return cursor.token, pointer
		}
		if cursor.children == nil {
			cursor.children = map[string]*pathNode{}
		}
		next := cursor.children[segment]
		if next == nil {
			next = &pathNode{}
			cursor.children[segment] = next
		}
		cursor = next
	}
	if cursor.token != "" {
		return cursor.token, pointer
	}
	if len(cursor.children) > 0 {
		return pointer, firstToken(cursor)
	}
	cursor.token = pointer
	return "", ""
}

func firstToken(node *pathNode) string {
	if node.token != "" {
		return node.token
	}
	keys := slices.Sorted(maps.Keys(node.children))
	for _, key := range keys {
		if token := firstToken(node.children[key]); token != "" {
			return token
		}
	}
	return ""
}
