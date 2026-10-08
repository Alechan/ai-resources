package main

import (
	"strings"
	"testing"
	"time"

	"github.com/Alechan/ai-resources/tools/gh-copilot-credits/src/internal/copilot"
)

func TestRunCurrentAppendWritesSnapshot(t *testing.T) {
	data := copilot.UserData{
		CopilotPlan:       "business",
		QuotaResetDateUTC: "2026-10-01T00:00:00Z",
		QuotaSnapshots: map[string]copilot.QuotaSnapshot{
			copilot.PremiumInteractionsQuotaID: {
				QuotaID:        copilot.PremiumInteractionsQuotaID,
				Entitlement:    20000,
				QuotaRemaining: 2272,
				Remaining:      2272,
			},
		},
	}
	var stdout, stderr strings.Builder
	code := run([]string{"current", "append", "--csv", t.TempDir() + "/usage.csv"}, fakeClient{data: data}, &stdout, &stderr, func() time.Time {
		return time.Date(2026, 9, 26, 13, 35, 2, 0, time.UTC)
	})
	if code != 0 {
		t.Fatalf("run() code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "appended") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestRunCurrentAppendRequiresCSV(t *testing.T) {
	var stdout, stderr strings.Builder
	code := run([]string{"current", "append"}, fakeClient{}, &stdout, &stderr, time.Now)
	if code != exitUsage {
		t.Fatalf("run() code = %d, want %d", code, exitUsage)
	}
}
