package slack

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/auth"
)

func testCredential(t *testing.T) auth.Credential {
	t.Helper()
	return auth.Credential{
		WorkspaceHost: "alpha.slack.com",
		WorkspaceID:   "T11111111",
		Token:         auth.NewSecret("generated-token-" + t.Name()),
		Cookie:        auth.NewSecret("generated-cookie-" + t.Name()),
	}
}

func TestClientAuthAndCredentialHeaders(t *testing.T) {
	cred := testCredential(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != cred.Cookie.Reveal() {
			t.Error("cookie header missing")
		}
		if err := r.ParseForm(); err != nil || r.Form.Get("token") != cred.Token.Reveal() {
			t.Error("token form field missing")
		}
		_, _ = io.WriteString(w, `{"ok":true,"user_id":"U11111111","user":"Example User","team_id":"T11111111"}`)
	}))
	defer server.Close()

	var logs strings.Builder
	client := NewClient(server.URL, cred, WithLogger(func(s string) { logs.WriteString(s) }), WithRequestDelay(0))
	got, err := client.AuthTest(t.Context())
	if err != nil || got.UserID != "U11111111" {
		t.Fatalf("AuthTest = %#v, %v", got, err)
	}
	if strings.Contains(logs.String(), cred.Token.Reveal()) || strings.Contains(logs.String(), cred.Cookie.Reveal()) {
		t.Fatal("debug log leaked credentials")
	}
}

func TestClientRetriesTransientResponses(t *testing.T) {
	var calls atomic.Int32
	var sleeps []time.Duration
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		switch calls.Add(1) {
		case 1:
			w.WriteHeader(http.StatusInternalServerError)
		case 2:
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			_, _ = io.WriteString(w, `{"ok":true,"user_id":"U11111111"}`)
		}
	}))
	defer server.Close()
	client := NewClient(server.URL, testCredential(t), WithSleeper(func(_ context.Context, d time.Duration) error {
		sleeps = append(sleeps, d)
		return nil
	}), WithRequestDelay(0), WithMaxAttempts(3))
	if _, err := client.AuthTest(t.Context()); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 || len(sleeps) != 2 || sleeps[1] != 2*time.Second {
		t.Fatalf("calls=%d sleeps=%v", calls.Load(), sleeps)
	}
}

func TestClientDoesNotRetryAPIOrMalformedErrors(t *testing.T) {
	tests := []struct {
		name, body string
		wantKind   ErrorKind
	}{
		{"auth", `{"ok":false,"error":"invalid_auth"}`, ErrorAuth},
		{"permission", `{"ok":false,"error":"not_in_channel"}`, ErrorPermission},
		{"malformed", `{`, ErrorMalformed},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			client := NewClient(server.URL, testCredential(t), WithRequestDelay(0), WithMaxAttempts(3))
			_, err := client.AuthTest(t.Context())
			var apiErr *APIError
			if !errors.As(err, &apiErr) || apiErr.Kind != tc.wantKind || calls.Load() != 1 {
				t.Fatalf("error=%v calls=%d", err, calls.Load())
			}
		})
	}
}

func TestClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		time.Sleep(50 * time.Millisecond)
	}))
	defer server.Close()
	httpClient := &http.Client{Timeout: time.Millisecond}
	client := NewClient(server.URL, testCredential(t), WithHTTPClient(httpClient), WithRequestDelay(0), WithMaxAttempts(1))
	if _, err := client.AuthTest(t.Context()); err == nil {
		t.Fatal("expected timeout")
	}
}
