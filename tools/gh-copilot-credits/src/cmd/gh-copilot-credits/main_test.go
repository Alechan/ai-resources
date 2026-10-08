package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Alechan/ai-resources/tools/gh-copilot-credits/src/internal/copilot"
)

type fakeClient struct {
	data copilot.UserData
	err  error
}

func (f fakeClient) Current(context.Context) (copilot.UserData, error) {
	return f.data, f.err
}

func TestRunCurrentTable(t *testing.T) {
	data := copilot.UserData{
		CopilotPlan:       "business",
		QuotaResetDateUTC: "2026-10-01T00:00:00Z",
		QuotaSnapshots: map[string]copilot.QuotaSnapshot{
			copilot.PremiumInteractionsQuotaID: {
				QuotaID:        copilot.PremiumInteractionsQuotaID,
				TimestampUTC:   "2026-09-26T13:35:02Z",
				Entitlement:    20000,
				QuotaRemaining: 2272,
				Remaining:      2272,
			},
		},
	}

	var stdout, stderr strings.Builder
	code := run([]string{"current"}, fakeClient{data: data}, &stdout, &stderr, func() time.Time {
		return time.Date(2026, 9, 26, 13, 35, 2, 0, time.UTC)
	})
	if code != 0 {
		t.Fatalf("run() code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "premium_interactions") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestRunCurrentJSON(t *testing.T) {
	data := copilot.UserData{
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
	code := run([]string{"current", "--json"}, fakeClient{data: data}, &stdout, &stderr, func() time.Time {
		return time.Date(2026, 9, 26, 13, 35, 2, 0, time.UTC)
	})
	if code != 0 {
		t.Fatalf("run() code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "{") || !strings.Contains(stdout.String(), `"quota"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestRunCurrentReportsSafeError(t *testing.T) {
	var stdout, stderr strings.Builder
	code := run([]string{"current"}, fakeClient{err: errors.New("authentication failed")}, &stdout, &stderr, time.Now)
	if code == 0 {
		t.Fatal("run() code = 0, want failure")
	}
	if !strings.Contains(stderr.String(), "authentication failed") {
		t.Fatalf("stderr = %s", stderr.String())
	}
	if strings.Contains(stderr.String(), "access_token") {
		t.Fatalf("stderr contains a credential-like field: %s", stderr.String())
	}
}
