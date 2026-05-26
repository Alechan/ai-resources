package main

import "testing"

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

func TestValidateInitAuthMaterial_RequiresCSRFAndSessionCookie(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cookieStr string
		wantOK    bool
	}{
		{
			name:      "valid with csrf and dogweb",
			cookieStr: "dogweb=abc; dd_csrf_token=token",
			wantOK:    true,
		},
		{
			name:      "missing csrf",
			cookieStr: "dogweb=abc",
			wantOK:    false,
		},
		{
			name:      "missing session cookie",
			cookieStr: "dd_csrf_token=token",
			wantOK:    false,
		},
		{
			name:      "valid with csrf and _dd_s_v2",
			cookieStr: "_dd_s_v2=abc; dd_csrf_token=token",
			wantOK:    true,
		},
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
