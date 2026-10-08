package datadogapi

import (
	"strings"
	"testing"
)

func TestRedactResponseBody_RedactsLogEvents(t *testing.T) {
	t.Parallel()

	// Given
	body := `{"hitCount":3,"result":{"events":[{"message":"secret log line"}]}}`

	// When
	got := redactResponseBody(body)

	// Then
	if strings.Contains(got, "secret log line") {
		t.Fatalf("redactResponseBody leaked event body: %q", got)
	}
	if !strings.Contains(got, "redacted") {
		t.Fatalf("redactResponseBody = %q", got)
	}
}

func TestRedactResponseBody_PreservesErrorJSON(t *testing.T) {
	t.Parallel()

	// Given
	body := `{"errors":["something failed"],"query":"avg:system.cpu.user{*}"}`

	// When
	got := redactResponseBody(body)

	// Then
	if got != body {
		t.Fatalf("redactResponseBody = %q", got)
	}
}

func TestRedactResponseBody_RedactsTokens(t *testing.T) {
	t.Parallel()

	// Given
	body := `{"_authentication_token":"secret-token","errors":["bad request"]}`

	// When
	got := redactResponseBody(body)

	// Then
	if strings.Contains(got, "secret-token") {
		t.Fatalf("redactResponseBody leaked token: %q", got)
	}
	if !strings.Contains(got, `"errors":["bad request"]`) {
		t.Fatalf("redactResponseBody = %q", got)
	}
}

func TestRedactResponseBody_TruncatesLongBodies(t *testing.T) {
	t.Parallel()

	// Given
	body := strings.Repeat("x", 600)

	// When
	got := redactResponseBody(body)

	// Then
	if len(got) > 520 {
		t.Fatalf("redactResponseBody too long: len=%d", len(got))
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("redactResponseBody = %q", got)
	}
}

func TestRedactResponseBody_EmptyBody(t *testing.T) {
	t.Parallel()

	// When
	got := redactResponseBody("   ")

	// Then
	if got != "" {
		t.Fatalf("redactResponseBody = %q", got)
	}
}
