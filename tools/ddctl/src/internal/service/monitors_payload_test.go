package service

import "testing"

func TestMetricQueryForPreflight(t *testing.T) {
	t.Parallel()

	got := metricQueryForPreflight("avg(last_5m):avg:system.cpu.user{*} > 100")
	want := "avg:system.cpu.user{*}"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestApplyGlobalMute_SetsGlobalScope(t *testing.T) {
	t.Parallel()

	payload := map[string]any{
		"name":  "test",
		"query": "avg(last_5m):avg:system.cpu.user{*} > 1",
		"type":  "metric alert",
	}
	if err := applyGlobalMute(payload); err != nil {
		t.Fatalf("applyGlobalMute() error = %v", err)
	}
	if !monitorGloballyMuted(payload) {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestApplyGlobalMute_ConflictsWithScopedSilenced(t *testing.T) {
	t.Parallel()

	payload := map[string]any{
		"options": map[string]any{
			"silenced": map[string]any{
				"host:app1": nil,
			},
		},
	}
	err := applyGlobalMute(payload)
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestMergeMonitorSilencedFromRemote_PreservesMute(t *testing.T) {
	t.Parallel()

	next := map[string]any{
		"name":  "test",
		"query": "avg(last_5m):avg:system.cpu.user{*} > 1",
		"type":  "metric alert",
		"options": map[string]any{
			"thresholds": map[string]any{"critical": 1},
		},
	}
	current := map[string]any{
		"options": map[string]any{
			"silenced": map[string]any{
				"*": nil,
			},
		},
	}
	mergeMonitorSilencedFromRemote(next, current)
	if !monitorGloballyMuted(next) {
		t.Fatalf("next = %#v", next)
	}
}

func TestMergeMonitorSilencedFromRemote_ExplicitEmptyHonored(t *testing.T) {
	t.Parallel()

	next := map[string]any{
		"options": map[string]any{
			"silenced": map[string]any{},
		},
	}
	current := map[string]any{
		"options": map[string]any{
			"silenced": map[string]any{
				"*": nil,
			},
		},
	}
	mergeMonitorSilencedFromRemote(next, current)
	if monitorGloballyMuted(next) {
		t.Fatalf("next = %#v", next)
	}
}
