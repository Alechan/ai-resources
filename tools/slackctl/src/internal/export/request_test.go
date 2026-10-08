package export

import (
	"strings"
	"testing"
	"time"
)

func TestParseConversation(t *testing.T) {
	tests := []struct {
		name, input, workspace string
		want                   ConversationRef
		wantErr                string
	}{
		{"app URL", "https://app.slack.com/client/T11111111/C22222222", "", ConversationRef{WorkspaceID: "T11111111", ConversationID: "C22222222", Kind: ConversationRefAppURL}, ""},
		{"bare public channel", "C22222222", "", ConversationRef{ConversationID: "C22222222", Kind: ConversationRefID}, ""},
		{"public channel permalink", "https://example.slack.com/archives/C22222222/p1786455295071869", "", ConversationRef{WorkspaceHost: "example.slack.com", ConversationID: "C22222222", StartingTimestamp: "1786455295.071869", Kind: ConversationRefPermalink}, ""},
		{"private channel permalink", "https://example.slack.com/archives/G22222222/p1786455295071869", "", ConversationRef{WorkspaceHost: "example.slack.com", ConversationID: "G22222222", StartingTimestamp: "1786455295.071869", Kind: ConversationRefPermalink}, ""},
		{"direct message permalink", "https://example.slack.com/archives/D22222222/p1786455295071869", "", ConversationRef{WorkspaceHost: "example.slack.com", ConversationID: "D22222222", StartingTimestamp: "1786455295.071869", Kind: ConversationRefPermalink}, ""},
		{"permalink query and fragment", "https://example.slack.com/archives/C22222222/p1786455295071869?foo=bar#message", "", ConversationRef{WorkspaceHost: "example.slack.com", ConversationID: "C22222222", StartingTimestamp: "1786455295.071869", Kind: ConversationRefPermalink}, ""},
		{"workspace conflict", "https://app.slack.com/client/T11111111/C22222222", "T33333333", ConversationRef{}, "workspace"},
		{"non Slack host", "https://invalid.example/archives/C22222222/p1786455295071869", "", ConversationRef{}, "Slack"},
		{"root Slack host", "https://slack.com/archives/C22222222/p1786455295071869", "", ConversationRef{}, "Slack"},
		{"missing conversation ID", "https://example.slack.com/archives//p1786455295071869", "", ConversationRef{}, "malformed"},
		{"malformed conversation ID", "https://example.slack.com/archives/X22222222/p1786455295071869", "", ConversationRef{}, "malformed"},
		{"missing timestamp", "https://example.slack.com/archives/C22222222", "", ConversationRef{}, "malformed"},
		{"short timestamp", "https://example.slack.com/archives/C22222222/p123456", "", ConversationRef{}, "timestamp"},
		{"non numeric timestamp", "https://example.slack.com/archives/C22222222/p1786455295abcdef", "", ConversationRef{}, "timestamp"},
		{"double path separator", "https://example.slack.com//archives/C22222222/p1786455295071869", "", ConversationRef{}, "malformed"},
		{"trailing path separator", "https://example.slack.com/archives/C22222222/p1786455295071869/", "", ConversationRef{}, "malformed"},
		{"encoded path separator", "https://example.slack.com/archives/C22222222%2Fp1786455295071869", "", ConversationRef{}, "malformed"},
		{"reply permalink", "https://example.slack.com/archives/C22222222/p1786455295071869?thread_ts=1786455000.000001", "", ConversationRef{}, "root-message permalink"},
		{"malformed", "not-an-id", "", ConversationRef{}, "conversation"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseConversation(tc.input, tc.workspace)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("got %#v, %v; want %#v", got, err, tc.want)
			}
		})
	}
}

func TestNormalizeRequest(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if _, err := NormalizeRequest(RequestInput{}, now); err == nil {
		t.Fatal("either --all or --from should be required")
	}

	_, err := NormalizeRequest(RequestInput{All: true, From: "2026-01-01T00:00:00Z"}, now)
	if err == nil {
		t.Fatal("--all and --from should conflict")
	}
	got, err := NormalizeRequest(RequestInput{From: "now-2d", To: "now-1h"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !got.From.Equal(now.Add(-48*time.Hour)) || !got.To.Equal(now.Add(-time.Hour)) {
		t.Fatalf("range = %s to %s", got.From, got.To)
	}
	if _, err := NormalizeRequest(RequestInput{From: "2026-01-03T00:00:00Z", To: "2026-01-02T00:00:00Z"}, now); err == nil {
		t.Fatal("reversed range should fail")
	}
}

func TestNormalizeRequest_PermalinkLowerBoundary(t *testing.T) {
	now := time.Date(2026, 8, 12, 9, 32, 6, 437000000, time.UTC)
	tests := []struct {
		name    string
		input   RequestInput
		wantTo  time.Time
		wantErr string
	}{
		{
			name:   "permalink freezes upper bound at command start",
			input:  RequestInput{PermalinkFrom: "1786455295.071869"},
			wantTo: now,
		},
		{
			name:   "permalink allows explicit upper bound",
			input:  RequestInput{PermalinkFrom: "1786455295.071869", To: "2026-08-12T10:00:00Z"},
			wantTo: time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC),
		},
		{
			name:    "permalink rejects all",
			input:   RequestInput{PermalinkFrom: "1786455295.071869", All: true},
			wantErr: "--all",
		},
		{
			name:    "permalink rejects explicit from",
			input:   RequestInput{PermalinkFrom: "1786455295.071869", From: "now-1d"},
			wantErr: "--from",
		},
		{
			name:    "permalink lower boundary must precede to",
			input:   RequestInput{PermalinkFrom: "1786455295.071869", To: "2026-08-11T13:00:00Z"},
			wantErr: "permalink",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeRequest(tc.input, now)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			wantFrom := time.Unix(1786455295, 71869000).UTC()
			if !got.From.Equal(wantFrom) || got.From.Nanosecond() != 71869000 {
				t.Fatalf("from = %s (%d ns), want %s", got.From, got.From.Nanosecond(), wantFrom)
			}
			if !got.To.Equal(tc.wantTo) {
				t.Fatalf("to = %s, want %s", got.To, tc.wantTo)
			}
			if got.FromSource != RangeFromPermalink {
				t.Fatalf("from source = %q", got.FromSource)
			}
		})
	}
}
