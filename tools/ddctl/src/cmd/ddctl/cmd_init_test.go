package main

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

// ── sanitizeCookieString ──────────────────────────────────────────────────

func TestSanitizeCookieString_DropsInvalidFragments(t *testing.T) {
	t.Parallel()

	in := `dogweb=abc; tcm={"bad":"json"}; _dd_s_v2=ok; stray; quoted="bad"`
	got, dropped := sanitizeCookieString(in)

	if got != "dogweb=abc; _dd_s_v2=ok" {
		t.Fatalf("sanitizeCookieString() = %q, want %q", got, "dogweb=abc; _dd_s_v2=ok")
	}
	if len(dropped) == 0 {
		t.Fatalf("dropped = %v, want dropped cookie names", dropped)
	}
}

func TestSanitizeCookieString_KeepsValidCookies(t *testing.T) {
	t.Parallel()

	in := "dogweb=abc123; _dd_s_v2=xyz789; dd_csrf_token=token"
	got, dropped := sanitizeCookieString(in)

	if !strings.Contains(got, "dogweb=abc123") {
		t.Fatalf("sanitizeCookieString() missing dogweb: %q", got)
	}
	if !strings.Contains(got, "_dd_s_v2=xyz789") {
		t.Fatalf("sanitizeCookieString() missing _dd_s_v2: %q", got)
	}
	if len(dropped) > 0 {
		t.Fatalf("sanitizeCookieString() unexpectedly dropped: %v", dropped)
	}
}

// ── validateInitAuthMaterial ──────────────────────────────────────────────

func TestValidateInitAuthMaterial_RequiresCSRFAndSessionCookie(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cookieStr string
		wantOK    bool
	}{
		{"valid with csrf and dogweb", "dogweb=abc; dd_csrf_token=token", true},
		{"missing csrf", "dogweb=abc", false},
		{"missing session cookie", "dd_csrf_token=token", false},
		{"valid with csrf and _dd_s_v2", "_dd_s_v2=abc; dd_csrf_token=token", true},
		{"valid with dogwebu", "dogwebu=abc; dd_csrf_token=token", true},
		{"valid with _csrf alias", "dogweb=abc; _csrf=token", true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateInitAuthMaterial(tc.cookieStr)
			if tc.wantOK && err != nil {
				t.Fatalf("validateInitAuthMaterial() error = %v, want nil", err)
			}
			if !tc.wantOK && err == nil {
				t.Fatalf("validateInitAuthMaterial() error = nil, want non-nil")
			}
		})
	}
}

// ── initFromStdinWithDetector ─────────────────────────────────────────────

func TestInitFromStdinWithDetector_ShowsHelpOnTerminal(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := initFromStdinWithDetector(
		context.Background(),
		app.Services{},
		app.Config{},
		strings.NewReader(""),
		&stdout,
		&stderr,
		func(io.Reader) bool { return true },
	)

	if code != fail.CodeOK {
		t.Fatalf("code = %d, want %d", code, fail.CodeOK)
	}
	if !strings.Contains(stdout.String(), "pbpaste | ddctl init") {
		t.Fatalf("stdout = %q, want init documentation", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestInitFromStdinWithDetector_ShowsHelpOnEmptyPipedInput(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := initFromStdinWithDetector(
		context.Background(),
		app.Services{},
		app.Config{},
		strings.NewReader(""),
		&stdout,
		&stderr,
		func(io.Reader) bool { return false }, // not a terminal
	)

	if code != fail.CodeOK {
		t.Fatalf("code = %d, want %d", code, fail.CodeOK)
	}
	if !strings.Contains(stdout.String(), "pbpaste | ddctl init") {
		t.Fatalf("stdout = %q, want init documentation on empty pipe", stdout.String())
	}
}

func TestInitFromStdinWithDetector_ShowsHelpOnWhitespaceOnlyInput(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := initFromStdinWithDetector(
		context.Background(),
		app.Services{},
		app.Config{},
		strings.NewReader("   \n\t\n  "),
		&stdout,
		&stderr,
		func(io.Reader) bool { return false },
	)

	if code != fail.CodeOK {
		t.Fatalf("code = %d, want %d", code, fail.CodeOK)
	}
	if !strings.Contains(stdout.String(), "pbpaste | ddctl init") {
		t.Fatalf("stdout = %q, want init documentation on whitespace-only input", stdout.String())
	}
}

func TestInitFromStdinWithDetector_FailsOnCurlMissingCookies(t *testing.T) {
	t.Parallel()

	// A cURL command with no -b or Cookie header → ExtractCookieHeader will fail.
	curlNoCookies := `curl 'https://app.datadoghq.com/api/v1/logs-analytics/list' -X POST -H 'content-type: application/json'`

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := initFromStdinWithDetector(
		context.Background(),
		app.Services{},
		app.Config{},
		strings.NewReader(curlNoCookies),
		&stdout,
		&stderr,
		func(io.Reader) bool { return false },
	)

	if code != fail.CodeValidation {
		t.Fatalf("code = %d, want %d (validation error)", code, fail.CodeValidation)
	}
	if !strings.Contains(stderr.String(), "Cookie") {
		t.Errorf("stderr = %q, want mention of missing Cookie header", stderr.String())
	}
}

func TestInitFromStdinWithDetector_FailsOnMissingCSRFToken(t *testing.T) {
	t.Parallel()

	// cURL with a cookie header but no CSRF token and no dd_csrf_token cookie.
	curlNoCSRF := `curl 'https://app.datadoghq.com/api/v1/logs-analytics/list' -b 'dogweb=abc123; _dd_s_v2=xyz'`

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := initFromStdinWithDetector(
		context.Background(),
		app.Services{},
		app.Config{},
		strings.NewReader(curlNoCSRF),
		&stdout,
		&stderr,
		func(io.Reader) bool { return false },
	)

	if code != fail.CodeValidation {
		t.Fatalf("code = %d, want %d (validation error for missing CSRF)", code, fail.CodeValidation)
	}
	if !strings.Contains(stderr.String(), "CSRF") {
		t.Errorf("stderr = %q, want mention of missing CSRF token", stderr.String())
	}
}

func TestInitFromStdinWithDetector_FailsOnMissingSessionCookie(t *testing.T) {
	t.Parallel()

	// Has CSRF token via -H header, but no session cookie (dogweb/dogwebu/_dd_s_v2).
	curlNoSession := `curl 'https://app.datadoghq.com/api/v1/logs-analytics/list' -b 'random=value' -H 'x-csrf-token: tok123'`

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := initFromStdinWithDetector(
		context.Background(),
		app.Services{},
		app.Config{},
		strings.NewReader(curlNoSession),
		&stdout,
		&stderr,
		func(io.Reader) bool { return false },
	)

	if code != fail.CodeValidation {
		t.Fatalf("code = %d, want %d (validation error for missing session)", code, fail.CodeValidation)
	}
	if !strings.Contains(stderr.String(), "session") {
		t.Errorf("stderr = %q, want mention of missing session cookie", stderr.String())
	}
}

// ── isTerminalInput ──────────────────────────────────────────────────────

func TestIsTerminalInput_ReturnsFalseForNonFileReader(t *testing.T) {
	t.Parallel()

	// A strings.Reader is not an *os.File, so should return false.
	if isTerminalInput(strings.NewReader("hello")) {
		t.Fatal("isTerminalInput(strings.Reader) = true, want false")
	}
}
