// Package timeutil provides time parsing helpers for ddctl commands.
package timeutil

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseToUnixMs converts a time expression to Unix milliseconds.
//
// Accepted formats:
//   - "now"                   → current time
//   - "now-<N>m"              → N minutes ago
//   - "now-<N>h"              → N hours ago
//   - "now-<N>d"              → N days ago
//   - Unix milliseconds       → returned as-is (e.g. 1776540000000)
//   - RFC3339 / ISO-8601      → parsed (e.g. 2024-01-01T00:00:00Z)
func ParseToUnixMs(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "now" {
		return time.Now().UnixMilli(), nil
	}
	if strings.HasPrefix(s, "now-") {
		return parseRelative(s)
	}
	// Unix milliseconds (13 digits typically)
	if ms, err := strconv.ParseInt(s, 10, 64); err == nil {
		return ms, nil
	}
	// RFC3339 / ISO-8601
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0, fmt.Errorf("cannot parse %q: use relative (now-1h), Unix ms, or RFC3339", s)
	}
	return t.UnixMilli(), nil
}

// ParseToUnixSec converts a time expression to Unix seconds.
// Accepts the same formats as ParseToUnixMs.
func ParseToUnixSec(s string) (int64, error) {
	ms, err := ParseToUnixMs(s)
	if err != nil {
		return 0, err
	}
	return ms / 1000, nil
}

func parseRelative(s string) (int64, error) {
	suffix := s[4:] // strip "now-"
	if suffix == "" {
		return 0, fmt.Errorf("invalid relative duration %q", s)
	}
	// Find where digits end
	i := 0
	for i < len(suffix) && suffix[i] >= '0' && suffix[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, fmt.Errorf("invalid relative duration %q: expected number before unit", s)
	}
	n, err := strconv.ParseInt(suffix[:i], 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid relative duration %q", s)
	}
	unit := suffix[i:]
	var d time.Duration
	switch unit {
	case "m", "min":
		d = time.Duration(n) * time.Minute
	case "h":
		d = time.Duration(n) * time.Hour
	case "d":
		d = time.Duration(n) * 24 * time.Hour
	case "w":
		d = time.Duration(n) * 7 * 24 * time.Hour
	default:
		return 0, fmt.Errorf("unknown time unit %q in %q (use m, h, d, w)", unit, s)
	}
	return time.Now().Add(-d).UnixMilli(), nil
}
