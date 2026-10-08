package search

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

func RewriteQuery(body, query string) (string, error) {
	if strings.TrimSpace(query) == "" {
		return "", errors.New("search query is empty")
	}
	form, err := url.ParseQuery(body)
	if err != nil {
		return "", errors.New("stored search capture is not a form body")
	}
	raw := form.Get("f.req")
	if raw == "" {
		return "", errors.New("stored search capture is missing f.req; run gchatctl learn search.messages")
	}
	rewritten, err := rewriteFreq(raw, query)
	if err != nil {
		return "", err
	}
	form.Set("f.req", rewritten)
	form.Del("")
	return form.Encode(), nil
}

func rewriteFreq(raw, query string) (string, error) {
	var root []any
	if err := unmarshalList(raw, &root); err != nil {
		return "", errors.New("stored search f.req is not JSON")
	}
	if !replaceQuerySlot(root, query) {
		return "", errors.New("stored search f.req has no query slot")
	}
	return marshal(root)
}

func replaceQuerySlot(value any, query string) bool {
	list, ok := value.([]any)
	if !ok || len(list) == 0 {
		return false
	}
	if len(list) > 1 {
		if payload, ok := list[1].(string); ok {
			inner, err := decodeList(payload)
			if err == nil && len(inner) > 3 {
				if _, ok := inner[3].(string); ok {
					inner[3] = query
					encoded, err := marshal(inner)
					if err != nil {
						return false
					}
					list[1] = encoded
					return true
				}
			}
		}
	}
	for _, item := range list {
		if replaceQuerySlot(item, query) {
			return true
		}
	}
	return false
}

func decodeList(raw string) ([]any, error) {
	var inner []any
	if err := unmarshalList(raw, &inner); err != nil {
		return nil, err
	}
	return inner, nil
}

func unmarshalList(raw string, dest *[]any) error {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(dest); err != nil {
		return err
	}
	return nil
}

func marshal(value any) (string, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return "", err
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}
