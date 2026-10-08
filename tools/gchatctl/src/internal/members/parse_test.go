package members

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/topics"
)

func TestParseRosterJoinsNamesByUserID(t *testing.T) {
	// Given
	raw := rosterResponse([]memberFixture{
		{id: "123456789012345678901", spaceID: "spaceidxxx1", name: "Member One Name"},
		{id: "123456789012345678902", spaceID: "spaceidxxx1", name: "Member Two Name"},
	})

	// When
	roster, err := Parse(raw)

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if roster.RPC != "dfe.lm.lm" || len(roster.Members) != 2 {
		t.Fatalf("roster = %#v", roster)
	}
	if roster.Members[0] != (Member{ID: "123456789012345678901", SpaceID: "spaceidxxx1", Name: "Member One Name"}) {
		t.Fatalf("first = %#v", roster.Members[0])
	}
	if roster.Members[1].ID != "123456789012345678902" || roster.Members[1].Name != "Member Two Name" {
		t.Fatalf("second = %#v", roster.Members[1])
	}
	if strings.Contains(topics.Shape(raw), "Member One Name") {
		t.Fatal("shape leaked member name")
	}
}

func TestParseRosterKeepsIDWhenNameMissing(t *testing.T) {
	// Given
	raw := rosterResponse([]memberFixture{
		{id: "123456789012345678903", spaceID: "spaceidxxx2"},
	})

	// When
	roster, err := Parse(raw)

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if len(roster.Members) != 1 {
		t.Fatalf("roster = %#v", roster)
	}
	if roster.Members[0].ID != "123456789012345678903" || roster.Members[0].SpaceID != "spaceidxxx2" || roster.Members[0].Name != "" {
		t.Fatalf("member = %#v", roster.Members[0])
	}
}

func TestParseRosterSkipsTopicIDs(t *testing.T) {
	// Given
	raw := rosterResponse([]memberFixture{
		{id: "1789000000001000", spaceID: "spaceidxxx1"},
	})

	// When
	roster, err := Parse(raw)

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if len(roster.Members) != 0 {
		t.Fatalf("16-digit topic ids must not become members: %#v", roster)
	}
}

func TestParseRosterEmpty(t *testing.T) {
	if _, err := Parse(nil); err == nil {
		t.Fatal("expected error")
	}
}

type memberFixture struct {
	id      string
	spaceID string
	name    string
}

func rosterResponse(fixtures []memberFixture) []byte {
	var rows []any
	var names []any
	for _, fixture := range fixtures {
		rows = append(rows, []any{
			[]any{[]any{[]any{fixture.id}}, nil, []any{[]any{fixture.spaceID}}},
			"1789000000001000",
			1,
			nil,
			1,
		})
		if fixture.name != "" {
			names = append(names, []any{fixture.name, []any{fixture.id}})
		}
	}
	inner := []any{"dfe.lm.lm", rows, nil, "", nil, nil, nil, names}
	payload, err := json.Marshal([]any{inner})
	if err != nil {
		panic(err)
	}
	return append([]byte(")]}'\n\n"), payload...)
}
