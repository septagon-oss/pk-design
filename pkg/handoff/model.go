// Package handoff defines provider-neutral, optimistic-concurrency contracts
// for round-tripping design-token changes through external design tools.
package handoff

// Implements: REQ-011.
// Per: ADR-0076.
// Discipline: C-14.
// model.go owns the public snapshot, profile, provenance, ownership, and
// change-set vocabulary. Provider adapters translate their native model into
// these contracts; they do not define a second mutation protocol.
//
// ADR: ADR-0029 (file purpose declaration), ADR-0076 (layered design delivery).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"fmt"
	"time"

	"github.com/septagon-oss/pk-design/pkg/tokens"
)

const (
	// SnapshotSchemaVersion is the stable schema for an exported token snapshot.
	SnapshotSchemaVersion = "pk.design.token-snapshot.v1"
	// ChangeSetSchemaVersion is the stable schema for a proposed token mutation.
	ChangeSetSchemaVersion = "pk.design.token-changes.v1"
	// DefaultMode is the canonical mode for mode-independent tokens.
	DefaultMode = "default"
)

// ProfileKind identifies the extension scope that owns a delivery.
type ProfileKind string

const (
	// ProfileOSS is a public, standalone OSS design profile.
	ProfileOSS ProfileKind = "oss"
	// ProfilePro extends an OSS profile with private product contracts.
	ProfilePro ProfileKind = "pro"
	// ProfileClient extends a parent profile with client-owned design state.
	ProfileClient ProfileKind = "client"
)

// Profile identifies one layer in the design-delivery graph. Pro and client
// profiles pin the exact digest of the parent they extend.
type Profile struct {
	ID           string      `json:"id"`
	Kind         ProfileKind `json:"kind"`
	Version      string      `json:"version"`
	ClientID     string      `json:"clientId,omitempty"`
	ParentID     string      `json:"parentId,omitempty"`
	ParentDigest string      `json:"parentDigest,omitempty"`
}

// Provenance records who produced a snapshot or proposed a change and why.
// GeneratedAt is evidence only and is deliberately excluded from content
// digests so rebuilding identical design state remains reproducible.
type Provenance struct {
	Source      string    `json:"source"`
	GeneratedAt time.Time `json:"generatedAt"`
	Author      string    `json:"author,omitempty"`
	Reason      string    `json:"reason,omitempty"`
}

// Origin identifies the code-owned layer and source location for a token.
// Adapters may expose read-only resolved tokens, but only Writable origins may
// be changed by Diff or Apply.
type Origin struct {
	Owner     string `json:"owner"`
	Layer     string `json:"layer"`
	SourceURI string `json:"sourceUri"`
	Pointer   string `json:"pointer,omitempty"`
	Writable  bool   `json:"writable"`
}

// Token is one token value in one mode together with immutable code ownership.
// Path is an RFC 6901 JSON Pointer whose segments are the exact DTCG group and
// token names. A pointer is used instead of dotted notation because DTCG names
// may themselves contain dots or slashes. Value is lossless DTCG-compatible
// JSON data.
type Token struct {
	Path        string       `json:"path"`
	Mode        string       `json:"mode"`
	Type        tokens.Type  `json:"type,omitempty"`
	Value       tokens.Value `json:"value"`
	Description string       `json:"description,omitempty"`
	Origin      Origin       `json:"origin"`
}

// Snapshot is the complete managed-token state imported into an external
// design tool. Tokens are normalized into deterministic coordinate order.
type Snapshot struct {
	SchemaVersion string            `json:"schemaVersion"`
	Profile       Profile           `json:"profile"`
	Provenance    Provenance        `json:"provenance"`
	Tokens        []Token           `json:"tokens"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// Operation is the explicit mutation performed by one change.
type Operation string

const (
	// OperationAdd introduces a token at a previously unused coordinate.
	OperationAdd Operation = "add"
	// OperationUpdate changes the value, type, or description of a token.
	OperationUpdate Operation = "update"
	// OperationRemove deletes a token from its owning layer.
	OperationRemove Operation = "remove"
)

// Change describes one atomic before/after transition. Before and After carry
// full tokens so receivers can verify both the expected value and ownership.
type Change struct {
	Operation Operation `json:"operation"`
	Before    *Token    `json:"before,omitempty"`
	After     *Token    `json:"after,omitempty"`
}

// ChangeSet is a proposed transaction derived from one exact snapshot. A
// receiver applies every change or none.
type ChangeSet struct {
	SchemaVersion string     `json:"schemaVersion"`
	Profile       Profile    `json:"profile"`
	BaseDigest    string     `json:"baseDigest"`
	Provenance    Provenance `json:"provenance"`
	Changes       []Change   `json:"changes"`
}

// ConflictCode is a stable machine-readable reason an apply operation failed.
type ConflictCode string

const (
	// ConflictBaseDigest means code changed after the external snapshot.
	ConflictBaseDigest ConflictCode = "PKH001_BASE_DIGEST"
	// ConflictTokenMismatch means the expected previous token no longer matches.
	ConflictTokenMismatch ConflictCode = "PKH002_TOKEN_MISMATCH"
	// ConflictTokenExists means an add targeted an occupied coordinate.
	ConflictTokenExists ConflictCode = "PKH003_TOKEN_EXISTS"
	// ConflictTokenMissing means an update or remove targeted a missing token.
	ConflictTokenMissing ConflictCode = "PKH004_TOKEN_MISSING"
	// ConflictOwnership means a change crossed a read-only or foreign origin.
	ConflictOwnership ConflictCode = "PKH005_OWNERSHIP"
	// ConflictProfile means a change set targeted another delivery profile.
	ConflictProfile ConflictCode = "PKH006_PROFILE"
)

// ConflictError is returned when a valid change set cannot safely apply to the
// current snapshot.
type ConflictError struct {
	Code       ConflictCode
	Coordinate string
	Message    string
}

// Error implements error with stable code and optional token coordinate.
func (e ConflictError) Error() string {
	if e.Coordinate == "" {
		return fmt.Sprintf("handoff conflict %s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("handoff conflict %s at %s: %s", e.Code, e.Coordinate, e.Message)
}
