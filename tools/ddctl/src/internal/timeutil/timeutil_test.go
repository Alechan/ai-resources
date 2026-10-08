package timeutil_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/timeutil"
)

func TestParseToUnixMs_Now(t *testing.T) {
	before := time.Now().UnixMilli()
	ms, err := timeutil.ParseToUnixMs("now")
	after := time.Now().UnixMilli()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ms < before || ms > after {
		t.Errorf("'now' %d not in [%d, %d]", ms, before, after)
	}
}

func TestParseToUnixMs_Relative(t *testing.T) {
	cases := []struct {
		input string
		minMs int64 // must be at least this many ms in the past
		maxMs int64 // must be at most this many ms in the past
	}{
		{"now-30m", 29 * 60 * 1000, 31 * 60 * 1000},
		{"now-1h", 59 * 60 * 1000, 61 * 60 * 1000},
		{"now-2d", 2*24*60*60*1000 - 60000, 2*24*60*60*1000 + 60000},
		{"now-1w", 7*24*60*60*1000 - 60000, 7*24*60*60*1000 + 60000},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			ms, err := timeutil.ParseToUnixMs(c.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			ago := time.Now().UnixMilli() - ms
			if ago < c.minMs || ago > c.maxMs {
				t.Errorf("%s: %d ms ago, want [%d, %d]", c.input, ago, c.minMs, c.maxMs)
			}
		})
	}
}

func TestParseToUnixMs_UnixMs(t *testing.T) {
	ms, err := timeutil.ParseToUnixMs("1776540000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ms != 1776540000000 {
		t.Errorf("got %d, want 1776540000000", ms)
	}
}

func TestParseToUnixMs_RFC3339(t *testing.T) {
	ms, err := timeutil.ParseToUnixMs("2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	if ms != expected {
		t.Errorf("got %d, want %d", ms, expected)
	}
}

func TestParseToUnixMs_Errors(t *testing.T) {
	bad := []string{"now-", "now-0h", "now-1x", "notadate", ""}
	for _, s := range bad {
		_, err := timeutil.ParseToUnixMs(s)
		if err == nil {
			t.Errorf("expected error for %q", s)
		}
		_ = strings.Contains(err.Error(), s) // just exercise the message
	}
}
