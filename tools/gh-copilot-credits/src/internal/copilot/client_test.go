package copilot

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeCommandRunner struct {
	name string
	args []string
	data []byte
	err  error
}

func (f *fakeCommandRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.name = name
	f.args = append([]string(nil), args...)
	return f.data, f.err
}

func TestGHClientCurrentUsesGhAPIWithoutShell(t *testing.T) {
	runner := &fakeCommandRunner{data: []byte(`{
		"quota_reset_date_utc":"2026-10-01T00:00:00Z",
		"quota_snapshots":{"premium_interactions":{"quota_id":"premium_interactions","entitlement":20000,"quota_remaining":19000,"remaining":19000}}
	}`)}
	client := NewGHClient(runner, time.Second)
	data, err := client.Current(context.Background())
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if runner.name != "gh" {
		t.Fatalf("command = %q, want gh", runner.name)
	}
	joined := strings.Join(runner.args, " ")
	for _, want := range []string{"api", "/copilot_internal/user", "--hostname", "github.com"} {
		if !strings.Contains(joined, want) {
			t.Errorf("args %q do not contain %q", joined, want)
		}
	}
	if _, err := data.PremiumQuota(); err != nil {
		t.Fatalf("PremiumQuota() error = %v", err)
	}
}

func TestGHClientCurrentDoesNotExposeCommandOutputOnFailure(t *testing.T) {
	runner := &fakeCommandRunner{err: errors.New("exit status 1")}
	client := NewGHClient(runner, time.Second)
	_, err := client.Current(context.Background())
	if err == nil || !strings.Contains(err.Error(), "gh api request failed") {
		t.Fatalf("Current() error = %v", err)
	}
}
