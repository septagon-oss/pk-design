package catalog

// version.go owns semantic-version validation for contribution manifests. The
// catalog keeps compatibility metadata strict enough for tooling without
// depending on an external versioning library.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"fmt"
	"strconv"
	"strings"
)

type semanticVersion struct {
	major      int
	minor      int
	patch      int
	prerelease []string
}

func normalizeSemverField(source, field, value string) (string, semanticVersion, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", semanticVersion{}, false, nil
	}
	parsed, ok := parseSemanticVersion(value)
	if !ok {
		return "", semanticVersion{}, false, fmt.Errorf("design contribution %q manifest %s %q must be a semantic version", source, field, value)
	}
	return value, parsed, true, nil
}

func parseSemanticVersion(value string) (semanticVersion, bool) {
	value = strings.TrimSpace(value)
	value, _ = strings.CutPrefix(value, "v")
	if value == "" {
		return semanticVersion{}, false
	}
	coreAndBuild := strings.SplitN(value, "+", 2)
	if len(coreAndBuild) == 2 && !validSemverIdentifiers(coreAndBuild[1], true) {
		return semanticVersion{}, false
	}
	coreAndPre := strings.SplitN(coreAndBuild[0], "-", 2)
	if len(coreAndPre) == 2 && !validSemverIdentifiers(coreAndPre[1], false) {
		return semanticVersion{}, false
	}
	core := strings.Split(coreAndPre[0], ".")
	if len(core) != 3 {
		return semanticVersion{}, false
	}
	major, ok := parseSemverNumber(core[0])
	if !ok {
		return semanticVersion{}, false
	}
	minor, ok := parseSemverNumber(core[1])
	if !ok {
		return semanticVersion{}, false
	}
	patch, ok := parseSemverNumber(core[2])
	if !ok {
		return semanticVersion{}, false
	}
	var prerelease []string
	if len(coreAndPre) == 2 {
		prerelease = strings.Split(coreAndPre[1], ".")
	}
	return semanticVersion{major: major, minor: minor, patch: patch, prerelease: prerelease}, true
}

func parseSemverNumber(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	if len(value) > 1 && value[0] == '0' {
		return 0, false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	parsed, err := strconv.Atoi(value)
	return parsed, err == nil
}

func validSemverIdentifiers(value string, allowNumericLeadingZero bool) bool {
	if value == "" {
		return false
	}
	for _, part := range strings.Split(value, ".") {
		if part == "" {
			return false
		}
		numeric := true
		for _, r := range part {
			switch {
			case r >= '0' && r <= '9':
			case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '-':
				numeric = false
			default:
				return false
			}
		}
		if numeric && !allowNumericLeadingZero && len(part) > 1 && part[0] == '0' {
			return false
		}
	}
	return true
}

func compareSemanticVersions(a, b semanticVersion) int {
	if delta := a.major - b.major; delta != 0 {
		return sign(delta)
	}
	if delta := a.minor - b.minor; delta != 0 {
		return sign(delta)
	}
	if delta := a.patch - b.patch; delta != 0 {
		return sign(delta)
	}
	return comparePrerelease(a.prerelease, b.prerelease)
}

func comparePrerelease(a, b []string) int {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	if len(a) == 0 {
		return 1
	}
	if len(b) == 0 {
		return -1
	}
	for i := 0; i < len(a) && i < len(b); i++ {
		aNumber, aNumeric := parsePrereleaseNumber(a[i])
		bNumber, bNumeric := parsePrereleaseNumber(b[i])
		switch {
		case aNumeric && bNumeric:
			if aNumber != bNumber {
				return sign(aNumber - bNumber)
			}
		case aNumeric:
			return -1
		case bNumeric:
			return 1
		case a[i] != b[i]:
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	return sign(len(a) - len(b))
}

func parsePrereleaseNumber(value string) (int, bool) {
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	parsed, err := strconv.Atoi(value)
	return parsed, err == nil
}

func sign(value int) int {
	switch {
	case value < 0:
		return -1
	case value > 0:
		return 1
	default:
		return 0
	}
}
