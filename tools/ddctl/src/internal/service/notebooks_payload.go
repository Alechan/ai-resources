package service

import (
	"encoding/json"
	"fmt"
	"strconv"

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
	for _, cell := range cells {
		if err := assertNotebookCell(cell); err != nil {
			return nil, err
		}
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
	for _, cell := range cells {
		if err := assertNotebookCell(cell); err != nil {
			return nil, err
		}
	}

	data["type"] = "notebooks"
	data["id"] = id
	return map[string]any{"data": data}, nil
}

// ExtractTimeseriesQueries returns all metric query strings from notebook timeseries cells.
func ExtractTimeseriesQueries(env map[string]any) ([]string, error) {
	data := mustMap(env["data"])
	if data == nil {
		return nil, fail.NewValidation(`missing "data"`, `expected envelope with "data" object`)
	}
	attrs := mustMap(data["attributes"])
	if attrs == nil {
		return nil, fail.NewValidation(`missing "data.attributes"`, `expected envelope with "data.attributes" object`)
	}
	cells, ok := attrs["cells"].([]any)
	if !ok {
		return nil, fail.NewValidation(`missing "attributes.cells"`, "set attributes.cells to an array")
	}

	var queries []string
	for _, rawCell := range cells {
		cell := mustMap(rawCell)
		if cell == nil {
			continue
		}
		cellAttrs := mustMap(cell["attributes"])
		if cellAttrs == nil {
			continue
		}
		def := mustMap(cellAttrs["definition"])
		if def == nil {
			continue
		}
		if defType, _ := def["type"].(string); defType != "timeseries" {
			continue
		}
		requests, ok := def["requests"].([]any)
		if !ok {
			return nil, fail.NewValidation("invalid timeseries definition", `timeseries "requests" must be an array`)
		}
		for _, rawReq := range requests {
			req := mustMap(rawReq)
			if req == nil {
				continue
			}
			qArr, ok := req["queries"].([]any)
			if !ok {
				return nil, fail.NewValidation("invalid timeseries queries", `timeseries request "queries" must be an array`)
			}
			for _, rawQ := range qArr {
				q := mustMap(rawQ)
				if q == nil {
					continue
				}
				query, _ := q["query"].(string)
				if query == "" {
					return nil, fail.NewValidation("invalid metric query entry", `each timeseries query entry must include non-empty "query"`)
				}
				queries = append(queries, query)
			}
		}
	}
	return queries, nil
}

func mustMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func pretty(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func assertNotebookCell(cell any) error {
	m := mustMap(cell)
	if m == nil {
		return fail.NewValidation("invalid cell entry", "expected cell to be an object")
	}
	if t, _ := m["type"].(string); t != "notebook_cells" {
		return fail.NewValidation("invalid cell.type", fmt.Sprintf(`expected "notebook_cells", got %s`, pretty(m["type"])))
	}
	attrs := mustMap(m["attributes"])
	if attrs == nil {
		return fail.NewValidation("invalid cell.attributes", "expected attributes object in each cell")
	}
	if mustMap(attrs["definition"]) == nil {
		return fail.NewValidation("invalid cell definition", "expected attributes.definition object in each cell")
	}
	return nil
}
