package topics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Markdown(page Page, spaceID, query string) string {
	var builder strings.Builder
	builder.WriteString("# Google Chat topics\n\n")
	if spaceID != "" {
		fmt.Fprintf(&builder, "Space filter: `%s`\n\n", spaceID)
	}
	if query != "" {
		fmt.Fprintf(&builder, "Text filter: `%s`\n\n", query)
	}
	topicCount, messageCount := Counts(page)
	fmt.Fprintf(&builder, "Topics: %d  \nMessages: %d\n\n", topicCount, messageCount)
	if topicCount == 0 {
		builder.WriteString("No matching topics.\n")
		return builder.String()
	}
	for _, topic := range page.Topics {
		title := topic.ID
		if topic.SpaceID != "" {
			title = topic.SpaceID + " / " + topic.ID
		}
		fmt.Fprintf(&builder, "## %s\n\n", title)
		if !topic.CreatedAt.IsZero() {
			fmt.Fprintf(&builder, "Created: %s\n\n", topic.CreatedAt.UTC().Format(time.RFC3339))
		}
		if len(topic.Messages) == 0 {
			builder.WriteString("_No messages._\n\n")
			continue
		}
		for _, message := range topic.Messages {
			author := message.AuthorID
			if author == "" {
				author = "unknown"
			}
			stamp := ""
			if !message.CreatedAt.IsZero() {
				stamp = message.CreatedAt.UTC().Format(time.RFC3339) + " "
			}
			fmt.Fprintf(&builder, "- %s%s: %s\n", stamp, author, message.Text)
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

func WriteExport(directory string, page Page, formats []string, spaceID, query string) ([]string, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, err
	}
	var paths []string
	for _, format := range formats {
		path, err := writeFormat(directory, page, format, spaceID, query)
		if err != nil {
			return paths, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func writeFormat(directory string, page Page, format, spaceID, query string) (string, error) {
	var (
		name string
		data []byte
		err  error
	)
	switch format {
	case "json":
		name = "topics.json"
		data, err = json.MarshalIndent(page, "", "  ")
		if err != nil {
			return "", err
		}
		data = append(data, '\n')
	case "markdown":
		name = "topics.md"
		data = []byte(Markdown(page, spaceID, query))
	default:
		return "", fmt.Errorf("unsupported format %q", format)
	}
	path := filepath.Join(directory, name)
	return path, os.WriteFile(path, data, 0o600)
}
