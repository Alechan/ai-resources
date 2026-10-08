package spaces

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/topics"
)

func TestParseWorldFindsNamedSpaces(t *testing.T) {
	// Given
	raw := []byte(`)]}'

[["dfe.pw.pw",[[
  [[null,"spaceidxxx1"],null,"1","2",null,null,null,null,null,"Space One Name",null,null,null,"spaceidxxx1"],
  [[null,"spaceidxxx2"],null,"1","2",null,null,null,null,null,"Space Two Name",null,null,null,"spaceidxxx2"]
]]]]`)

	// When
	world, err := Parse(raw)

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if world.RPC != "dfe.pw.pw" || len(world.Spaces) != 2 {
		t.Fatalf("world = %#v", world)
	}
	if world.Spaces[0].ID != "spaceidxxx1" || world.Spaces[0].Name != "Space One Name" {
		t.Fatalf("first = %#v", world.Spaces[0])
	}
	if world.Spaces[1].ID != "spaceidxxx2" {
		t.Fatalf("second = %#v", world.Spaces[1])
	}
	if strings.Contains(topics.Shape(raw), "Space One Name") {
		t.Fatal("shape leaked space name")
	}
}

func TestParseWorldKeepsIDWhenNameMissing(t *testing.T) {
	raw := []byte(`[["dfe.pw.pw",[[[null,"spaceidxxx3"]]]]]`)
	world, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Spaces) != 1 || world.Spaces[0].ID != "spaceidxxx3" || world.Spaces[0].Name != "" {
		t.Fatalf("world = %#v", world)
	}
}

func TestParseWorldEmpty(t *testing.T) {
	if _, err := Parse(nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseGroupReadsNamedSpaceRecord(t *testing.T) {
	// Given
	raw := groupResponse([]any{[]any{[]any{[]any{[]any{"spaceidxxx1"}}}}, "Space One Name", nil, nil, "1789000000001000"})

	// When
	group, err := ParseGroup(raw)

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if group.RPC != "dfe.g.gg" || group.ID != "spaceidxxx1" || group.Name != "Space One Name" {
		t.Fatalf("group = %#v", group)
	}
	if strings.Contains(topics.Shape(raw), "Space One Name") {
		t.Fatal("shape leaked space name")
	}
}

func TestParseGroupKeepsIDWhenNameMissing(t *testing.T) {
	// Given
	row := []any{[]any{[]any{[]any{"123456789012345678901"}}, nil, []any{nil, nil, []any{"spaceidxxx4"}}}}
	raw := groupResponse([]any{row})

	// When
	group, err := ParseGroup(raw)

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if group.ID != "spaceidxxx4" || group.Name != "" {
		t.Fatalf("group = %#v", group)
	}
}

func TestParseGroupEmpty(t *testing.T) {
	if _, err := ParseGroup(nil); err == nil {
		t.Fatal("expected error")
	}
}

func groupResponse(record []any) []byte {
	payload, err := json.Marshal([]any{[]any{"dfe.g.gg", record}})
	if err != nil {
		panic(err)
	}
	return append([]byte(")]}'\n\n"), payload...)
}
