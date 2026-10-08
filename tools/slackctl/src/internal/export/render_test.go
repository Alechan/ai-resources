package export

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/slack"
)

func TestRenderMarkdownSlackTextAndThreads(t *testing.T) {
	doc := Document{
		SchemaVersion: 1,
		Conversation:  ConversationDTO{ID: "C22222222", Type: "private_channel"},
		Participants: map[string]Participant{
			"U11111111": {DisplayName: "Example *One*"},
		},
		Messages: []MessageDTO{{
			Timestamp:  "100.000001",
			Datetime:   "1970-01-01T00:01:40.000001Z",
			AuthorID:   "U11111111",
			AuthorName: "Example *One*",
			Text:       "hello <https://docs.example|guide>\n<@U11111111> *literal*",
			ThreadReplies: []MessageDTO{{
				Timestamp:  "101.000001",
				Datetime:   "1970-01-01T00:01:41.000001Z",
				AuthorID:   "U99999999",
				AuthorName: "U99999999 (unresolved)",
				Text:       "reply\nline",
			}},
		}},
	}
	got := RenderMarkdown(doc)
	for _, expected := range []string{
		"# Slack conversation export — C22222222",
		"[guide](https://docs.example)",
		"**1970-01-01 00:01:40 UTC — Example \\*One\\***",
		"@Example \\*One\\* \\*literal\\*",
		"> **1970-01-01 00:01:41 UTC — U99999999 (unresolved)**",
		"> reply\n> line",
	} {
		if !strings.Contains(got, expected) {
			t.Errorf("expected %q in:\n%s", expected, got)
		}
	}
}

func TestNormalizeUsesAttachmentFallbackAndStableJSON(t *testing.T) {
	messages := []RawMessage{{
		Timestamp: "100.000001",
		BotID:     "B11111111",
		Username:  "Example Bot",
		Attachments: []RawAttachment{{
			Fallback: "<tag> & fallback text",
		}},
	}}
	doc := Normalize("C22222222", "channel", messages, nil, map[string]Participant{})
	if doc.Messages[0].Text != "<tag> & fallback text" || doc.Messages[0].AuthorName != "Example Bot" {
		t.Fatalf("document = %#v", doc)
	}

	first, err := MarshalDocument(doc)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := MarshalDocument(doc)
	if string(first) != string(second) || !strings.HasSuffix(string(first), "\n") {
		t.Fatal("normalized JSON is not byte-stable")
	}
	for _, expected := range []string{`"schema_version": 2`, `"text": "<tag> & fallback text"`, `"reactions": []`} {
		if !strings.Contains(string(first), expected) {
			t.Errorf("normalized JSON missing %q:\n%s", expected, first)
		}
	}
}

func TestNormalizeReactions(t *testing.T) {
	root := RawMessage{
		Timestamp: "100.000001",
		Reactions: []slack.Reaction{
			{Name: "z_custom-party", Count: 4, Users: []string{"U33333333", "U22222222", "U33333333"}},
			{Name: "eyes", Count: 1, Users: []string{"U11111111"}},
		},
	}
	reply := RawMessage{
		Timestamp: "101.000001",
		ThreadTS:  root.Timestamp,
		Reactions: []slack.Reaction{{Name: "ok", Count: 3, Users: []string{"U33333333"}}},
	}
	document := Normalize(
		"C22222222",
		"channel",
		[]RawMessage{root, {Timestamp: "102.000001"}},
		map[string][]RawMessage{root.Timestamp: {reply}},
		map[string]Participant{},
	)

	if document.SchemaVersion != 2 {
		t.Fatalf("schema version = %d, want 2", document.SchemaVersion)
	}
	wantRoot := ReactionList{
		{Name: "eyes", Count: 1, UserIDs: []string{"U11111111"}},
		{Name: "z_custom-party", Count: 4, UserIDs: []string{"U22222222", "U33333333"}},
	}
	if !reflect.DeepEqual(document.Messages[0].Reactions, wantRoot) {
		t.Fatalf("root reactions = %#v, want %#v", document.Messages[0].Reactions, wantRoot)
	}
	wantReply := ReactionList{{Name: "ok", Count: 3, UserIDs: []string{"U33333333"}}}
	if !reflect.DeepEqual(document.Messages[0].ThreadReplies[0].Reactions, wantReply) {
		t.Fatalf("reply reactions = %#v, want %#v", document.Messages[0].ThreadReplies[0].Reactions, wantReply)
	}
	if document.Messages[1].Reactions == nil || document.Messages[0].ThreadReplies[0].ThreadReplies == nil {
		t.Fatal("message slices must encode as non-null arrays")
	}
}

func TestValidateDetectsRangeAndMissingThread(t *testing.T) {
	manifest := Manifest{RequestedFrom: stringPointer("1970-01-01T00:01:39Z"), RequestedTo: "1970-01-01T00:01:50Z"}
	doc := Document{Messages: []MessageDTO{{Timestamp: "100.000001", ThreadReplyCount: 1}}}
	if err := Validate(manifest, doc, map[string]bool{}); err == nil {
		t.Fatal("expected missing thread validation error")
	}
	doc.Messages[0].ThreadReplyCount = 0
	doc.Messages[0].Timestamp = "120.000001"
	if err := Validate(manifest, doc, map[string]bool{}); err == nil {
		t.Fatal("expected out-of-range validation error")
	}
}

func TestValidateDetectsIncorrectCountsAndReplyRange(t *testing.T) {
	manifest := Manifest{
		RequestedTo:      "1970-01-01T00:01:50Z",
		RootMessageCount: 2,
	}
	doc := Document{Messages: []MessageDTO{{
		Timestamp:        "100.000001",
		ThreadReplyCount: 1,
		ThreadReplies: []MessageDTO{{
			Timestamp: "120.000001",
		}},
	}}}
	if err := Validate(manifest, doc, map[string]bool{"100.000001": true}); err == nil {
		t.Fatal("expected count or reply range error")
	}
}

func TestManifestPreservesPermalinkMicrosecondsAndLateReplyBounds(t *testing.T) {
	options := Options{
		Range: TimeRange{
			From:       time.Unix(1786455295, 71869000).UTC(),
			To:         time.Unix(1786456000, 0).UTC(),
			FromSource: RangeFromPermalink,
		},
	}
	manifest := newManifest(options, time.Unix(1786456000, 0).UTC())
	if manifest.RequestedFrom == nil || *manifest.RequestedFrom != "2026-08-11T13:34:55.071869Z" {
		t.Fatalf("requested_from = %#v", manifest.RequestedFrom)
	}
	document := Document{
		Messages: []MessageDTO{{
			Timestamp:        "1786455295.071869",
			ThreadReplyCount: 1,
			ThreadReplies: []MessageDTO{{
				Timestamp: "1786457000.000001",
			}},
		}},
		Participants: map[string]Participant{},
	}
	populateManifest(&manifest, document, true)
	if manifest.NewestExported != "2026-08-11T14:03:20.000001Z" ||
		manifest.RootMessageCount != 1 ||
		manifest.ThreadReplyCount != 1 ||
		!manifest.Complete {
		t.Fatalf("manifest = %#v", manifest)
	}
}

func TestValidateAppliesRequestedRangeOnlyToRoots(t *testing.T) {
	manifest := Manifest{
		RequestedFrom:    stringPointer("1970-01-01T00:01:39.000001Z"),
		RequestedTo:      "1970-01-01T00:01:45Z",
		RootMessageCount: 1,
		ThreadReplyCount: 2,
	}
	document := Document{
		Participants: map[string]Participant{},
		Messages: []MessageDTO{{
			Timestamp:        "100.000001",
			ThreadReplyCount: 2,
			ThreadReplies: []MessageDTO{
				{Timestamp: "89.000001"},
				{Timestamp: "110.000001"},
			},
		}},
	}
	if err := Validate(manifest, document, map[string]bool{"100.000001": true}); err != nil {
		t.Fatalf("complete thread outside root range should be valid: %v", err)
	}
	document.Messages[0].Timestamp = "110.000001"
	if err := Validate(manifest, document, map[string]bool{"110.000001": true}); err == nil || !strings.Contains(err.Error(), "message") {
		t.Fatalf("root outside range error = %v", err)
	}
}

func TestRenderMarkdownReactions(t *testing.T) {
	document := Document{
		SchemaVersion: 2,
		Conversation:  ConversationDTO{ID: "C22222222", Type: "channel"},
		Participants: map[string]Participant{
			"U11111111": {DisplayName: "Name *One* [A] > `admin` ~x~"},
			"U22222222": {DisplayName: "Name Two"},
			"U99999999": {DisplayName: "U99999999", Unresolved: true},
		},
		Messages: []MessageDTO{{
			Timestamp:  "100.000001",
			Datetime:   "1970-01-01T00:01:40.000001Z",
			AuthorName: "Example Author",
			Text:       "root text",
			Reactions: []ReactionDTO{
				{Name: "ok", Count: 3, UserIDs: []string{}},
				{Name: "custom_*[x]`~<tag>", Count: 4, UserIDs: []string{"U22222222", "U11111111"}},
			},
			ThreadReplies: []MessageDTO{{
				Timestamp:  "101.000001",
				Datetime:   "1970-01-01T00:01:41.000001Z",
				AuthorName: "Reply Author",
				Text:       "reply text",
				Reactions: []ReactionDTO{{
					Name: "eyes", Count: 2, UserIDs: []string{"U99999999"},
				}},
				ThreadReplies: []MessageDTO{},
			}},
		}},
	}

	got := RenderMarkdown(document)
	for _, expected := range []string{
		"root text\n\n_Reactions: :custom\\_\\*\\[x\\]\\`\\~\\<tag\\>: x4 - returned users: Name \\*One\\* \\[A\\] \\> \\`admin\\` \\~x\\~, Name Two; :ok: x3_",
		"> reply text\n> _Reactions: :eyes: x2 - returned users: U99999999 (unresolved)_",
	} {
		if !strings.Contains(got, expected) {
			t.Errorf("expected %q in:\n%s", expected, got)
		}
	}
	if strings.Contains(got, ":ok: x3 - returned users:") {
		t.Fatalf("empty returned users should be omitted:\n%s", got)
	}
}

func TestValidateReactions(t *testing.T) {
	tests := []struct {
		name      string
		reactions []ReactionDTO
		wantErr   string
	}{
		{
			name:      "non-empty name",
			reactions: []ReactionDTO{{Name: "", Count: 1, UserIDs: []string{}}},
			wantErr:   "name",
		},
		{
			name:      "non-negative count",
			reactions: []ReactionDTO{{Name: "ok", Count: -1, UserIDs: []string{}}},
			wantErr:   "count",
		},
		{
			name:      "deduplicated user IDs",
			reactions: []ReactionDTO{{Name: "ok", Count: 2, UserIDs: []string{"U11111111", "U11111111"}}},
			wantErr:   "duplicate",
		},
		{
			name:      "returned users cannot exceed count",
			reactions: []ReactionDTO{{Name: "ok", Count: 1, UserIDs: []string{"U11111111", "U22222222"}}},
			wantErr:   "exceeds",
		},
		{
			name:      "returned users may be fewer than count",
			reactions: []ReactionDTO{{Name: "ok", Count: 3, UserIDs: []string{"U11111111"}}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			document := Document{
				SchemaVersion: 2,
				Participants:  map[string]Participant{},
				Messages: []MessageDTO{{
					Timestamp:     "100.000001",
					Reactions:     tc.reactions,
					ThreadReplies: []MessageDTO{},
				}},
			}
			manifest := Manifest{
				SchemaVersion:    1,
				RequestedTo:      "1970-01-01T00:01:50Z",
				RootMessageCount: 1,
			}
			err := Validate(manifest, document, map[string]bool{})
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() error = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

func stringPointer(value string) *string { return &value }
