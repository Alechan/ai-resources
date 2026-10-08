package topics

import (
	"strings"
	"testing"
)

func TestParsePermalink(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Permalink
		wantErr string
	}{
		{
			name:  "room thread message",
			input: "https://chat.google.com/room/AAQAP4TTGjI/qMAc4oTc2i8/qMAc4oTc2i8?cls=10",
			want:  Permalink{SpaceID: "AAQAP4TTGjI", ThreadID: "qMAc4oTc2i8", MessageID: "qMAc4oTc2i8"},
		},
		{
			name:  "thread only",
			input: "https://chat.google.com/room/AAQAP4TTGjI/qMAc4oTc2i8",
			want:  Permalink{SpaceID: "AAQAP4TTGjI", ThreadID: "qMAc4oTc2i8", MessageID: "qMAc4oTc2i8"},
		},
		{
			name:    "space only",
			input:   "https://chat.google.com/room/AAQAP4TTGjI",
			wantErr: "starting point",
		},
		{
			name:    "slack url",
			input:   "https://example.slack.com/archives/C22222222/p1786455295071869",
			wantErr: "chat.google.com",
		},
		{
			name:    "http",
			input:   "http://chat.google.com/room/AAQAP4TTGjI/qMAc4oTc2i8",
			wantErr: "chat.google.com",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// When
			got, err := ParsePermalink(tc.input)

			// Then
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
			if got.StartWebID() != tc.want.MessageID {
				t.Fatalf("StartWebID = %q", got.StartWebID())
			}
		})
	}
}
