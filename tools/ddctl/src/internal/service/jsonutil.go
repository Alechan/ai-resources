package service

import (
	"encoding/json"
	"strings"
	"time"
)

func modifiedAtMatches(got, expected string) bool {
	got = strings.TrimSpace(got)
	expected = strings.TrimSpace(expected)
	if got == expected {
		return true
	}
	gt, gerr := parseFlexibleTime(got)
	et, eerr := parseFlexibleTime(expected)
	if gerr != nil || eerr != nil {
		return false
	}
	return gt.Equal(et)
}

func parseFlexibleTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000000+00:00",
		"2006-01-02T15:04:05Z",
	}
	var last error
	for _, f := range formats {
		ts, err := time.Parse(f, s)
		if err == nil {
			return ts, nil
		}
		last = err
	}
	return time.Time{}, last
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	b, err := json.Marshal(in)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return map[string]any{}
	}
	return out
}
