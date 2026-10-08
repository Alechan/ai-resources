package topics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func SlotShape(raw []byte) string {
	trimmed := StripXSSI(raw)
	if len(trimmed) == 0 {
		return "empty"
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "not_json"
	}
	list, ok := value.([]any)
	if !ok {
		return shapeValue(value, 0)
	}
	parts := []string{fmt.Sprintf("len=%d", len(list))}
	for i, item := range list {
		if item == nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("%d:%s", i, shapeValue(item, 0)))
	}
	return strings.Join(parts, " ")
}

func Shape(raw []byte) string {
	trimmed := StripXSSI(raw)
	if len(trimmed) == 0 {
		return "empty"
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "not_json"
	}
	return fmt.Sprintf("bytes=%d prefix=%s %s", len(raw), prefixKind(raw), shapeValue(value, 0))
}

func prefixKind(raw []byte) string {
	trimmed := bytes.TrimSpace(raw)
	switch {
	case bytes.HasPrefix(trimmed, []byte(")]}'")):
		return "xssi"
	case len(trimmed) > 0 && trimmed[0] == '[':
		return "json_array"
	case len(trimmed) > 0 && trimmed[0] == '{':
		return "json_object"
	default:
		return "other"
	}
}

func shapeValue(value any, depth int) string {
	if depth > 6 {
		return "..."
	}
	switch typed := value.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case json.Number:
		return "num"
	case string:
		kind := "str"
		if isDigits(typed) {
			kind = "digits"
		} else if isAlnumID(typed) {
			kind = "id"
		}
		return fmt.Sprintf("str(%d,%s)", len(typed), kind)
	case []any:
		if len(typed) == 0 {
			return "[]"
		}
		parts := make([]string, 0, 12)
		for i, item := range typed {
			if i >= 12 {
				parts = append(parts, fmt.Sprintf("+%d", len(typed)-12))
				break
			}
			parts = append(parts, shapeValue(item, depth+1))
		}
		return "[" + strings.Join(parts, ",") + "]"
	default:
		return "other"
	}
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isAlnumID(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9', r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}
