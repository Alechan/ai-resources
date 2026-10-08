package spaces

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Markdown(world World) string {
	var builder strings.Builder
	builder.WriteString("# Google Chat spaces\n\n")
	fmt.Fprintf(&builder, "Spaces: %d\n\n", len(world.Spaces))
	if len(world.Spaces) == 0 {
		builder.WriteString("No spaces.\n")
		return builder.String()
	}
	for _, space := range world.Spaces {
		name := space.Name
		if name == "" {
			name = space.ID
		}
		fmt.Fprintf(&builder, "- `%s` %s\n", space.ID, name)
	}
	return builder.String()
}

func WriteExport(directory string, world World, formats []string) ([]string, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, err
	}
	var paths []string
	for _, format := range formats {
		path, err := writeFormat(directory, world, format)
		if err != nil {
			return paths, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func writeFormat(directory string, world World, format string) (string, error) {
	var (
		name string
		data []byte
		err  error
	)
	switch format {
	case "json":
		name = "spaces.json"
		data, err = json.MarshalIndent(world, "", "  ")
		if err != nil {
			return "", err
		}
		data = append(data, '\n')
	case "markdown":
		name = "spaces.md"
		data = []byte(Markdown(world))
	default:
		return "", fmt.Errorf("unsupported format %q", format)
	}
	path := filepath.Join(directory, name)
	return path, os.WriteFile(path, data, 0o600)
}
