package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

// NormalizeNotebookEnvelope accepts:
// 1) {"data":{"type":"notebooks","attributes":...}}
// 2) {"attributes":...}
// and always returns shape (1) in a mutable map form.
func NormalizeNotebookEnvelope(raw []byte) (map[string]any, error) {
	var env map[string]any
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fail.NewValidation("invalid JSON notebook file", "ensure the file contains valid JSON")
	}

	// If only attributes are provided, wrap them.
	if _, hasData := env["data"]; !hasData {
		attrs, ok := env["attributes"].(map[string]any)
		if !ok {
			return nil, fail.NewValidation("missing notebook attributes", `expected "data.attributes" or top-level "attributes"`)
		}
		env = map[string]any{
			"data": map[string]any{
				"type":       "notebooks",
				"attributes": attrs,
			},
		}
	}

	data, ok := env["data"].(map[string]any)
	if !ok {
		return nil, fail.NewValidation(`invalid "data" object`, `expected "data" to be an object`)
	}
	if _, ok := data["attributes"].(map[string]any); !ok {
		return nil, fail.NewValidation(`missing "data.attributes"`, `expected "data.attributes" to be an object`)
	}
	data["type"] = "notebooks"
	return env, nil
}

// PrepareNotebookCreatePayload normalizes envelope and applies optional --name/--time overrides.
func PrepareNotebookCreatePayload(env map[string]any, nameOverride, timeOverride string) (map[string]any, error) {
	data := mustMap(env["data"])
	if data == nil {
		return nil, fail.NewValidation(`missing "data"`, `expected envelope with "data" object`)
	}
	attrs := mustMap(data["attributes"])
	if attrs == nil {
		return nil, fail.NewValidation(`missing "data.attributes"`, `expected envelope with "data.attributes" object`)
	}

	if nameOverride != "" {
		attrs["name"] = nameOverride
	}
	if timeOverride != "" {
		attrs["time"] = map[string]any{"live_span": timeOverride}
	}

	if name, _ := attrs["name"].(string); name == "" {
		return nil, fail.NewValidation(`missing "attributes.name"`, "provide --name or set attributes.name in the file")
	}
	if _, ok := attrs["time"].(map[string]any); !ok {
		return nil, fail.NewValidation(`missing "attributes.time"`, "set attributes.time (e.g. {\"live_span\":\"1w\"})")
	}
	cells, ok := attrs["cells"].([]any)
	if !ok || len(cells) == 0 {
		return nil, fail.NewValidation(`missing "attributes.cells"`, "set attributes.cells to a non-empty array")
	}
	if _, err := PrepareNotebookSchema(env); err != nil {
		return nil, err
	}

	delete(data, "id")
	data["type"] = "notebooks"
	return map[string]any{"data": data}, nil
}

// PrepareNotebookUpdatePayload normalizes envelope, enforces full replacement safety, and sets notebook ID.
func PrepareNotebookUpdatePayload(env map[string]any, notebookID string, replaceAll bool) (map[string]any, error) {
	if !replaceAll {
		return nil, fail.NewValidation("--replace-all is required", "update is full replacement; pass --replace-all to confirm")
	}
	id, err := strconv.ParseInt(notebookID, 10, 64)
	if err != nil {
		return nil, fail.NewValidation("notebook ID must be a number", "usage: ddctl notebooks update <id> --from-file <path> --replace-all")
	}

	data := mustMap(env["data"])
	if data == nil {
		return nil, fail.NewValidation(`missing "data"`, `expected envelope with "data" object`)
	}
	attrs := mustMap(data["attributes"])
	if attrs == nil {
		return nil, fail.NewValidation(`missing "data.attributes"`, `expected envelope with "data.attributes" object`)
	}
	if name, _ := attrs["name"].(string); name == "" {
		return nil, fail.NewValidation(`missing "attributes.name"`, "set attributes.name before update")
	}
	if _, ok := attrs["time"].(map[string]any); !ok {
		return nil, fail.NewValidation(`missing "attributes.time"`, "set attributes.time before update")
	}
	cells, ok := attrs["cells"].([]any)
	if !ok || len(cells) == 0 {
		return nil, fail.NewValidation(`missing "attributes.cells"`, "set attributes.cells to a non-empty array before update")
	}
	if _, err := PrepareNotebookSchema(env); err != nil {
		return nil, err
	}

	data["type"] = "notebooks"
	data["id"] = id
	return map[string]any{"data": data}, nil
}

func mustMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func pretty(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func notebookCanonicalURL(site, id string) string {
	if site == "" {
		site = "datadoghq.com"
	}
	if id == "" {
		return ""
	}
	return fmt.Sprintf("https://app.%s/notebook/%s", site, id)
}

func notebookIDFromEnvelope(env map[string]any) string {
	data := mustMap(env["data"])
	if data == nil {
		return ""
	}
	switch v := data["id"].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprint(v)
	}
}

func attachNotebookURL(out map[string]any, site, id string) {
	if id == "" {
		id = notebookIDFromEnvelope(out)
	}
	if url := notebookCanonicalURL(site, id); url != "" {
		out["url"] = url
	}
}

func notebookModifiedAt(env map[string]any) string {
	data := mustMap(env["data"])
	if data == nil {
		return ""
	}
	if meta := mustMap(data["meta"]); meta != nil {
		if v, ok := meta["modified_at"]; ok && v != nil {
			return fmt.Sprint(v)
		}
	}
	if attrs := mustMap(data["attributes"]); attrs != nil {
		if v, ok := attrs["modified_at"]; ok && v != nil {
			return fmt.Sprint(v)
		}
	}
	return ""
}

func notebookName(env map[string]any) string {
	data := mustMap(env["data"])
	attrs := mustMap(data["attributes"])
	name, _ := attrs["name"].(string)
	return name
}

func notebookCellCount(env map[string]any) int {
	data := mustMap(env["data"])
	attrs := mustMap(data["attributes"])
	cells, _ := attrs["cells"].([]any)
	return len(cells)
}

func stripNotebookIdentity(env map[string]any) {
	data := mustMap(env["data"])
	if data == nil {
		return
	}
	delete(data, "id")
	delete(data, "meta")
	delete(data, "relationships")
}

func SemanticDiffNotebookPayloads(current, next map[string]any) string {
	var parts []string
	curName := notebookName(current)
	nextName := notebookName(next)
	if curName != nextName {
		parts = append(parts, fmt.Sprintf("name: %q -> %q", curName, nextName))
	}
	curCells := notebookCellCount(current)
	nextCells := notebookCellCount(next)
	if curCells != nextCells {
		parts = append(parts, fmt.Sprintf("cells: %d -> %d", curCells, nextCells))
	}
	if len(parts) == 0 {
		return "no semantic changes"
	}
	return strings.Join(parts, "; ")
}
