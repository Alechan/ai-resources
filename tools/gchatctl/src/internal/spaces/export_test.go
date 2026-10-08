package spaces

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarkdownIncludesIDsAndNames(t *testing.T) {
	// Given
	world := World{Spaces: []Space{
		{ID: "spaceidxxx1", Name: "Space One Name"},
		{ID: "spaceidxxx2"},
	}}

	// When
	got := Markdown(world)

	// Then
	for _, expected := range []string{"Spaces: 2", "`spaceidxxx1`", "Space One Name", "`spaceidxxx2`"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("markdown missing %q:\n%s", expected, got)
		}
	}
}

func TestWriteExportWritesJSONAndMarkdown(t *testing.T) {
	// Given
	directory := filepath.Join(t.TempDir(), "spaces-export")
	world := World{Spaces: []Space{{ID: "spaceidxxx1", Name: "Space One Name"}}}

	// When
	paths, err := WriteExport(directory, world, []string{"json", "markdown"})

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("paths = %v", paths)
	}
	jsonData, err := os.ReadFile(filepath.Join(directory, "spaces.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(jsonData), "Space One Name") {
		t.Fatalf("json = %s", jsonData)
	}
	markdown, err := os.ReadFile(filepath.Join(directory, "spaces.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdown), "Space One Name") {
		t.Fatalf("markdown = %s", markdown)
	}
}
