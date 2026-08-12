package slack

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
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

func TestHistoryDecodesMessageReactions(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []Reaction
	}{
		{
			name: "one reaction",
			body: `{"ok":true,"messages":[{"ts":"100.000001","reactions":[{"name":"ok","count":3,"users":["U22222222","U33333333"]}]}]}`,
			want: []Reaction{{Name: "ok", Count: 3, Users: []string{"U22222222", "U33333333"}}},
		},
		{
			name: "multiple reactions",
			body: `{"ok":true,"messages":[{"ts":"100.000001","reactions":[{"name":"eyes","count":1,"users":["U22222222"]},{"name":"custom_status","count":2,"users":["U33333333"]}]}]}`,
			want: []Reaction{
				{Name: "eyes", Count: 1, Users: []string{"U22222222"}},
				{Name: "custom_status", Count: 2, Users: []string{"U33333333"}},
			},
		},
		{
			name: "missing reactions",
			body: `{"ok":true,"messages":[{"ts":"100.000001"}]}`,
		},
		{
			name: "authoritative count exceeds returned users",
			body: `{"ok":true,"messages":[{"ts":"100.000001","reactions":[{"name":"ok","count":4,"users":["U22222222"]}]}]}`,
			want: []Reaction{{Name: "ok", Count: 4, Users: []string{"U22222222"}}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()

			page, raw, err := NewClient(server.URL, testCredential(t), WithRequestDelay(0)).History(
				t.Context(), "C22222222", "", "", "", 100,
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(page.Messages) != 1 {
				t.Fatalf("messages = %#v", page.Messages)
			}
			if !reflect.DeepEqual(page.Messages[0].Reactions, tc.want) {
				t.Fatalf("reactions = %#v, want %#v", page.Messages[0].Reactions, tc.want)
			}
			if string(raw) != tc.body {
				t.Fatalf("raw response changed: %q, want %q", raw, tc.body)
			}
		})
	}
}
