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
