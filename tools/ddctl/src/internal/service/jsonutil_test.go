package service

import (
	"testing"
	"time"
)

func TestModifiedAtMatches_FlexibleFormats(t *testing.T) {
	t.Parallel()

	got := "2026-09-09T01:00:00.000000+00:00"
	want := "2026-09-09T01:00:00Z"
	if !modifiedAtMatches(got, want) {
		t.Fatalf("modifiedAtMatches(%q, %q) = false", got, want)
	}
}

func TestParseFlexibleTime_RFC3339(t *testing.T) {
	t.Parallel()

	ts, err := parseFlexibleTime("2026-09-09T01:00:00Z")
	if err != nil {
		t.Fatalf("parseFlexibleTime() error = %v", err)
	}
	if ts.UTC() != time.Date(2026, 9, 9, 1, 0, 0, 0, time.UTC) {
		t.Fatalf("time = %v", ts)
	}
}

func TestCloneMap_DeepCopy(t *testing.T) {
	t.Parallel()

	src := map[string]any{"nested": map[string]any{"x": 1}}
	copy := cloneMap(src)
	nested, _ := copy["nested"].(map[string]any)
	nested["x"] = 2
	orig, _ := src["nested"].(map[string]any)
	if orig["x"] != 1 {
		t.Fatal("cloneMap should deep-copy nested maps")
	}
}

func TestAssertModifiedAtMatches_RejectsDrift(t *testing.T) {
	t.Parallel()

	err := assertModifiedAtMatches("2026-09-09T02:00:00Z", "2026-09-09T01:00:00Z", "dashboard")
	if err == nil {
		t.Fatal("expected mismatch error")
	}
}
