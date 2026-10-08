package topics

import (
	"strings"
	"testing"
	"time"
)

func TestStripXSSI(t *testing.T) {
	// Given
	input := []byte(")]}'\n\n[[\"dfe.t.lt\"]]")

	// When
	got := StripXSSI(input)

	// Then
	if string(got) != `[["dfe.t.lt"]]` {
		t.Fatalf("StripXSSI = %q", got)
	}
}

func TestParseTopicsAndFilter(t *testing.T) {
	// Given
	raw := []byte(`)]}'

[["dfe.t.lt",[[
  [null,"spaceidxxxx1",[null,null,["spaceidxxxx1"]]],
  "1788269183261582",
  null,null,null,null,
  [[
    [[null,null,null,null],"authoridxxx"],
    "ignored",
    "messageidxxxxxx",
    "messageidxxxxxx",
    null,null,null,null,null,
    "Deployment finished with NPS notes",
    null,null,null,
    "webidxxxx01"
  ]]
]]]]`)

	// When
	page, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	// Then
	if page.RPC != "dfe.t.lt" || len(page.Topics) != 1 {
		t.Fatalf("page = %#v", page)
	}
	topic := page.Topics[0]
	if topic.ID != "1788269183261582" || topic.SpaceID != "spaceidxxxx1" {
		t.Fatalf("topic = %#v", topic)
	}
	wantTime := time.UnixMicro(1788269183261582).UTC()
	if !topic.CreatedAt.Equal(wantTime) {
		t.Fatalf("created = %s, want %s", topic.CreatedAt, wantTime)
	}
	if len(topic.Messages) != 1 {
		t.Fatalf("messages = %#v", topic.Messages)
	}
	message := topic.Messages[0]
	if message.AuthorID != "authoridxxx" || message.Text != "Deployment finished with NPS notes" || message.ID != "messageidxxxxxx" || message.WebID != "webidxxxx01" {
		t.Fatalf("message = %#v", message)
	}

	filtered := Filter(page, "nps")
	if len(filtered.Topics) != 1 {
		t.Fatalf("filter nps = %#v", filtered)
	}
	if empty := Filter(page, "absent-term"); len(empty.Topics) != 0 {
		t.Fatalf("filter absent = %#v", empty)
	}
	if bySpace := FilterSpace(page, "spaceidxxxx1"); len(bySpace.Topics) != 1 {
		t.Fatalf("filter space = %#v", bySpace)
	}
	if missing := Apply(page, "other-space", "nps"); len(missing.Topics) != 0 {
		t.Fatalf("apply missing space = %#v", missing)
	}
}

func TestParseRejectsMalformedResponse(t *testing.T) {
	if _, err := Parse([]byte("not-json")); err == nil || !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseNumericTopicIDs(t *testing.T) {
	// Given
	raw := []byte(`[["dfe.t.lt",[[[null,"spaceidxxxx1"],1788269183261582,null,null,null,null,[[[[null,null,null,null],"authoridxxx"],"ignored",1788269183261583,1788269183261583,null,null,null,null,null,"numeric ids"]]]]]]`)

	// When
	page, err := Parse(raw)

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Topics) != 1 || page.Topics[0].ID != "1788269183261582" {
		t.Fatalf("page = %#v", page)
	}
	if page.Topics[0].Messages[0].ID != "1788269183261583" {
		t.Fatalf("message = %#v", page.Topics[0].Messages[0])
	}
}

func TestShapeOmitsText(t *testing.T) {
	raw := []byte(`)]}'

[["dfe.t.lt",[[
  [null,"spaceidxxxx1"],
  "1788269183261582",
  null,null,null,null,
  [[[[null],"authoridxxx"],"ignored","messageidxxxxxx","messageidxxxxxx",null,null,null,null,null,"secret text here"]]
]]]]`)
	got := Shape(raw)
	if strings.Contains(got, "secret") || strings.Contains(got, "dfe.t.lt") {
		t.Fatalf("shape leaked content: %s", got)
	}
	if !strings.Contains(got, "prefix=xssi") || !strings.Contains(got, "str(") {
		t.Fatalf("shape = %s", got)
	}
}

func TestParseFindsTopicsWhenLeadingSlotIsNull(t *testing.T) {
	// Given
	raw := []byte(`[["dfe.t.lt",null,[[[null,"spaceidxxxx1"],"1788269183261582",null,null,null,null,[[[[null,null,null,null],"authoridxxx"],"ignored","messageidxxxxxx","messageidxxxxxx",null,null,null,null,null,"shifted slot",null,null,null,"webidxxxx01"]]]]]]`)

	// When
	page, err := Parse(raw)

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Topics) != 1 || page.Topics[0].ID != "1788269183261582" {
		t.Fatalf("page = %#v", page)
	}
}
