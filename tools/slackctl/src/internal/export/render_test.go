package export

import (
	"strings"
	"testing"
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
			Fallback: "fallback text",
		}},
	}}
	doc := Normalize("C22222222", "channel", messages, nil, map[string]Participant{})
	if doc.Messages[0].Text != "fallback text" || doc.Messages[0].AuthorName != "Example Bot" {
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

func stringPointer(value string) *string { return &value }
