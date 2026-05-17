// Package tokens provides provider-neutral, DTCG-native design-token documents.
package tokens

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

const (
	TypeBorder      Type = "border"
	TypeColor       Type = "color"
	TypeCubicBezier Type = "cubicBezier"
	TypeDimension   Type = "dimension"
	TypeDuration    Type = "duration"
	TypeFontFamily  Type = "fontFamily"
	TypeFontWeight  Type = "fontWeight"
	TypeGradient    Type = "gradient"
	TypeNumber      Type = "number"
	TypeShadow      Type = "shadow"
	TypeString      Type = "string"
	TypeStrokeStyle Type = "strokeStyle"
	TypeTransition  Type = "transition"
	TypeTypography  Type = "typography"
)

const rootSegment = "$root"

// Severity describes validation issue impact.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// IssueCode is stable enough for tooling and tests to match without parsing
// human-readable messages.
type IssueCode string

const (
	IssueEmptySet            IssueCode = "PKD001_EMPTY_SET"
	IssueInvalidName         IssueCode = "PKD002_INVALID_NAME"
	IssueInvalidPath         IssueCode = "PKD003_INVALID_PATH"
	IssueEmptyValue          IssueCode = "PKD004_EMPTY_VALUE"
	IssueDuplicatePath       IssueCode = "PKD005_DUPLICATE_PATH"
	IssueUnknownMetadataPath IssueCode = "PKD006_UNKNOWN_METADATA_PATH"
	IssueInvalidType         IssueCode = "PKD007_INVALID_TYPE"
	IssuePathConflict        IssueCode = "PKD008_PATH_CONFLICT"
	IssueInvalidReference    IssueCode = "PKD009_INVALID_REFERENCE"
	IssueMissingReference    IssueCode = "PKD010_MISSING_REFERENCE"
	IssueReferenceCycle      IssueCode = "PKD011_REFERENCE_CYCLE"
	IssueInvalidGroup        IssueCode = "PKD012_INVALID_GROUP"
	IssueInvalidDTCG         IssueCode = "PKD013_INVALID_DTCG"
	IssueUnrenderableCSS     IssueCode = "PKD014_UNRENDERABLE_CSS"
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
	Deprecated  any            `json:"deprecated,omitempty"`
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
	Deprecated   map[string]any
	Groups       map[string]Group
	Metadata     map[string]any
}
