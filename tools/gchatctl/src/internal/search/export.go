package search

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Markdown(result Result) string {
	var builder strings.Builder
	builder.WriteString("# Google Chat search\n\n")
	if result.Query != "" {
		fmt.Fprintf(&builder, "Query: `%s`\n\n", result.Query)
	}
	fmt.Fprintf(&builder, "Hits: %d\n\n", len(result.Hits))
	if len(result.Hits) == 0 {
		builder.WriteString("No matching hits.\n")
		return builder.String()
	}
	for _, hit := range result.Hits {
		title := hit.SpaceID
		if hit.TopicID != "" {
			title += " / " + hit.TopicID
		}
		fmt.Fprintf(&builder, "## %s\n\n", title)
		if hit.Permalink != "" {
			fmt.Fprintf(&builder, "Permalink: %s\n\n", hit.Permalink)
		}
		if hit.Text == "" {
			builder.WriteString("_No snippet._\n\n")
			continue
		}
		fmt.Fprintf(&builder, "%s\n\n", hit.Text)
	}
	return builder.String()
}

func WriteExport(directory string, result Result, formats []string) ([]string, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, err
	}
	var paths []string
	for _, format := range formats {
		path, err := writeFormat(directory, result, format)
		if err != nil {
			return paths, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func writeFormat(directory string, result Result, format string) (string, error) {
	var (
		name string
		data []byte
		err  error
	)
	switch format {
	case "json":
		name = "search.json"
		data, err = json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		data = append(data, '\n')
	case "markdown":
		name = "search.md"
		data = []byte(Markdown(result))
	default:
		return "", fmt.Errorf("unsupported format %q", format)
	}
	path := filepath.Join(directory, name)
	return path, os.WriteFile(path, data, 0o600)
}
