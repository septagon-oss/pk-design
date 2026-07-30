// Implements: REQ-011.
// Per: ADR-0029.
// Discipline: C-14.

package tokens

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	// WCAGAAContrast is the minimum contrast ratio for normal text under
	// WCAG 2.x success criterion 1.4.3.
	WCAGAAContrast = 4.5
)

// ContrastRatio returns the WCAG relative-luminance contrast ratio for two
// opaque hexadecimal sRGB colors. Three- and six-digit CSS hex forms are
// accepted; alpha-bearing colors are rejected because their effective color
// depends on an additional backdrop.
func ContrastRatio(foreground string, background string) (float64, error) {
	foregroundRGB, err := parseOpaqueHexColor(foreground)
	if err != nil {
		return 0, fmt.Errorf("foreground: %w", err)
	}
	backgroundRGB, err := parseOpaqueHexColor(background)
	if err != nil {
		return 0, fmt.Errorf("background: %w", err)
	}

	foregroundLuminance := relativeLuminance(foregroundRGB)
	backgroundLuminance := relativeLuminance(backgroundRGB)
	lighter := math.Max(foregroundLuminance, backgroundLuminance)
	darker := math.Min(foregroundLuminance, backgroundLuminance)
	return (lighter + 0.05) / (darker + 0.05), nil
}

func parseOpaqueHexColor(value string) ([3]float64, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "#") {
		return [3]float64{}, fmt.Errorf("color %q is not hexadecimal sRGB", value)
	}
	value = strings.TrimPrefix(value, "#")
	switch len(value) {
	case 3:
		value = strings.Repeat(value[0:1], 2) +
			strings.Repeat(value[1:2], 2) +
			strings.Repeat(value[2:3], 2)
	case 6:
		// Already canonical enough to decode.
	case 4, 8:
		return [3]float64{}, fmt.Errorf(
			"color #%s contains alpha; composite it before measuring contrast",
			value,
		)
	default:
		return [3]float64{}, fmt.Errorf("color #%s must contain 3 or 6 hexadecimal digits", value)
	}

	var result [3]float64
	for index := range result {
		channel, err := strconv.ParseUint(value[index*2:index*2+2], 16, 8)
		if err != nil {
			return [3]float64{}, fmt.Errorf("color #%s is invalid: %w", value, err)
		}
		result[index] = float64(channel) / 255
	}
	return result, nil
}

func relativeLuminance(color [3]float64) float64 {
	linear := func(channel float64) float64 {
		if channel <= 0.04045 {
			return channel / 12.92
		}
		return math.Pow((channel+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(color[0]) +
		0.7152*linear(color[1]) +
		0.0722*linear(color[2])
}
