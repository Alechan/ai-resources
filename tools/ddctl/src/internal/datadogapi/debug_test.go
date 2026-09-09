package datadogapi

import (
	"strings"
	"testing"
)

func TestRedactDetails_RedactsLogEventsInDebug(t *testing.T) {
	t.Parallel()

	body := `{"hitCount":3,"result":{"events":[{"message":"secret log line"}]}}`
	got := redactDetails(body, true)
	if strings.Contains(got, "secret log line") {
		t.Fatalf("redactDetails leaked event body: %q", got)
	}
	if !strings.Contains(got, "redacted") {
		t.Fatalf("redactDetails = %q", got)
	}
}

func TestRedactDetails_NonDebugAlwaysRedacts(t *testing.T) {
	t.Parallel()

	body := `{"errors":["something failed"],"query":"avg:system.cpu.user{*}"}`
	got := redactDetails(body, false)
	if got != redactedBodyPlaceholder {
		t.Fatalf("redactDetails = %q, want placeholder", got)
	}
}

func TestRedactDetails_DebugRedactsTokens(t *testing.T) {
	t.Parallel()

	body := `{"_authentication_token":"secret-token","errors":["bad request"]}`
	got := redactDetails(body, true)
	if strings.Contains(got, "secret-token") {
		t.Fatalf("redactDetails leaked token: %q", got)
	}
}

func TestRedactDetails_DebugTruncatesLongBodies(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("x", 600)
	got := redactDetails(body, true)
	if len(got) > 520 {
		t.Fatalf("redactDetails too long: len=%d", len(got))
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("redactDetails = %q", got)
	}
}
