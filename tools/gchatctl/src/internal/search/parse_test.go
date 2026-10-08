package search

import (
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/topics"
)

func TestParseHitsFromSyntheticEnvelope(t *testing.T) {
	// Given
	raw := searchResponse("spaceidxxx1", "1789000000001000", "hit-text-01")

	// When
	result, err := Parse(raw)

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if result.RPC != "SBNmJb" || len(result.Hits) != 1 {
		t.Fatalf("result = %#v", result)
	}
	hit := result.Hits[0]
	if hit.SpaceID != "spaceidxxx1" || hit.TopicID != "1789000000001000" || hit.Text != "hit-text-01" {
		t.Fatalf("hit = %#v", hit)
	}
	if strings.Contains(topics.Shape(raw), "hit-text-01") {
		t.Fatal("shape leaked hit text")
	}
	if hit.Permalink != "" {
		t.Fatalf("digit topic fixture should not invent a permalink: %#v", hit)
	}
}

func TestParseHitsReadsPermalinkWebIDs(t *testing.T) {
	// Given
	raw := searchResponseWithWebIDs("spaceidxxx1", "1789000000001000", "threadweb01", "messageweb1", "hit-text-01")

	// When
	result, err := Parse(raw)

	// Then
	if err != nil {
		t.Fatal(err)
	}
	hit := result.Hits[0]
	if hit.ThreadWebID != "threadweb01" || hit.MessageWebID != "messageweb1" {
		t.Fatalf("hit = %#v", hit)
	}
	if hit.Permalink != "https://chat.google.com/room/spaceidxxx1/threadweb01/messageweb1" {
		t.Fatalf("permalink = %q", hit.Permalink)
	}
}

func TestParseHitsReadsTopicIDFromDigitSlotWhenTitleOccupiesSlotTwo(t *testing.T) {
	// Given
	raw := searchTitleAndTopic("spaceidxxx1", "title / not-id", "1789000000001000", "hit-text-01")

	// When
	result, err := Parse(raw)

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if result.Hits[0].TopicID != "1789000000001000" {
		t.Fatalf("hit = %#v", result.Hits[0])
	}
}

func TestParseEmpty(t *testing.T) {
	if _, err := Parse(nil); err == nil {
		t.Fatal("expected error")
	}
}

func searchResponse(spaceID, topicID, text string) []byte {
	return encodeSearchHit(func(hit []any) {
		hit[0] = []any{"space/" + spaceID, spaceID, 1}
		hit[2] = topicID
		hit[48] = text
	})
}

func searchResponseWithWebIDs(spaceID, topicID, threadID, messageID, text string) []byte {
	return encodeSearchHit(func(hit []any) {
		hit[0] = []any{"space/" + spaceID, spaceID, 1}
		hit[2] = topicID
		hit[13] = threadID
		hit[14] = messageID
		hit[48] = text
	})
}

func searchTitleAndTopic(spaceID, title, topicID, text string) []byte {
	return encodeSearchHit(func(hit []any) {
		hit[0] = []any{"space/" + spaceID, spaceID, 1}
		hit[2] = title
		hit[8] = topicID
		hit[48] = text
	})
}

func encodeSearchHit(fill func([]any)) []byte {
	hit := make([]any, 49)
	fill(hit)
	payload, _ := marshal([]any{"", 1, nil, "", []any{hit}})
	row := []any{"wrb.fr", "SBNmJb", payload, nil, nil, nil, "1"}
	outer, _ := marshal([]any{row})
	return []byte(")]}'\n\n" + outer)
}
