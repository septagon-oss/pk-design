package tokens

// value.go owns small public helpers for safe token-value handling. Extensions
// can use these helpers instead of depending on package internals or ad-hoc
// type assertions.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"strconv"
	"strings"
)

// Reference formats a validated DTCG alias reference such as
// "{color.brand.primary}".
func Reference(path string) (string, error) {
	path = strings.TrimSpace(path)
	if err := validatePath(path, true); err != nil {
		return "", err
	}
	return "{" + path + "}", nil
}

// CopyValue returns a defensive copy of a DTCG token value.
func CopyValue(value Value) Value {
	return deepCopyValue(value)
}

// StringValue returns the token value as a string when it is explicitly stored
// as a JSON or Go string.
func (t Token) StringValue() (string, bool) {
	return stringValue(t.Value)
}

// NumberValue returns the token value as a json.Number when it is explicitly
// stored as a JSON or Go number.
func (t Token) NumberValue() (json.Number, bool) {
	return numberValue(t.Value)
}

// MapValue returns the token value as a defensive copy when it is explicitly
// stored as a JSON or Go object.
func (t Token) MapValue() (map[string]any, bool) {
	return mapValue(t.Value)
}

func stringValue(value Value) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	case json.RawMessage:
		decoded, ok := decodeRawJSON(typed)
		if !ok {
			return "", false
		}
		return stringValue(decoded)
	default:
		return "", false
	}
}

func numberValue(value Value) (json.Number, bool) {
	switch typed := value.(type) {
	case json.Number:
		return typed, true
	case int:
		return json.Number(strconv.Itoa(typed)), true
	case int8:
		return json.Number(strconv.FormatInt(int64(typed), 10)), true
	case int16:
		return json.Number(strconv.FormatInt(int64(typed), 10)), true
	case int32:
		return json.Number(strconv.FormatInt(int64(typed), 10)), true
	case int64:
		return json.Number(strconv.FormatInt(typed, 10)), true
	case uint:
		return json.Number(strconv.FormatUint(uint64(typed), 10)), true
	case uint8:
		return json.Number(strconv.FormatUint(uint64(typed), 10)), true
	case uint16:
		return json.Number(strconv.FormatUint(uint64(typed), 10)), true
	case uint32:
		return json.Number(strconv.FormatUint(uint64(typed), 10)), true
	case uint64:
		return json.Number(strconv.FormatUint(typed, 10)), true
	case float32:
		value := float64(typed)
		if !finiteNumber(value) {
			return "", false
		}
		return json.Number(strconv.FormatFloat(value, 'f', -1, 32)), true
	case float64:
		if !finiteNumber(typed) {
			return "", false
		}
		return json.Number(strconv.FormatFloat(typed, 'f', -1, 64)), true
	case json.RawMessage:
		decoded, ok := decodeRawJSON(typed)
		if !ok {
			return "", false
		}
		return numberValue(decoded)
	default:
		return "", false
	}
}

func mapValue(value Value) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		copied, ok := deepCopyValue(typed).(map[string]any)
		return copied, ok
	case map[string]string:
		copied, ok := deepCopyValue(typed).(map[string]any)
		return copied, ok
	case json.RawMessage:
		decoded, ok := decodeRawJSON(typed)
		if !ok {
			return nil, false
		}
		return mapValue(decoded)
	default:
		return nil, false
	}
}

func finiteNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func decodeRawJSON(value json.RawMessage) (any, bool) {
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, false
	}
	return decoded, true
}
