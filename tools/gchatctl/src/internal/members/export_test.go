package members

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarkdownIncludesIDsAndNames(t *testing.T) {
	// Given
	roster := Roster{Members: []Member{
		{ID: "123456789012345678901", SpaceID: "spaceidxxx1", Name: "Member One Name"},
		{ID: "123456789012345678902", SpaceID: "spaceidxxx1"},
	}}

	// When
	got := Markdown(roster)

	// Then
	for _, expected := range []string{"Members: 2", "`123456789012345678901`", "Member One Name", "`123456789012345678902`"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("markdown missing %q:\n%s", expected, got)
		}
	}
}

func TestWriteExportWritesJSONAndMarkdown(t *testing.T) {
	// Given
	directory := filepath.Join(t.TempDir(), "members-export")
	roster := Roster{Members: []Member{{ID: "123456789012345678901", Name: "Member One Name"}}}

	// When
	paths, err := WriteExport(directory, roster, []string{"json", "markdown"})

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("paths = %v", paths)
	}
	jsonData, err := os.ReadFile(filepath.Join(directory, "members.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(jsonData), "Member One Name") {
		t.Fatalf("json = %s", jsonData)
	}
	markdown, err := os.ReadFile(filepath.Join(directory, "members.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdown), "Member One Name") {
		t.Fatalf("markdown = %s", markdown)
	}
}
