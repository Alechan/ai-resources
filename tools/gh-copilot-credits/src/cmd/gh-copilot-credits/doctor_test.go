package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Alechan/ai-resources/tools/gh-copilot-credits/src/internal/copilot"
)

func TestRunDoctorReportsHealthyEndpoint(t *testing.T) {
	data := copilot.UserData{
		QuotaSnapshots: map[string]copilot.QuotaSnapshot{
			copilot.PremiumInteractionsQuotaID: {
				QuotaID:        copilot.PremiumInteractionsQuotaID,
				Entitlement:    20000,
				QuotaRemaining: 10000,
				Remaining:      10000,
			},
		},
	}
	var stdout, stderr strings.Builder
	code := runWithLookPath([]string{"doctor"}, fakeClient{data: data}, &stdout, &stderr, time.Now, func(string) (string, error) {
		return "/opt/homebrew/bin/gh", nil
	})
	if code != exitOK {
		t.Fatalf("run() code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"gh: found", "endpoint: ok", "premium_interactions"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("stdout does not contain %q: %s", want, stdout.String())
		}
	}
}

func TestRunDoctorReportsMissingGH(t *testing.T) {
	var stdout, stderr strings.Builder
	code := runWithLookPath([]string{"doctor"}, fakeClient{}, &stdout, &stderr, time.Now, func(string) (string, error) {
		return "", fmt.Errorf("not found")
	})
	if code != exitFailure {
		t.Fatalf("run() code = %d, want %d", code, exitFailure)
	}
	if !strings.Contains(stderr.String(), "gh: not found") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestRunDoctorReportsEndpointFailure(t *testing.T) {
	var stdout, stderr strings.Builder
	code := runWithLookPath([]string{"doctor"}, fakeClient{err: errors.New("authentication failed")}, &stdout, &stderr, time.Now, func(string) (string, error) {
		return "/opt/homebrew/bin/gh", nil
	})
	if code != exitFailure {
		t.Fatalf("run() code = %d, want %d", code, exitFailure)
	}
	if !strings.Contains(stderr.String(), "endpoint: authentication failed") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}
