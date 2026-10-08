package search

import (
	"strings"
	"testing"
)

func TestMarkdownIncludesPermalink(t *testing.T) {
	// Given
	result := Result{
		Query: "hello",
		Hits: []Hit{{
			SpaceID:      "spaceidxxx1",
			TopicID:      "1789000000001000",
			ThreadWebID:  "threadweb01",
			MessageWebID: "messageweb1",
			Permalink:    "https://chat.google.com/room/spaceidxxx1/threadweb01/messageweb1",
			Text:         "hit-text-01",
		}},
	}

	// When
	got := Markdown(result)

	// Then
	for _, expected := range []string{"Hits: 1", "https://chat.google.com/room/spaceidxxx1/threadweb01/messageweb1", "hit-text-01"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("markdown missing %q:\n%s", expected, got)
		}
	}
}
