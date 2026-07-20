// Package tokens provides provider-neutral, DTCG-native design-token documents.
package tokens

// Implements: REQ-011.
// Per: ADR-0004.
// Discipline: C-14.
// model.go owns the public DTCG token data model and stable validation issue
// vocabulary for pk-design.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

// Value is a lossless DTCG token value. It may be a scalar, an alias string such
// as "{color.brand.primary}", or a JSON-like composite value.
type Value = any

// Type is a DTCG-compatible token type. Unknown platform-specific semantics
// should use an x.* extension token instead of inventing new core vocabulary.
type Type string

// DTCG core token types. These mirror the Design Tokens Community Group
// specification; use an x.* extension token for any semantics not covered here.
const (
	// TypeBorder is a composite border token (color, width, and style).
	TypeBorder Type = "border"
	// TypeColor is a color token.
	TypeColor Type = "color"
	// TypeCubicBezier is a cubic-bezier easing token.
	TypeCubicBezier Type = "cubicBezier"
	// TypeDimension is a length/size token such as spacing or radius.
	TypeDimension Type = "dimension"
	// TypeDuration is a time-duration token.
	TypeDuration Type = "duration"
	// TypeFontFamily is a font-family token.
	TypeFontFamily Type = "fontFamily"
	// TypeFontWeight is a font-weight token.
	TypeFontWeight Type = "fontWeight"
	// TypeGradient is a gradient token.
	TypeGradient Type = "gradient"
	// TypeNumber is a unitless number token.
	TypeNumber Type = "number"
	// TypeShadow is a shadow token (single or composite).
	TypeShadow Type = "shadow"
	// TypeString is a free-form string token.
	TypeString Type = "string"
	// TypeStrokeStyle is a stroke-style token.
	TypeStrokeStyle Type = "strokeStyle"
	// TypeTransition is a composite transition token.
	TypeTransition Type = "transition"
	// TypeTypography is a composite typography token.
	TypeTypography Type = "typography"
)

const rootSegment = "$root"

// Severity describes validation issue impact.
type Severity string

// Validation severities ordered from most to least impactful.
const (
	// SeverityError marks an issue that makes the set invalid.
	SeverityError Severity = "error"
	// SeverityWarning marks a non-fatal issue that callers may choose to ignore.
	SeverityWarning Severity = "warning"
)

// IssueCode is stable enough for tooling and tests to match without parsing
// human-readable messages.
type IssueCode string

// Stable issue codes emitted by validation. Codes are append-only so tooling
// and tests can match on them without parsing human-readable messages.
const (
	// IssueEmptySet reports a set with no tokens.
	IssueEmptySet IssueCode = "PKD001_EMPTY_SET"
	// IssueInvalidName reports an invalid set name.
	IssueInvalidName IssueCode = "PKD002_INVALID_NAME"
	// IssueInvalidPath reports a malformed token path.
	IssueInvalidPath IssueCode = "PKD003_INVALID_PATH"
	// IssueEmptyValue reports a token that has no value.
	IssueEmptyValue IssueCode = "PKD004_EMPTY_VALUE"
	// IssueDuplicatePath reports two tokens sharing the same path.
	IssueDuplicatePath IssueCode = "PKD005_DUPLICATE_PATH"
	// IssueUnknownMetadataPath reports metadata keyed to a non-existent path.
	IssueUnknownMetadataPath IssueCode = "PKD006_UNKNOWN_METADATA_PATH"
	// IssueInvalidType reports a token type outside the DTCG vocabulary.
	IssueInvalidType IssueCode = "PKD007_INVALID_TYPE"
	// IssuePathConflict reports a token path that collides with a group path.
	IssuePathConflict IssueCode = "PKD008_PATH_CONFLICT"
	// IssueInvalidReference reports a malformed alias reference.
	IssueInvalidReference IssueCode = "PKD009_INVALID_REFERENCE"
	// IssueMissingReference reports an alias that points at an unknown token.
	IssueMissingReference IssueCode = "PKD010_MISSING_REFERENCE"
	// IssueReferenceCycle reports a cycle among alias references.
	IssueReferenceCycle IssueCode = "PKD011_REFERENCE_CYCLE"
	// IssueInvalidGroup reports invalid group-level metadata.
	IssueInvalidGroup IssueCode = "PKD012_INVALID_GROUP"
	// IssueInvalidDTCG reports a document that is not valid DTCG.
	IssueInvalidDTCG IssueCode = "PKD013_INVALID_DTCG"
	// IssueUnrenderableCSS reports a token that cannot be rendered to CSS.
	IssueUnrenderableCSS IssueCode = "PKD014_UNRENDERABLE_CSS"
)

// Issue is one machine-readable validation finding.
type Issue struct {
	Code     IssueCode `json:"code"`
	Severity Severity  `json:"severity"`
	Path     string    `json:"path,omitempty"`
	Message  string    `json:"message"`
}

// Report is a deterministic validation result. A report with errors implements
// error so callers can return it directly while retaining issue details.
type Report struct {
	Issues []Issue `json:"issues,omitempty"`
}

// Token is the normalized public view of one design token.
type Token struct {
	Path        string         `json:"path"`
	Type        Type           `json:"type,omitempty"`
	Value       Value          `json:"value"`
	Description string         `json:"description,omitempty"`
	Extensions  map[string]any `json:"extensions,omitempty"`
}

// Group describes DTCG group-level metadata.
type Group struct {
	Type        Type           `json:"type,omitempty"`
	Description string         `json:"description,omitempty"`
	Extends     string         `json:"extends,omitempty"`
	Extensions  map[string]any `json:"extensions,omitempty"`
}

// Set is a named group of DTCG design tokens. Values is a flat map keyed by
// DTCG dotted token path. Root tokens use the reserved "$root" path segment,
// e.g. "color.brand.$root".
type Set struct {
	Name         string
	Version      string
	Values       map[string]Value
	Types        map[string]Type
	Descriptions map[string]string
	Extensions   map[string]map[string]any
	Groups       map[string]Group
	Metadata     map[string]any
}
