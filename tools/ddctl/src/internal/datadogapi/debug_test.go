package datadogapi

import (
	"strings"
	"testing"
)

func TestRedactDetails_RedactsLogEvents(t *testing.T) {
	t.Parallel()

	body := `{"hitCount":3,"result":{"events":[{"message":"secret log line"}]}}`
	got := redactDetails(body)
	if strings.Contains(got, "secret log line") {
		t.Fatalf("redactDetails leaked event body: %q", got)
	}
	if !strings.Contains(got, "redacted") {
		t.Fatalf("redactDetails = %q", got)
	}
}

func TestRedactDetails_TruncatesLongBodies(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("x", 600)
	got := redactDetails(body)
	if len(got) > 520 {
		t.Fatalf("redactDetails too long: len=%d", len(got))
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("redactDetails = %q", got)
	}
}
