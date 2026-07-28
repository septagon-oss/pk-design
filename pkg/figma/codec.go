package figma

// Implements: REQ-011.
// Per: ADR-0076.
// Discipline: C-14.
// codec.go converts DTCG primitive values into Figma native values while
// retaining enough representation metadata for a lossless governed export.
//
// ADR: ADR-0029 (file purpose declaration), ADR-0076 (layered design delivery).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/septagon-oss/pk-design/pkg/tokens"
)

const (
	codecColor     = "color"
	codecNumber    = "number"
	codecDimension = "dimension"
	codecDuration  = "duration"
	codecString    = "string"
	codecBoolean   = "boolean"
	codecJSON      = "json"
)

var rgbFunctionPattern = regexp.MustCompile(
	`(?i)^rgba?\(\s*([0-9.]+)\s*[, ]\s*([0-9.]+)\s*[, ]\s*([0-9.]+)(?:\s*[,/]\s*([0-9.]+%?))?\s*\)$`,
)

func encodeValue(tokenType tokens.Type, value tokens.Value) (string, any, Codec, bool, error) {
	switch tokenType {
	case tokens.TypeColor:
		native, codec, err := encodeColor(value)
		return "COLOR", native, codec, err == nil, err
	case tokens.TypeDimension:
		native, codec, err := encodeScaledNumber(value, codecDimension)
		return "FLOAT", native, codec, err == nil, err
	case tokens.TypeDuration:
		native, codec, err := encodeScaledNumber(value, codecDuration)
		return "FLOAT", native, codec, err == nil, err
	case tokens.TypeNumber, tokens.TypeFontWeight:
		native, codec, err := encodeNumber(value)
		return "FLOAT", native, codec, err == nil, err
	case tokens.TypeString, tokens.TypeFontFamily, tokens.TypeStrokeStyle:
		native, ok := value.(string)
		if !ok {
			return "", nil, Codec{}, false, fmt.Errorf("string token value is %T", value)
		}
		return "STRING", native, Codec{Kind: codecString}, true, nil
	default:
		if tokenType == tokens.Type("x.boolean") {
			native, ok := value.(bool)
			if !ok {
				return "", nil, Codec{}, false, fmt.Errorf("boolean token value is %T", value)
			}
			return "BOOLEAN", native, Codec{Kind: codecBoolean}, true, nil
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return "", nil, Codec{}, false, err
		}
		return "STRING", string(raw), Codec{Kind: codecJSON}, false, nil
	}
}

func encodeColor(value any) (map[string]float64, Codec, error) {
	text, ok := value.(string)
	if !ok {
		return nil, Codec{}, fmt.Errorf("color token value is %T", value)
	}
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "#") {
		r, g, b, a, style, err := parseHexColor(text)
		if err != nil {
			return nil, Codec{}, err
		}
		return rgba(r, g, b, a), Codec{Kind: codecColor, Style: style}, nil
	}
	if match := rgbFunctionPattern.FindStringSubmatch(text); match != nil {
		r, err := colorChannel(match[1])
		if err != nil {
			return nil, Codec{}, err
		}
		g, err := colorChannel(match[2])
		if err != nil {
			return nil, Codec{}, err
		}
		b, err := colorChannel(match[3])
		if err != nil {
			return nil, Codec{}, err
		}
		alpha := 1.0
		if match[4] != "" {
			alpha, err = alphaChannel(match[4])
			if err != nil {
				return nil, Codec{}, err
			}
		}
		style := "rgb-function"
		if strings.HasPrefix(strings.ToLower(text), "rgba") || match[4] != "" {
			style = "rgba-function"
		}
		return rgba(r, g, b, alpha), Codec{Kind: codecColor, Style: style}, nil
	}
	fields := strings.Fields(text)
	if len(fields) == 3 {
		r, err := colorChannel(fields[0])
		if err != nil {
			return nil, Codec{}, err
		}
		g, err := colorChannel(fields[1])
		if err != nil {
			return nil, Codec{}, err
		}
		b, err := colorChannel(fields[2])
		if err != nil {
			return nil, Codec{}, err
		}
		return rgba(r, g, b, 1), Codec{Kind: codecColor, Style: "rgb-triplet"}, nil
	}
	return nil, Codec{}, fmt.Errorf("unsupported color representation %q", text)
}

func parseHexColor(value string) (r, g, b, a float64, style string, err error) {
	raw := strings.TrimPrefix(value, "#")
	upper := raw == strings.ToUpper(raw)
	switch len(raw) {
	case 3, 4:
		expanded := make([]byte, 0, len(raw)*2)
		for index := range raw {
			expanded = append(expanded, raw[index], raw[index])
		}
		raw = string(expanded)
	case 6, 8:
	default:
		return 0, 0, 0, 0, "", fmt.Errorf("unsupported hex color %q", value)
	}
	bytes := make([]uint64, len(raw)/2)
	for index := range bytes {
		bytes[index], err = strconv.ParseUint(raw[index*2:index*2+2], 16, 8)
		if err != nil {
			return 0, 0, 0, 0, "", fmt.Errorf("invalid hex color %q", value)
		}
	}
	a = 1
	if len(bytes) == 4 {
		a = float64(bytes[3]) / 255
	}
	style = fmt.Sprintf("hex%d-%s", len(raw), map[bool]string{true: "upper", false: "lower"}[upper])
	return float64(bytes[0]), float64(bytes[1]), float64(bytes[2]), a, style, nil
}

func encodeScaledNumber(value any, kind string) (float64, Codec, error) {
	text, ok := value.(string)
	if !ok {
		return 0, Codec{}, fmt.Errorf("%s token value is %T", kind, value)
	}
	text = strings.TrimSpace(text)
	units := []struct {
		suffix string
		scale  float64
	}{
		{"rem", 16},
		{"em", 16},
		{"px", 1},
		{"ms", 1},
		{"s", 1000},
	}
	for _, candidate := range units {
		if !strings.HasSuffix(text, candidate.suffix) {
			continue
		}
		number := strings.TrimSpace(strings.TrimSuffix(text, candidate.suffix))
		parsed, err := strconv.ParseFloat(number, 64)
		if err != nil {
			return 0, Codec{}, fmt.Errorf("invalid %s value %q", kind, text)
		}
		return parsed * candidate.scale, Codec{
			Kind: kind, Unit: candidate.suffix, Scale: candidate.scale,
			Precision: decimalPrecision(number),
		}, nil
	}
	if parsed, err := strconv.ParseFloat(text, 64); err == nil && isFinite(parsed) {
		return parsed, Codec{
			Kind: kind, Style: "unitless-string", Scale: 1,
			Precision: decimalPrecision(text),
		}, nil
	}
	return 0, Codec{}, fmt.Errorf("unsupported %s representation %q", kind, text)
}

func encodeNumber(value any) (float64, Codec, error) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, Codec{Kind: codecNumber, Style: "json-number", Precision: decimalPrecision(string(typed))}, err
	case float64:
		if !isFinite(typed) {
			return 0, Codec{}, fmt.Errorf("number is not finite")
		}
		return typed, Codec{Kind: codecNumber, Style: "number"}, nil
	case int:
		return float64(typed), Codec{Kind: codecNumber, Style: "number"}, nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, Codec{}, fmt.Errorf("invalid number %q", typed)
		}
		return parsed, Codec{
			Kind: codecNumber, Style: "string",
			Precision: decimalPrecision(strings.TrimSpace(typed)),
		}, nil
	default:
		return 0, Codec{}, fmt.Errorf("number token value is %T", value)
	}
}

func rgba(r, g, b, a float64) map[string]float64 {
	return map[string]float64{
		"r": r / 255,
		"g": g / 255,
		"b": b / 255,
		"a": a,
	}
}

func colorChannel(value string) (float64, error) {
	channel, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || channel < 0 || channel > 255 {
		return 0, fmt.Errorf("invalid color channel %q", value)
	}
	return channel, nil
}

func alphaChannel(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if percent, ok := strings.CutSuffix(value, "%"); ok {
		alpha, err := strconv.ParseFloat(percent, 64)
		if err != nil || alpha < 0 || alpha > 100 {
			return 0, fmt.Errorf("invalid alpha channel %q", value)
		}
		return alpha / 100, nil
	}
	alpha, err := strconv.ParseFloat(value, 64)
	if err != nil || alpha < 0 || alpha > 1 {
		return 0, fmt.Errorf("invalid alpha channel %q", value)
	}
	return alpha, nil
}

func decimalPrecision(value string) int {
	value = strings.TrimSpace(value)
	if exponent := strings.IndexAny(value, "eE"); exponent >= 0 {
		value = value[:exponent]
	}
	dot := strings.IndexByte(value, '.')
	if dot < 0 {
		return 0
	}
	return len(strings.TrimRight(value[dot+1:], "0"))
}

func isFinite(value float64) bool {
	return !math.IsInf(value, 0) && !math.IsNaN(value)
}
