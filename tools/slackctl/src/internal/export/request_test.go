package export

import (
	"testing"
	"time"
)

func TestParseConversation(t *testing.T) {
	tests := []struct {
		name, input, workspace string
		want                   ConversationRef
		wantErr                bool
	}{
		{"url", "https://app.slack.com/client/T11111111/C22222222", "", ConversationRef{WorkspaceID: "T11111111", ConversationID: "C22222222"}, false},
		{"bare", "C22222222", "", ConversationRef{ConversationID: "C22222222"}, false},
		{"workspace conflict", "https://app.slack.com/client/T11111111/C22222222", "T33333333", ConversationRef{}, true},
		{"wrong host", "https://invalid.example/client/T11111111/C22222222", "", ConversationRef{}, true},
		{"malformed", "not-an-id", "", ConversationRef{}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseConversation(tc.input, tc.workspace)
			if (err != nil) != tc.wantErr || got != tc.want {
				t.Fatalf("got %#v, %v; want %#v, error=%v", got, err, tc.want, tc.wantErr)
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
