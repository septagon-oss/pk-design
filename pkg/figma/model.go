// Package figma projects governed design-token snapshots into the portable
// native-variable bundle consumed by the PlatformKit Figma plugin.
package figma

// Implements: REQ-011.
// Per: ADR-0076.
// Discipline: C-14.
// model.go owns the OSS Figma transport contract. Product and client layers
// extend this bundle with their own snapshots; they do not fork the protocol.
//
// ADR: ADR-0029 (file purpose declaration), ADR-0076 (layered design delivery).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import "github.com/septagon-oss/pk-design/pkg/handoff"

const (
	// BundleKind identifies a native Figma Variables delivery.
	BundleKind = "figma-variables"
	// BundleSchemaVersion is the round-trip-capable variables contract.
	BundleSchemaVersion = "pk.design.figma-variables.v2"
)

// Bundle is a complete, self-verifying import delivery. Snapshot is retained
// verbatim so the plugin can later export an optimistic-concurrency ChangeSet.
type Bundle struct {
	Kind           string            `json:"kind"`
	Schema         string            `json:"schema"`
	Product        string            `json:"product"`
	ClientID       string            `json:"clientId,omitempty"`
	SnapshotDigest string            `json:"snapshotDigest"`
	Snapshot       handoff.Snapshot  `json:"snapshot"`
	Collections    []Collection      `json:"collections"`
	Diagnostics    []Diagnostic      `json:"diagnostics,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// Collection maps to one local Figma VariableCollection.
type Collection struct {
	Name      string     `json:"name"`
	Modes     []string   `json:"modes"`
	Variables []Variable `json:"variables"`
}

// Variable is one native Figma Variable plus its exact code-owned binding.
// Key is stable across display-name changes and is used by aliases.
type Variable struct {
	Key         string            `json:"key"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	Values      map[string]any    `json:"values,omitempty"`
	Aliases     map[string]string `json:"aliases,omitempty"`
	Binding     Binding           `json:"binding"`
}

// Binding maps Figma collection modes back to exact snapshot coordinates and
// describes how native values round-trip to their authored DTCG representation.
type Binding struct {
	TokenPath string            `json:"tokenPath"`
	ModeMap   map[string]string `json:"modeMap"`
	Codecs    map[string]Codec  `json:"codecs,omitempty"`
	Writable  bool              `json:"writable"`
}

// Codec is a provider-neutral recipe for converting one Figma native value
// back to the exact primitive representation accepted by the source compiler.
type Codec struct {
	Kind      string  `json:"kind"`
	Style     string  `json:"style,omitempty"`
	Unit      string  `json:"unit,omitempty"`
	Scale     float64 `json:"scale,omitempty"`
	Precision int     `json:"precision,omitempty"`
}

// Diagnostic explains a safe degradation such as exposing a composite token
// read-only because Figma has no lossless native variable type for it.
type Diagnostic struct {
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}
