package topics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarkdownAndWriteExport(t *testing.T) {
	// Given
	page := Page{
		RPC:      "dfe.t.lt",
		Complete: true,
		Topics: []Topic{{
			ID:      "1788269183261582",
			SpaceID: "spaceidxxxx1",
			Messages: []Message{{
				ID:       "messageidxxxxxx",
				AuthorID: "authoridxxx",
				Text:     "Deployment finished with NPS notes",
			}},
		}},
	}
	directory := t.TempDir()

	// When
	paths, err := WriteExport(directory, page, []string{"json", "markdown"}, "spaceidxxxx1", "nps")
	if err != nil {
		t.Fatal(err)
	}

	// Then
	if len(paths) != 2 {
		t.Fatalf("paths = %v", paths)
	}
	markdown, err := os.ReadFile(filepath.Join(directory, "topics.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(markdown)
	if !strings.Contains(text, "Deployment finished with NPS notes") || !strings.Contains(text, "Text filter: `nps`") {
		t.Fatalf("markdown = %s", text)
	}
	jsonBody, err := os.ReadFile(filepath.Join(directory, "topics.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(jsonBody), `"id": "1788269183261582"`) {
		t.Fatalf("json = %s", jsonBody)
	}
}
