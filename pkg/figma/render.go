package figma

// Implements: REQ-011.
// Per: ADR-0076.
// Discipline: C-14.
// render.go deterministically turns an OSS handoff snapshot into native Figma
// collections, preserves DTCG aliases, and binds every mode back to code.
//
// ADR: ADR-0029 (file purpose declaration), ADR-0076 (layered design delivery).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"

	"github.com/septagon-oss/pk-design/pkg/handoff"
	"github.com/septagon-oss/pk-design/pkg/tokens"
)

const (
	sourceCollectionName   = "PlatformKit Sources"
	resolvedCollectionName = "PlatformKit Colors"
)

// Options supplies display identity without changing snapshot ownership.
type Options struct {
	Product  string
	ClientID string
	Metadata map[string]string
}

// Render builds and validates one deterministic Figma delivery.
func Render(snapshot handoff.Snapshot, options Options) (Bundle, error) {
	normalized, err := snapshot.Normalize()
	if err != nil {
		return Bundle{}, fmt.Errorf("figma bundle: normalize snapshot: %w", err)
	}
	digest, err := normalized.Digest()
	if err != nil {
		return Bundle{}, fmt.Errorf("figma bundle: digest snapshot: %w", err)
	}
	product := strings.TrimSpace(options.Product)
	if product == "" {
		return Bundle{}, fmt.Errorf("figma bundle: product is required")
	}
	clientID := strings.TrimSpace(options.ClientID)
	if normalized.Profile.Kind == handoff.ProfileClient && clientID == "" {
		clientID = normalized.Profile.ClientID
	}
	if normalized.Profile.Kind != handoff.ProfileClient && clientID != "" {
		return Bundle{}, fmt.Errorf("figma bundle: clientId is only valid for a client profile")
	}

	grouped := map[string][]handoff.Token{}
	for _, token := range normalized.Tokens {
		grouped[token.Path] = append(grouped[token.Path], token)
	}
	paths := slices.Sorted(maps.Keys(grouped))
	variables := make(map[string]Variable, len(paths))
	collectionByPath := make(map[string]string, len(paths))
	diagnostics := make([]Diagnostic, 0)
	for _, path := range paths {
		variable, tokenDiagnostics, err := renderVariable(path, grouped[path])
		if err != nil {
			return Bundle{}, err
		}
		variables[path] = variable
		collectionByPath[path] = collectionForTokens(grouped[path])
		diagnostics = append(diagnostics, tokenDiagnostics...)
	}
	if err := validateAliases(variables); err != nil {
		return Bundle{}, err
	}

	collections := map[string]*Collection{
		sourceCollectionName:   {Name: sourceCollectionName},
		resolvedCollectionName: {Name: resolvedCollectionName},
	}
	for _, path := range paths {
		variable := variables[path]
		collection := collections[collectionByPath[path]]
		for modeName := range variable.Binding.ModeMap {
			if !slices.Contains(collection.Modes, modeName) {
				collection.Modes = append(collection.Modes, modeName)
			}
		}
		collection.Variables = append(collection.Variables, variable)
	}
	orderedCollections := make([]Collection, 0, 2)
	for _, name := range []string{sourceCollectionName, resolvedCollectionName} {
		collection := collections[name]
		if len(collection.Variables) == 0 {
			continue
		}
		slices.Sort(collection.Modes)
		slices.SortFunc(collection.Variables, func(left, right Variable) int {
			return strings.Compare(left.Key, right.Key)
		})
		orderedCollections = append(orderedCollections, *collection)
	}
	bundle := Bundle{
		Kind:           BundleKind,
		Schema:         BundleSchemaVersion,
		Product:        product,
		ClientID:       clientID,
		SnapshotDigest: digest,
		Snapshot:       normalized,
		Collections:    orderedCollections,
		Diagnostics:    diagnostics,
		Metadata:       maps.Clone(options.Metadata),
	}
	if err := Validate(bundle); err != nil {
		return Bundle{}, err
	}
	return bundle, nil
}

func collectionForTokens(grouped []handoff.Token) string {
	if len(grouped) == 0 {
		return resolvedCollectionName
	}
	switch grouped[0].Origin.Layer {
	case "semantic-computed", "resolved":
		return resolvedCollectionName
	case "semantic":
		// Authored semantic colors belong beside the resolved color plane.
		// Semantic dimensions, motion, opacity, shadows, and typography remain
		// editable sources and must not add a meaningless Default mode to the
		// Light/Dark Colors collection.
		if grouped[0].Type == tokens.TypeColor {
			return resolvedCollectionName
		}
		return sourceCollectionName
	default:
		return sourceCollectionName
	}
}

// RenderJSON returns a newline-terminated portable bundle.
func RenderJSON(snapshot handoff.Snapshot, options Options) ([]byte, error) {
	bundle, err := Render(snapshot, options)
	if err != nil {
		return nil, err
	}
	raw, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("figma bundle: marshal: %w", err)
	}
	return append(raw, '\n'), nil
}

func renderVariable(path string, grouped []handoff.Token) (Variable, []Diagnostic, error) {
	if len(grouped) == 0 {
		return Variable{}, nil, fmt.Errorf("figma bundle: token %q has no modes", path)
	}
	slices.SortFunc(grouped, func(left, right handoff.Token) int {
		return strings.Compare(left.Mode, right.Mode)
	})
	writable := grouped[0].Origin.Writable
	tokenType := grouped[0].Type
	description := grouped[0].Description
	variable := Variable{
		Key:         path,
		Name:        displayName(path),
		Description: description,
		Values:      map[string]any{},
		Aliases:     map[string]string{},
		Binding: Binding{
			TokenPath: path,
			ModeMap:   map[string]string{},
			Codecs:    map[string]Codec{},
			Writable:  writable,
		},
	}
	var diagnostics []Diagnostic
	for _, token := range grouped {
		if token.Origin.Writable != writable {
			return Variable{}, nil, fmt.Errorf(
				"figma bundle: token %q changes writability across modes",
				path,
			)
		}
		if token.Type != tokenType {
			return Variable{}, nil, fmt.Errorf(
				"figma bundle: token %q changes type across modes",
				path,
			)
		}
		modeName := displayMode(token.Mode)
		if _, duplicate := variable.Binding.ModeMap[modeName]; duplicate {
			return Variable{}, nil, fmt.Errorf(
				"figma bundle: token %q has duplicate display mode %q",
				path,
				modeName,
			)
		}
		variable.Binding.ModeMap[modeName] = token.Mode
		if reference, alias := tokenReference(token.Value); alias {
			target, err := pointerFromDottedReference(reference)
			if err != nil {
				return Variable{}, nil, fmt.Errorf(
					"figma bundle: token %q alias %q: %w",
					path,
					reference,
					err,
				)
			}
			variable.Aliases[modeName] = target
			continue
		}
		figmaType, value, codec, lossless, err := encodeValue(token.Type, token.Value)
		if err != nil {
			return Variable{}, nil, fmt.Errorf(
				"figma bundle: encode token %q mode %q: %w",
				path,
				token.Mode,
				err,
			)
		}
		if variable.Type == "" {
			variable.Type = figmaType
		} else if variable.Type != figmaType {
			return Variable{}, nil, fmt.Errorf(
				"figma bundle: token %q maps to multiple Figma types",
				path,
			)
		}
		variable.Values[modeName] = value
		variable.Binding.Codecs[modeName] = codec
		if writable && !lossless {
			variable.Binding.Writable = false
			diagnostics = append(diagnostics, Diagnostic{
				Code: "PKF001_READ_ONLY_CODEC",
				Path: path,
				Message: fmt.Sprintf(
					"Figma has no lossless writable variable codec for DTCG type %q; exposed read-only",
					token.Type,
				),
			})
		}
	}
	if variable.Type == "" {
		variable.Type = figmaTypeForToken(tokenType)
	}
	if len(variable.Values) == 0 {
		variable.Values = nil
	}
	if len(variable.Aliases) == 0 {
		variable.Aliases = nil
	}
	if len(variable.Binding.Codecs) == 0 {
		variable.Binding.Codecs = nil
	}
	return variable, diagnostics, nil
}

func validateAliases(variables map[string]Variable) error {
	for path, variable := range variables {
		for modeName, target := range variable.Aliases {
			targetVariable, exists := variables[target]
			if !exists {
				return fmt.Errorf(
					"figma bundle: token %q mode %q aliases missing token %q",
					path,
					modeName,
					target,
				)
			}
			targetMode := variable.Binding.ModeMap[modeName]
			if !bindingContainsSnapshotMode(targetVariable.Binding, targetMode) &&
				!bindingContainsSnapshotMode(targetVariable.Binding, handoff.DefaultMode) {
				return fmt.Errorf(
					"figma bundle: token %q mode %q aliases %q without matching or default snapshot mode %q",
					path,
					modeName,
					target,
					targetMode,
				)
			}
		}
	}
	return nil
}

func bindingContainsSnapshotMode(binding Binding, snapshotMode string) bool {
	for _, candidate := range binding.ModeMap {
		if candidate == snapshotMode {
			return true
		}
	}
	return false
}

func pointerFromDottedReference(reference string) (string, error) {
	segments := strings.Split(reference, ".")
	return handoff.PathFromSegments(segments...)
}

func displayName(pointer string) string {
	segments, err := handoff.PathSegments(pointer)
	if err != nil {
		return pointer
	}
	for index, segment := range segments {
		segment = strings.ReplaceAll(segment, "/", "∕")
		segment = strings.Map(func(r rune) rune {
			if unicode.IsControl(r) {
				return -1
			}
			return r
		}, segment)
		segments[index] = segment
	}
	return strings.Join(segments, "/")
}

func displayMode(mode string) string {
	mode = strings.TrimSpace(mode)
	if mode == "" || mode == handoff.DefaultMode {
		return "Default"
	}
	runes := []rune(mode)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func figmaTypeForToken(tokenType tokens.Type) string {
	switch tokenType {
	case tokens.TypeColor:
		return "COLOR"
	case tokens.TypeDimension, tokens.TypeDuration, tokens.TypeNumber, tokens.TypeFontWeight:
		return "FLOAT"
	default:
		if tokenType == tokens.Type("x.boolean") {
			return "BOOLEAN"
		}
		return "STRING"
	}
}

func tokenReference(value any) (string, bool) {
	text, ok := value.(string)
	if !ok {
		return "", false
	}
	return tokens.ParseReference(text)
}
