package topics

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode"
)

func RewriteSpaceIDs(raw, spaceID string) (string, error) {
	if strings.TrimSpace(spaceID) == "" {
		return "", errors.New("space ID is required")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var body []any
	if err := decoder.Decode(&body); err != nil {
		return "", errors.New("stored body is not JSON")
	}
	limit := len(body)
	if limit >= 100 {
		limit = 99
	}
	found := false
	for i := 0; i < limit; i++ {
		if containsSpaceID(body[i]) {
			body[i] = replaceExactSpaceID(body[i], spaceID)
			found = true
		}
	}
	if !found {
		return "", errors.New("stored body has no space identifier")
	}
	return marshalBody(body)
}

func containsSpaceID(value any) bool {
	switch typed := value.(type) {
	case string:
		return isWebSpaceID(typed)
	case []any:
		for _, item := range typed {
			if containsSpaceID(item) {
				return true
			}
		}
	}
	return false
}

func replaceExactSpaceID(value any, spaceID string) any {
	switch typed := value.(type) {
	case string:
		if isWebSpaceID(typed) {
			return spaceID
		}
		return typed
	case []any:
		replaced := make([]any, len(typed))
		for i, item := range typed {
			replaced[i] = replaceExactSpaceID(item, spaceID)
		}
		return replaced
	default:
		return typed
	}
}

func isWebSpaceID(value string) bool {
	if len(value) != 11 {
		return false
	}
	hasLetter := false
	for _, r := range value {
		if unicode.IsLetter(r) {
			hasLetter = true
		} else if !unicode.IsDigit(r) && r != '_' && r != '-' {
			return false
		}
	}
	return hasLetter
}
