package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

var dashboardIdentityFields = []string{
	"id",
	"url",
	"author_handle",
	"author_name",
	"created_at",
	"modified_at",
}

var silentWidgetTypes = map[string]bool{
	"note":      true,
	"image":     true,
	"iframe":    true,
	"free_text": true,
}

const (
	queryKindMetrics = "metrics"
	queryKindLogs    = "logs"
)

// DashboardQuery is one extracted widget query (or a skipped widget).
type DashboardQuery struct {
	Query         string `json:"query,omitempty"`
	Kind          string `json:"kind,omitempty"`
	WidgetType    string `json:"widget_type,omitempty"`
	SkippedReason string `json:"skipped_reason,omitempty"`
	DataSource    string `json:"data_source,omitempty"`
}

// DashboardQueries is the result of walking a dashboard's widgets.
type DashboardQueries struct {
	Metrics []DashboardQuery `json:"metrics"`
	Logs    []DashboardQuery `json:"logs"`
	Skipped []DashboardQuery `json:"skipped"`
}

// NormalizeDashboardPayload accepts a raw dashboard object or
// {"dashboard": {...}} and returns the inner dashboard map.
func NormalizeDashboardPayload(raw []byte) (map[string]any, error) {
	var env map[string]any
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fail.NewValidation("invalid JSON dashboard file", "ensure the file contains a dashboard object")
	}
	if data, ok := env["data"].(map[string]any); ok {
		if t, _ := data["type"].(string); t == "notebooks" {
			return nil, fail.NewValidation("notebook envelope is not a dashboard", `expected a dashboard object with "title", "widgets", and "layout_type"`)
		}
	}
	if inner, ok := env["dashboard"].(map[string]any); ok {
		return cloneMap(inner), nil
	}
	return env, nil
}

// PrepareDashboardCreatePayload validates required fields, applies an optional
// title override, and strips identity fields from a previous GET.
func PrepareDashboardCreatePayload(env map[string]any, titleOverride string) (map[string]any, error) {
	payload := cloneMap(env)
	if titleOverride != "" {
		payload["title"] = titleOverride
	}
	if err := assertDashboardRequired(payload); err != nil {
		return nil, err
	}
	for _, k := range dashboardIdentityFields {
		delete(payload, k)
	}
	return payload, nil
}

// PrepareDashboardUpdatePayload validates required fields and sets dashboard ID.
// Update is full replacement and requires replaceAll.
func PrepareDashboardUpdatePayload(env map[string]any, dashboardID string, replaceAll bool) (map[string]any, error) {
	if !replaceAll {
		return nil, fail.NewValidation("--replace-all is required", "update is full replacement; pass --replace-all to confirm")
	}
	if strings.TrimSpace(dashboardID) == "" {
		return nil, fail.NewValidation("missing dashboard ID", "usage: ddctl dashboards update <id> --from-file <path> --replace-all")
	}
	payload := cloneMap(env)
	if err := assertDashboardRequired(payload); err != nil {
		return nil, err
	}
	payload["id"] = dashboardID
	return payload, nil
}

// ExtractDashboardQueries walks widgets (including group children) and
// classifies metric, log, and skipped queries.
func ExtractDashboardQueries(env map[string]any) (DashboardQueries, error) {
	widgets, ok := env["widgets"].([]any)
	if !ok {
		return DashboardQueries{}, fail.NewValidation(`missing "widgets"`, "set widgets to a non-empty array")
	}
	var out DashboardQueries
	if err := extractFromWidgets(widgets, &out); err != nil {
		return DashboardQueries{}, err
	}
	return out, nil
}

// DiffDashboardPayloads returns a text diff of two dashboard JSON objects.
func DiffDashboardPayloads(current, next map[string]any) string {
	left := marshalStable(current)
	right := marshalStable(next)
	if bytes.Equal(left, right) {
		return "no changes\n"
	}
	var b strings.Builder
	b.WriteString("--- current\n")
	b.Write(left)
	if len(left) == 0 || left[len(left)-1] != '\n' {
		b.WriteByte('\n')
	}
	b.WriteString("+++ next\n")
	b.Write(right)
	if len(right) == 0 || right[len(right)-1] != '\n' {
		b.WriteByte('\n')
	}
	return b.String()
}

func assertDashboardRequired(payload map[string]any) error {
	title, _ := payload["title"].(string)
	if strings.TrimSpace(title) == "" {
		return fail.NewValidation(`missing "title"`, "set title in the file or pass --title")
	}
	layout, _ := payload["layout_type"].(string)
	if layout != "ordered" && layout != "free" {
		return fail.NewValidation(`invalid "layout_type"`, `set layout_type to "ordered" or "free"`)
	}
	widgets, ok := payload["widgets"].([]any)
	if !ok {
		return fail.NewValidation(`missing "widgets"`, "set widgets to a non-empty array")
	}
	if len(widgets) == 0 {
		return fail.NewValidation(`missing "widgets"`, "set widgets to a non-empty array")
	}
	return assertWidgets(widgets)
}

func assertWidgets(widgets []any) error {
	if len(widgets) == 0 {
		return fail.NewValidation(`missing "widgets"`, "set widgets to a non-empty array")
	}
	for _, raw := range widgets {
		m := mustMap(raw)
		if m == nil {
			return fail.NewValidation("invalid widget entry", "expected widget to be an object")
		}
		def := mustMap(m["definition"])
		if def == nil {
			return fail.NewValidation("invalid widget definition", "expected definition object in each widget")
		}
		t, _ := def["type"].(string)
		if t == "" {
			return fail.NewValidation("invalid widget definition.type", "each widget definition must include type")
		}
		if t == "group" {
			nested, ok := def["widgets"].([]any)
			if !ok {
				return fail.NewValidation("invalid group widgets", "group definition.widgets must be a non-empty array")
			}
			if err := assertWidgets(nested); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractFromWidgets(widgets []any, out *DashboardQueries) error {
	for _, raw := range widgets {
		m := mustMap(raw)
		if m == nil {
			continue
		}
		def := mustMap(m["definition"])
		if def == nil {
			continue
		}
		widgetType, _ := def["type"].(string)
		if err := extractFromDefinition(def, widgetType, out); err != nil {
			return err
		}
	}
	return nil
}

func extractFromDefinition(def map[string]any, widgetType string, out *DashboardQueries) error {
	if widgetType == "group" {
		nested, _ := def["widgets"].([]any)
		return extractFromWidgets(nested, out)
	}

	beforeMetrics, beforeLogs, beforeSkipped := len(out.Metrics), len(out.Logs), len(out.Skipped)

	if widgetType == "log_stream" {
		if q, _ := def["query"].(string); strings.TrimSpace(q) != "" {
			out.Logs = append(out.Logs, DashboardQuery{Query: q, Kind: queryKindLogs, WidgetType: widgetType})
		}
	}

	requests, _ := def["requests"].([]any)
	for _, rawReq := range requests {
		req := mustMap(rawReq)
		if req == nil {
			continue
		}
		if q, _ := req["q"].(string); strings.TrimSpace(q) != "" {
			out.Metrics = append(out.Metrics, DashboardQuery{Query: q, Kind: queryKindMetrics, WidgetType: widgetType})
		}
		if err := extractQueryList(req["queries"], widgetType, out); err != nil {
			return err
		}
		if qobj := mustMap(req["query"]); qobj != nil {
			if err := extractQueryObject(qobj, widgetType, out); err != nil {
				return err
			}
		}
	}

	extracted := len(out.Metrics) > beforeMetrics || len(out.Logs) > beforeLogs || len(out.Skipped) > beforeSkipped
	if extracted || silentWidgetTypes[widgetType] || widgetType == "" {
		return nil
	}
	out.Skipped = append(out.Skipped, DashboardQuery{
		WidgetType:    widgetType,
		SkippedReason: "no metrics/logs query in widget type " + widgetType,
	})
	return nil
}

func extractQueryList(raw any, widgetType string, out *DashboardQueries) error {
	queries, ok := raw.([]any)
	if !ok {
		return nil
	}
	for _, rawQ := range queries {
		q := mustMap(rawQ)
		if q == nil {
			continue
		}
		if err := extractQueryObject(q, widgetType, out); err != nil {
			return err
		}
	}
	return nil
}

func extractQueryObject(q map[string]any, widgetType string, out *DashboardQueries) error {
	ds, _ := q["data_source"].(string)
	query, _ := q["query"].(string)
	if strings.TrimSpace(query) == "" {
		query, _ = q["query_string"].(string)
	}
	kind := classifyDataSource(ds)
	switch kind {
	case queryKindMetrics:
		if strings.TrimSpace(query) == "" {
			return fail.NewValidation("invalid metric query entry", `each metrics query entry must include non-empty "query"`)
		}
		out.Metrics = append(out.Metrics, DashboardQuery{Query: query, Kind: queryKindMetrics, WidgetType: widgetType, DataSource: ds})
	case queryKindLogs:
		if strings.TrimSpace(query) == "" {
			return fail.NewValidation("invalid log query entry", `each logs query entry must include non-empty "query" or "query_string"`)
		}
		out.Logs = append(out.Logs, DashboardQuery{Query: query, Kind: queryKindLogs, WidgetType: widgetType, DataSource: ds})
	default:
		if ds == "" && strings.TrimSpace(query) == "" {
			return nil
		}
		reason := "unsupported data_source"
		if ds != "" {
			reason = "unsupported data_source " + ds
		}
		out.Skipped = append(out.Skipped, DashboardQuery{
			Query:         query,
			WidgetType:    widgetType,
			DataSource:    ds,
			SkippedReason: reason,
		})
	}
	return nil
}

func classifyDataSource(ds string) string {
	switch ds {
	case "", "metrics", "metrics_query":
		return queryKindMetrics
	case "logs", "logs_stream", "logs_analytics":
		return queryKindLogs
	default:
		return ""
	}
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	b, err := json.Marshal(in)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func marshalStable(v any) []byte {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return []byte(fmt.Sprintf("%v", v))
	}
	return append(b, '\n')
}
