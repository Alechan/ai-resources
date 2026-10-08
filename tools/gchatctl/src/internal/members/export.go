package members

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Markdown(roster Roster) string {
	var builder strings.Builder
	builder.WriteString("# Google Chat members\n\n")
	fmt.Fprintf(&builder, "Members: %d\n\n", len(roster.Members))
	if len(roster.Members) == 0 {
		builder.WriteString("No members.\n")
		return builder.String()
	}
	for _, member := range roster.Members {
		name := member.Name
		if name == "" {
			name = member.ID
		}
		fmt.Fprintf(&builder, "- `%s` %s\n", member.ID, name)
	}
	return builder.String()
}

func WriteExport(directory string, roster Roster, formats []string) ([]string, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, err
	}
	var paths []string
	for _, format := range formats {
		path, err := writeFormat(directory, roster, format)
		if err != nil {
			return paths, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func writeFormat(directory string, roster Roster, format string) (string, error) {
	var (
		name string
		data []byte
		err  error
	)
	switch format {
	case "json":
		name = "members.json"
		data, err = json.MarshalIndent(roster, "", "  ")
		if err != nil {
			return "", err
		}
		data = append(data, '\n')
	case "markdown":
		name = "members.md"
		data = []byte(Markdown(roster))
	default:
		return "", fmt.Errorf("unsupported format %q", format)
	}
	path := filepath.Join(directory, name)
	return path, os.WriteFile(path, data, 0o600)
}
