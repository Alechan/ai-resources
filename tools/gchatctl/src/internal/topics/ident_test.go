package topics

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRewriteSpaceIDsReplacesIdentAndKeepsBlob(t *testing.T) {
	// Given
	raw := hundredSlotBody(4, []any{[]any{"oldspaceid1"}}, []any{"keep-blob", "oldspaceid1"})

	// When
	got, err := RewriteSpaceIDs(raw, "spaceidxxx1")

	// Then
	if err != nil {
		t.Fatal(err)
	}
	var body []any
	if err := json.Unmarshal([]byte(got), &body); err != nil {
		t.Fatal(err)
	}
	slot, _ := json.Marshal(body[4])
	if !strings.Contains(string(slot), `"spaceidxxx1"`) || strings.Contains(string(slot), "oldspaceid1") {
		t.Fatalf("space slot = %s", slot)
	}
	blob, _ := json.Marshal(body[99])
	if !strings.Contains(string(blob), `"oldspaceid1"`) || strings.Contains(string(blob), "spaceidxxx1") {
		t.Fatalf("blob rewritten: %s", blob)
	}
}

func TestRewriteSpaceIDsLeavesUserIDs(t *testing.T) {
	// Given
	ident := []any{[]any{[]any{"123456789012345678901"}}, nil, []any{[]any{"oldspaceid1"}}}
	raw := hundredSlotBody(0, ident, []any{1, 2, 3})

	// When
	got, err := RewriteSpaceIDs(raw, "spaceidxxx1")

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"123456789012345678901"`) {
		t.Fatalf("user id rewritten: %s", got)
	}
	if strings.Contains(got, "oldspaceid1") {
		t.Fatal("space id was not replaced")
	}
}

func TestRewriteSpaceIDsRequiresSpace(t *testing.T) {
	if _, err := RewriteSpaceIDs("[null]", ""); err == nil {
		t.Fatal("expected error")
	}
	if _, err := RewriteSpaceIDs("[null,1]", "spaceidxxx1"); err == nil {
		t.Fatal("expected missing ident error")
	}
}

func hundredSlotBody(slot int, value, blob any) string {
	body := make([]any, 100)
	body[slot] = value
	body[99] = blob
	encoded, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}
