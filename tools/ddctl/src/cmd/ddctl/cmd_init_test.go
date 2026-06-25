package main

import (
	"strings"
	"testing"
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
