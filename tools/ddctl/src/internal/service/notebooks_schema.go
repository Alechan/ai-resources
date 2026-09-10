package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

var notebookTemplateVarRe = regexp.MustCompile(`\$([a-zA-Z0-9_]+)(?:\.value)?`)

type NotebookMetricQuery struct {
	CellIndex    int    `json:"cell_index"`
	RequestIndex int    `json:"request_index"`
	Original     string `json:"original"`
	Resolved     string `json:"resolved"`
}

var notebookTextDefinitionTypes = map[string]struct{}{
	"markdown":  {},
	"rich_text": {},
	"note":      {},
}

var notebookChartDefinitionTypes = map[string]struct{}{
	"timeseries":   {},
	"heatmap":      {},
	"distribution": {},
	"toplist":      {},
	"query_value":  {},
	"change":       {},
	"scatterplot":  {},
	"geomap":       {},
	"servicemap":   {},
	"trace":        {},
	"log_stream":   {},
}

// PrepareNotebookSchema normalizes template variables and validates notebook cells.
// Validation and create/update share this path.
func PrepareNotebookSchema(env map[string]any) ([]string, error) {
	data := mustMap(env["data"])
	if data == nil {
		return nil, fail.NewValidation(`missing "data"`, `expected envelope with "data" object`)
	}
	attrs := mustMap(data["attributes"])
	if attrs == nil {
		return nil, fail.NewValidation(`missing "data.attributes"`, `expected envelope with "data.attributes" object`)
	}
	warnings, err := normalizeNotebookTemplateVariables(attrs)
	if err != nil {
		return warnings, err
	}
	if err := validateNotebookAttributes(attrs); err != nil {
		return warnings, err
	}
	return warnings, nil
}

func normalizeNotebookTemplateVariables(attrs map[string]any) ([]string, error) {
	raw, ok := attrs["template_variables"].([]any)
	if !ok || len(raw) == 0 {
		return nil, nil
	}
	var warnings []string
	for i, item := range raw {
		tv := mustMap(item)
		if tv == nil {
			return warnings, fail.NewValidation(
				fmt.Sprintf("invalid template_variables[%d]", i),
				"each template variable must be an object",
			)
		}
		_, hasDefault := tv["default"]
		_, hasDefaults := tv["defaults"]
		if hasDefault && hasDefaults {
			return warnings, fail.NewValidation(
				fmt.Sprintf("conflicting template_variables[%d] default and defaults", i),
				"use defaults only; remove deprecated default",
			)
		}
		if hasDefault && !hasDefaults {
			tv["defaults"] = []any{tv["default"]}
			delete(tv, "default")
			warnings = append(warnings, fmt.Sprintf(
				"template_variables[%d]: deprecated field default converted to defaults",
				i,
			))
		}
	}
	return warnings, nil
}

func validateNotebookAttributes(attrs map[string]any) error {
	if name, _ := attrs["name"].(string); strings.TrimSpace(name) == "" {
		return fail.NewValidation(`missing "attributes.name"`, "set attributes.name in the notebook payload")
	}
	if _, ok := attrs["time"].(map[string]any); !ok {
		return fail.NewValidation(`missing "attributes.time"`, `set attributes.time (e.g. {"live_span":"4h"})`)
	}
	cells, ok := attrs["cells"].([]any)
	if !ok || len(cells) == 0 {
		return fail.NewValidation(`missing "attributes.cells"`, "set attributes.cells to a non-empty array")
	}
	for i, rawCell := range cells {
		if err := validateNotebookCell(rawCell, i); err != nil {
			return err
		}
	}
	return nil
}

func validateNotebookCell(rawCell any, cellIndex int) error {
	cell := mustMap(rawCell)
	if cell == nil {
		return fail.NewValidation(
			fmt.Sprintf("invalid cell[%d]", cellIndex),
			"each cell must be an object",
		)
	}
	if t, _ := cell["type"].(string); t != "notebook_cells" {
		return fail.NewValidation(
			fmt.Sprintf("invalid cell[%d].type", cellIndex),
			`expected "notebook_cells"`,
		)
	}
	cellAttrs := mustMap(cell["attributes"])
	if cellAttrs == nil {
		return fail.NewValidation(
			fmt.Sprintf("missing cell[%d].attributes", cellIndex),
			"each cell must include attributes",
		)
	}
	def := mustMap(cellAttrs["definition"])
	if def == nil {
		return fail.NewValidation(
			fmt.Sprintf("missing cell[%d].attributes.definition", cellIndex),
			"each cell must include attributes.definition",
		)
	}
	defType, _ := def["type"].(string)
	if strings.TrimSpace(defType) == "" {
		return fail.NewValidation(
			fmt.Sprintf("missing cell[%d].attributes.definition.type", cellIndex),
			"each cell definition must include type",
		)
	}
	if _, ok := notebookTextDefinitionTypes[defType]; ok {
		text, _ := def["text"].(string)
		if strings.TrimSpace(text) == "" {
			return fail.NewValidation(
				fmt.Sprintf("missing cell[%d] text", cellIndex),
				fmt.Sprintf("%s cells require non-empty definition.text", defType),
			)
		}
		return nil
	}
	if _, ok := notebookChartDefinitionTypes[defType]; ok {
		return validateNotebookChartCell(cellAttrs, def, cellIndex, defType)
	}
	return nil
}

func validateNotebookChartCell(cellAttrs, def map[string]any, cellIndex int, defType string) error {
	if defType == "timeseries" {
		if graphSize, _ := cellAttrs["graph_size"].(string); strings.TrimSpace(graphSize) == "" {
			return fail.NewValidation(
				fmt.Sprintf("missing cell[%d].attributes.graph_size", cellIndex),
				"timeseries cells require attributes.graph_size",
			)
		}
		if mustMap(cellAttrs["split_by"]) == nil {
			return fail.NewValidation(
				fmt.Sprintf("missing cell[%d].attributes.split_by", cellIndex),
				"timeseries cells require attributes.split_by",
			)
		}
		if _, ok := cellAttrs["time"]; !ok {
			return fail.NewValidation(
				fmt.Sprintf("missing cell[%d].attributes.time", cellIndex),
				"timeseries cells require attributes.time (null is allowed)",
			)
		}
	}
	requests, ok := def["requests"].([]any)
	if !ok || len(requests) == 0 {
		return fail.NewValidation(
			fmt.Sprintf("missing cell[%d].definition.requests", cellIndex),
			fmt.Sprintf("%s cells require a non-empty definition.requests array", defType),
		)
	}
	for reqIndex, rawReq := range requests {
		if err := validateNotebookRequest(rawReq, cellIndex, reqIndex); err != nil {
			return err
		}
	}
	return nil
}

func validateNotebookRequest(rawReq any, cellIndex, reqIndex int) error {
	req := mustMap(rawReq)
	if req == nil {
		return fail.NewValidation(
			fmt.Sprintf("invalid cell[%d].requests[%d]", cellIndex, reqIndex),
			"each request must be an object",
		)
	}
	q, _ := req["q"].(string)
	if strings.TrimSpace(q) != "" {
		return nil
	}
	qArr, hasQueries := req["queries"].([]any)
	if !hasQueries {
		return fail.NewValidation(
			fmt.Sprintf("missing cell[%d].requests[%d] query", cellIndex, reqIndex),
			`timeseries requests require non-empty "q" or a "queries" array`,
		)
	}
	if len(qArr) == 0 {
		return fail.NewValidation(
			fmt.Sprintf("missing cell[%d].requests[%d] query", cellIndex, reqIndex),
			`requests must include non-empty "q" or at least one queries[] entry with query`,
		)
	}
	for qi, rawQ := range qArr {
		entry := mustMap(rawQ)
		query, _ := entry["query"].(string)
		if strings.TrimSpace(query) == "" {
			return fail.NewValidation(
				fmt.Sprintf("missing cell[%d].requests[%d].queries[%d].query", cellIndex, reqIndex, qi),
				`each queries[] entry must include non-empty "query"`,
			)
		}
	}
	return nil
}

func ExtractNotebookMetricQueries(env map[string]any) ([]NotebookMetricQuery, error) {
	data := mustMap(env["data"])
	attrs := mustMap(data["attributes"])
	cells, _ := attrs["cells"].([]any)
	templateVars := notebookTemplateVariableMaps(attrs)

	var out []NotebookMetricQuery
	for cellIndex, rawCell := range cells {
		cell := mustMap(rawCell)
		cellAttrs := mustMap(cell["attributes"])
		def := mustMap(cellAttrs["definition"])
		if def == nil {
			continue
		}
		if _, ok := notebookChartDefinitionTypes[def["type"].(string)]; !ok {
			continue
		}
		requests, _ := def["requests"].([]any)
		for reqIndex, rawReq := range requests {
			req := mustMap(rawReq)
			if req == nil {
				continue
			}
			if q, _ := req["q"].(string); strings.TrimSpace(q) != "" {
				resolved, err := resolveNotebookQuery(q, templateVars)
				if err != nil {
					return nil, err
				}
				out = append(out, NotebookMetricQuery{
					CellIndex:    cellIndex,
					RequestIndex: reqIndex,
					Original:     q,
					Resolved:     resolved,
				})
				continue
			}
			qArr, _ := req["queries"].([]any)
			for _, rawQ := range qArr {
				entry := mustMap(rawQ)
				query, _ := entry["query"].(string)
				if strings.TrimSpace(query) == "" {
					continue
				}
				resolved, err := resolveNotebookQuery(query, templateVars)
				if err != nil {
					return nil, err
				}
				out = append(out, NotebookMetricQuery{
					CellIndex:    cellIndex,
					RequestIndex: reqIndex,
					Original:     query,
					Resolved:     resolved,
				})
			}
		}
	}
	return out, nil
}

func notebookTemplateVariableMaps(attrs map[string]any) []map[string]any {
	raw, _ := attrs["template_variables"].([]any)
	var out []map[string]any
	for _, item := range raw {
		if tv := mustMap(item); tv != nil {
			out = append(out, tv)
		}
	}
	return out
}

func resolveNotebookQuery(query string, templateVars []map[string]any) (string, error) {
	byName := map[string]map[string]any{}
	for _, tv := range templateVars {
		name, _ := tv["name"].(string)
		if name != "" {
			byName[name] = tv
		}
	}
	var unresolved []string
	resolved := notebookTemplateVarRe.ReplaceAllStringFunc(query, func(match string) string {
		parts := notebookTemplateVarRe.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		name := parts[1]
		tv, ok := byName[name]
		if !ok {
			unresolved = append(unresolved, match)
			return match
		}
		value, err := templateVariableDefaultValue(tv)
		if err != nil {
			unresolved = append(unresolved, match)
			return match
		}
		return value
	})
	if len(unresolved) > 0 {
		return "", fail.NewValidation(
			"unresolved notebook template variable in query",
			fmt.Sprintf("resolve template variables before preflight: %s", strings.Join(unresolved, ", ")),
		)
	}
	return resolved, nil
}

func templateVariableDefaultValue(tv map[string]any) (string, error) {
	if defaults, ok := tv["defaults"].([]any); ok && len(defaults) > 0 {
		return fmt.Sprint(defaults[0]), nil
	}
	if def, ok := tv["default"]; ok {
		return fmt.Sprint(def), nil
	}
	name, _ := tv["name"].(string)
	return "", fmt.Errorf("template variable %q has no defaults", name)
}

func NotebookMutationSummary(env map[string]any, site string) map[string]any {
	data := mustMap(env["data"])
	attrs := mustMap(data["attributes"])
	id := notebookIDFromEnvelope(env)
	cells, _ := attrs["cells"].([]any)
	cellIDs := make([]string, 0, len(cells))
	for _, rawCell := range cells {
		cell := mustMap(rawCell)
		if cellID, _ := cell["id"].(string); cellID != "" {
			cellIDs = append(cellIDs, cellID)
		}
	}
	status, _ := attrs["status"].(string)
	summary := map[string]any{
		"id":         id,
		"name":       notebookName(env),
		"url":        notebookCanonicalURL(site, id),
		"status":     status,
		"cell_count": len(cells),
		"cell_ids":   cellIDs,
	}
	if url, _ := env["url"].(string); url != "" {
		summary["url"] = url
	}
	return summary
}

func cloneNotebookEnvelope(env map[string]any) map[string]any {
	b, _ := json.Marshal(env)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}
