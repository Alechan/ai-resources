package datadogapi

import "testing"

func TestPathNeedsCSRF(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{"/api/v1/logs-analytics/list?type=logs", true},
		{"/api/v1/logs-analytics/list?type=feed", true},
		{"/api/v1/dashboard", true},
		{"/api/v1/dashboard/abc-def", true},
		{"/api/v1/notebooks", true},
		{"/api/v1/monitor/123", true},
		{"/api/v1/query", false},
		{"/api/v1/validate", false},
	}
	for _, tc := range tests {
		if got := pathNeedsCSRF(tc.path); got != tc.want {
			t.Fatalf("pathNeedsCSRF(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}
