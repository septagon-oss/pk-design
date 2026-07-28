package figma

// Implements: REQ-011.
// Per: ADR-0076.
// Discipline: C-14.
// validate.go fail-closes the portable Figma Variables contract before a
// product delivery is allowed to embed or execute it.
//
// ADR: ADR-0029 (file purpose declaration), ADR-0076 (layered design delivery).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"fmt"
	"strings"

	"github.com/septagon-oss/pk-design/pkg/handoff"
)

var supportedVariableTypes = map[string]struct{}{
	"BOOLEAN": {},
	"COLOR":   {},
	"FLOAT":   {},
	"STRING":  {},
}

// Validate verifies the complete governed Figma delivery, including the
// hidden source snapshot, its digest, native collection/mode coordinates, and
// every alias target. It is intentionally exported so Pro and client delivery
// assemblers can depend on the OSS protocol without reimplementing it.
func Validate(bundle Bundle) error {
	if bundle.Kind != BundleKind {
		return fmt.Errorf("figma bundle: unexpected kind %q", bundle.Kind)
	}
	if bundle.Schema != BundleSchemaVersion {
		return fmt.Errorf("figma bundle: unsupported schema %q", bundle.Schema)
	}
	if strings.TrimSpace(bundle.Product) == "" {
		return fmt.Errorf("figma bundle: product is required")
	}

	normalized, err := bundle.Snapshot.Normalize()
	if err != nil {
		return fmt.Errorf("figma bundle: snapshot: %w", err)
	}
	digest, err := normalized.Digest()
	if err != nil {
		return fmt.Errorf("figma bundle: digest snapshot: %w", err)
	}
	if bundle.SnapshotDigest != digest {
		return fmt.Errorf(
			"figma bundle: snapshotDigest %q does not match snapshot %q",
			bundle.SnapshotDigest,
			digest,
		)
	}
	clientID := strings.TrimSpace(bundle.ClientID)
	if normalized.Profile.Kind == handoff.ProfileClient {
		if clientID != normalized.Profile.ClientID {
			return fmt.Errorf(
				"figma bundle: clientId %q does not match snapshot profile %q",
				clientID,
				normalized.Profile.ClientID,
			)
		}
	} else if clientID != "" {
		return fmt.Errorf("figma bundle: clientId is only valid for a client profile")
	}
	if len(bundle.Collections) == 0 {
		return fmt.Errorf("figma bundle: at least one collection is required")
	}

	snapshotCoordinates := make(map[string]handoff.Token, len(normalized.Tokens))
	snapshotPaths := make(map[string]struct{}, len(normalized.Tokens))
	for _, token := range normalized.Tokens {
		snapshotCoordinates[tokenCoordinate(token.Path, token.Mode)] = token
		snapshotPaths[token.Path] = struct{}{}
	}

	collectionNames := make(map[string]struct{}, len(bundle.Collections))
	variableKeys := make(map[string]Variable)
	for collectionIndex, collection := range bundle.Collections {
		collectionName := strings.TrimSpace(collection.Name)
		if collectionName == "" {
			return fmt.Errorf("figma bundle: collection %d name is required", collectionIndex)
		}
		if _, duplicate := collectionNames[collectionName]; duplicate {
			return fmt.Errorf("figma bundle: duplicate collection %q", collectionName)
		}
		collectionNames[collectionName] = struct{}{}
		if len(collection.Modes) == 0 {
			return fmt.Errorf("figma bundle: collection %q requires at least one mode", collectionName)
		}
		modeNames := make(map[string]struct{}, len(collection.Modes))
		for _, mode := range collection.Modes {
			mode = strings.TrimSpace(mode)
			if mode == "" {
				return fmt.Errorf("figma bundle: collection %q has an empty mode", collectionName)
			}
			if _, duplicate := modeNames[mode]; duplicate {
				return fmt.Errorf("figma bundle: collection %q has duplicate mode %q", collectionName, mode)
			}
			modeNames[mode] = struct{}{}
		}

		variableNames := make(map[string]struct{}, len(collection.Variables))
		for variableIndex, variable := range collection.Variables {
			if err := validateVariable(
				collectionName,
				variableIndex,
				variable,
				modeNames,
				snapshotCoordinates,
			); err != nil {
				return err
			}
			if _, duplicate := variableNames[variable.Name]; duplicate {
				return fmt.Errorf(
					"figma bundle: collection %q has duplicate variable name %q",
					collectionName,
					variable.Name,
				)
			}
			variableNames[variable.Name] = struct{}{}
			if _, duplicate := variableKeys[variable.Key]; duplicate {
				return fmt.Errorf("figma bundle: duplicate variable key %q", variable.Key)
			}
			variableKeys[variable.Key] = variable
		}
	}

	for path := range snapshotPaths {
		if _, exists := variableKeys[path]; !exists {
			return fmt.Errorf("figma bundle: snapshot token path %q has no native variable", path)
		}
	}
	for _, variable := range variableKeys {
		for modeName, targetKey := range variable.Aliases {
			if _, exists := variableKeys[targetKey]; !exists {
				return fmt.Errorf(
					"figma bundle: variable %q mode %q aliases missing variable %q",
					variable.Key,
					modeName,
					targetKey,
				)
			}
		}
	}
	return nil
}

func validateVariable(
	collectionName string,
	variableIndex int,
	variable Variable,
	collectionModes map[string]struct{},
	snapshotCoordinates map[string]handoff.Token,
) error {
	label := fmt.Sprintf("collection %q variable %d", collectionName, variableIndex)
	if strings.TrimSpace(variable.Key) == "" {
		return fmt.Errorf("figma bundle: %s key is required", label)
	}
	if _, err := handoff.PathSegments(variable.Key); err != nil {
		return fmt.Errorf("figma bundle: %s key: %w", label, err)
	}
	if strings.TrimSpace(variable.Name) == "" {
		return fmt.Errorf("figma bundle: %s name is required", label)
	}
	if _, supported := supportedVariableTypes[variable.Type]; !supported {
		return fmt.Errorf("figma bundle: variable %q has unsupported type %q", variable.Key, variable.Type)
	}
	if variable.Binding.TokenPath != variable.Key {
		return fmt.Errorf(
			"figma bundle: variable %q binding tokenPath %q does not match its key",
			variable.Key,
			variable.Binding.TokenPath,
		)
	}
	if len(variable.Binding.ModeMap) == 0 {
		return fmt.Errorf("figma bundle: variable %q requires a mode binding", variable.Key)
	}

	for displayMode, snapshotMode := range variable.Binding.ModeMap {
		if _, exists := collectionModes[displayMode]; !exists {
			return fmt.Errorf(
				"figma bundle: variable %q references undeclared Figma mode %q",
				variable.Key,
				displayMode,
			)
		}
		token, exists := snapshotCoordinates[tokenCoordinate(variable.Key, snapshotMode)]
		if !exists {
			return fmt.Errorf(
				"figma bundle: variable %q mode %q references missing snapshot mode %q",
				variable.Key,
				displayMode,
				snapshotMode,
			)
		}
		// A provider adapter may safely downgrade a writable source token when
		// Figma has no lossless native codec (for example a composite shadow).
		// It may never upgrade a read-only source token to writable.
		if variable.Binding.Writable && !token.Origin.Writable {
			return fmt.Errorf(
				"figma bundle: variable %q makes read-only snapshot mode %q writable",
				variable.Key,
				snapshotMode,
			)
		}
		_, hasValue := variable.Values[displayMode]
		_, hasAlias := variable.Aliases[displayMode]
		if hasValue == hasAlias {
			return fmt.Errorf(
				"figma bundle: variable %q mode %q must define exactly one value or alias",
				variable.Key,
				displayMode,
			)
		}
	}
	for mode := range variable.Values {
		if _, exists := variable.Binding.ModeMap[mode]; !exists {
			return fmt.Errorf("figma bundle: variable %q has an unbound value mode %q", variable.Key, mode)
		}
	}
	for mode := range variable.Aliases {
		if _, exists := variable.Binding.ModeMap[mode]; !exists {
			return fmt.Errorf("figma bundle: variable %q has an unbound alias mode %q", variable.Key, mode)
		}
	}
	return nil
}

func tokenCoordinate(path, mode string) string {
	return mode + "\x00" + path
}
