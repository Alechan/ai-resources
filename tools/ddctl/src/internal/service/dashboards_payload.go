package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
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
	queryKindMetrics  = "metrics"
	queryKindLogs     = "logs"
	queryKindMonitors = "monitors"
)

var templateVarPattern = regexp.MustCompile(`\$[A-Za-z_][A-Za-z0-9_]*(?:\.value)?`)

// DashboardQuery is one extracted widget query (or a skipped widget).
type DashboardQuery struct {
	Query         string `json:"query,omitempty"`
	Kind          string `json:"kind,omitempty"`
	WidgetType    string `json:"widget_type,omitempty"`
	WidgetTitle   string `json:"widget_title,omitempty"`
	WidgetIndex   string `json:"widget_index,omitempty"`
	QueryIndex    int    `json:"query_index,omitempty"`
	SkippedReason string `json:"skipped_reason,omitempty"`
	DataSource    string `json:"data_source,omitempty"`
	MonitorID     int64  `json:"monitor_id,omitempty"`
}

// DashboardQueries is the result of walking a dashboard's widgets.
type DashboardQueries struct {
	Metrics  []DashboardQuery `json:"metrics"`
	Logs     []DashboardQuery `json:"logs"`
	Monitors []DashboardQuery `json:"monitors"`
	Skipped  []DashboardQuery `json:"skipped"`
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
// classifies metric, log, monitor, and skipped queries.
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

// DiffDashboardPayloads returns a JSON dump diff of two dashboard objects.
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

// SemanticDiffDashboardPayloads summarizes title, template variables, groups, and widgets.
func SemanticDiffDashboardPayloads(current, next map[string]any) string {
	var lines []string
	ct, _ := current["title"].(string)
	nt, _ := next["title"].(string)
	if ct != nt {
		lines = append(lines, fmt.Sprintf("~ title: %s -> %s", ct, nt))
	}
	cd, _ := current["description"].(string)
	nd, _ := next["description"].(string)
	if cd != nd {
		lines = append(lines, fmt.Sprintf("~ description: %s -> %s", cd, nd))
	}

	curVars := templateVarDefaults(current["template_variables"])
	nextVars := templateVarDefaults(next["template_variables"])
	names := map[string]bool{}
	for n := range curVars {
		names[n] = true
	}
	for n := range nextVars {
		names[n] = true
	}
	ordered := make([]string, 0, len(names))
	for n := range names {
		ordered = append(ordered, n)
	}
	sort.Strings(ordered)
	for _, name := range ordered {
		cv, cok := curVars[name]
		nv, nok := nextVars[name]
		switch {
		case cok && !nok:
			lines = append(lines, fmt.Sprintf("- template variable %s", name))
		case !cok && nok:
			lines = append(lines, fmt.Sprintf("+ template variable %s: %s", name, nv))
		case cv != nv:
			lines = append(lines, fmt.Sprintf("~ template variable %s: %s -> %s", name, cv, nv))
		}
	}

	curWidgets := collectWidgetSummaries(current["widgets"])
	nextWidgets := collectWidgetSummaries(next["widgets"])
	curCounts := countSummaries(curWidgets)
	nextCounts := countSummaries(nextWidgets)
	keys := map[string]bool{}
	for k := range curCounts {
		keys[k] = true
	}
	for k := range nextCounts {
		keys[k] = true
	}
	orderedKeys := make([]string, 0, len(keys))
	for k := range keys {
		orderedKeys = append(orderedKeys, k)
	}
	sort.Strings(orderedKeys)
	for _, key := range orderedKeys {
		c := curCounts[key]
		n := nextCounts[key]
		switch {
		case n > c:
			for i := 0; i < n-c; i++ {
				lines = append(lines, "+ "+formatSummaryKey(key, nextWidgets))
			}
		case c > n:
			for i := 0; i < c-n; i++ {
				lines = append(lines, "- "+formatSummaryKey(key, curWidgets))
			}
		}
	}

	if len(lines) == 0 {
		return "no changes\n"
	}
	return strings.Join(lines, "\n") + "\n"
}

func applyTemplateVariables(query string, vars map[string]string) string {
	if query == "" || len(vars) == 0 {
		return query
	}
	names := make([]string, 0, len(vars))
	for n := range vars {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })
	for _, name := range names {
		query = strings.ReplaceAll(query, "$"+name+".value", vars[name])
		query = strings.ReplaceAll(query, "$"+name, vars[name])
	}
	return query
}

func unresolvedTemplateVariables(query string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range templateVarPattern.FindAllString(query, -1) {
		if seen[m] {
			continue
		}
		seen[m] = true
		out = append(out, m)
	}
	sort.Strings(out)
	return out
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
	return extractFromWidgetsAt(widgets, "", out)
}

func extractFromWidgetsAt(widgets []any, prefix string, out *DashboardQueries) error {
	for i, raw := range widgets {
		idx := strconv.Itoa(i)
		if prefix != "" {
			idx = prefix + "." + idx
		}
		m := mustMap(raw)
		if m == nil {
			continue
		}
		def := mustMap(m["definition"])
		if def == nil {
			continue
		}
		meta := widgetMeta{
			Type:  stringField(def["type"]),
			Title: widgetTitle(def),
			Index: idx,
		}
		if err := extractFromDefinition(def, meta, out); err != nil {
			return err
		}
	}
	return nil
}

type widgetMeta struct {
	Type  string
	Title string
	Index string
}

func extractFromDefinition(def map[string]any, meta widgetMeta, out *DashboardQueries) error {
	if meta.Type == "group" {
		nested, _ := def["widgets"].([]any)
		return extractFromWidgetsAt(nested, meta.Index, out)
	}

	beforeMetrics, beforeLogs, beforeMonitors, beforeSkipped := len(out.Metrics), len(out.Logs), len(out.Monitors), len(out.Skipped)
	extractMonitorIDs(def, meta, out)

	if meta.Type == "log_stream" {
		if q, _ := def["query"].(string); strings.TrimSpace(q) != "" {
			out.Logs = append(out.Logs, meta.dashQuery(q, queryKindLogs, ""))
		}
	}

	requests, _ := def["requests"].([]any)
	for _, rawReq := range requests {
		req := mustMap(rawReq)
		if req == nil {
			continue
		}
		if q, _ := req["q"].(string); strings.TrimSpace(q) != "" {
			out.Metrics = append(out.Metrics, meta.dashQuery(q, queryKindMetrics, ""))
		}
		if err := extractQueryList(req["queries"], meta, out); err != nil {
			return err
		}
		if qobj := mustMap(req["query"]); qobj != nil {
			if err := extractQueryObject(qobj, meta, out); err != nil {
				return err
			}
		}
	}

	extracted := len(out.Metrics) > beforeMetrics || len(out.Logs) > beforeLogs || len(out.Monitors) > beforeMonitors || len(out.Skipped) > beforeSkipped
	if extracted || silentWidgetTypes[meta.Type] || meta.Type == "" {
		return nil
	}
	out.Skipped = append(out.Skipped, DashboardQuery{
		WidgetType:    meta.Type,
		WidgetTitle:   meta.Title,
		WidgetIndex:   meta.Index,
		SkippedReason: "no metrics/logs query in widget type " + meta.Type,
	})
	return nil
}

func extractMonitorIDs(def map[string]any, meta widgetMeta, out *DashboardQueries) {
	for _, key := range []string{"alert_id", "monitor_id"} {
		if id, ok := parseMonitorID(def[key]); ok {
			out.Monitors = append(out.Monitors, DashboardQuery{
				Kind:        queryKindMonitors,
				WidgetType:  meta.Type,
				WidgetTitle: meta.Title,
				WidgetIndex: meta.Index,
				MonitorID:   id,
				Query:       strconv.FormatInt(id, 10),
			})
		}
	}
}

func (m widgetMeta) dashQuery(query, kind, ds string) DashboardQuery {
	return DashboardQuery{
		Query:       query,
		Kind:        kind,
		WidgetType:  m.Type,
		WidgetTitle: m.Title,
		WidgetIndex: m.Index,
		DataSource:  ds,
	}
}

func extractQueryList(raw any, meta widgetMeta, out *DashboardQueries) error {
	queries, ok := raw.([]any)
	if !ok {
		return nil
	}
	for _, rawQ := range queries {
		q := mustMap(rawQ)
		if q == nil {
			continue
		}
		if err := extractQueryObject(q, meta, out); err != nil {
			return err
		}
	}
	return nil
}

func queryTextFromObject(q map[string]any) string {
	if s, _ := q["query"].(string); strings.TrimSpace(s) != "" {
		return s
	}
	if s, _ := q["query_string"].(string); strings.TrimSpace(s) != "" {
		return s
	}
	if search := mustMap(q["search"]); search != nil {
		if s, _ := search["query"].(string); strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func extractQueryObject(q map[string]any, meta widgetMeta, out *DashboardQueries) error {
	ds, _ := q["data_source"].(string)
	query := queryTextFromObject(q)
	kind := classifyDataSource(ds)
	switch kind {
	case queryKindMetrics:
		if strings.TrimSpace(query) == "" {
			return fail.NewValidation("invalid metric query entry", `each metrics query entry must include non-empty "query"`)
		}
		dq := meta.dashQuery(query, queryKindMetrics, ds)
		dq.QueryIndex = len(out.Metrics)
		out.Metrics = append(out.Metrics, dq)
	case queryKindLogs:
		if strings.TrimSpace(query) == "" {
			query = "*"
		}
		dq := meta.dashQuery(query, queryKindLogs, ds)
		dq.QueryIndex = len(out.Logs)
		out.Logs = append(out.Logs, dq)
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
			WidgetType:    meta.Type,
			WidgetTitle:   meta.Title,
			WidgetIndex:   meta.Index,
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

func parseMonitorID(v any) (int64, bool) {
	switch x := v.(type) {
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		return n, err == nil && n > 0
	case float64:
		n := int64(x)
		return n, float64(n) == x && n > 0
	case int:
		return int64(x), x > 0
	case int64:
		return x, x > 0
	case json.Number:
		n, err := x.Int64()
		return n, err == nil && n > 0
	default:
		return 0, false
	}
}

func widgetTitle(def map[string]any) string {
	if t := strings.TrimSpace(stringField(def["title"])); t != "" {
		return t
	}
	c := strings.TrimSpace(stringField(def["content"]))
	if c == "" {
		return ""
	}
	if i := strings.Index(c, "\n"); i >= 0 {
		c = c[:i]
	}
	if len(c) > 80 {
		c = c[:80]
	}
	return c
}

func stringField(v any) string {
	s, _ := v.(string)
	return s
}

type widgetSummary struct {
	Kind  string
	Type  string
	Title string
	Kids  int
}

func collectWidgetSummaries(raw any) []widgetSummary {
	widgets, _ := raw.([]any)
	var out []widgetSummary
	collectWidgetSummariesAt(widgets, &out)
	return out
}

func collectWidgetSummariesAt(widgets []any, out *[]widgetSummary) {
	for _, raw := range widgets {
		m := mustMap(raw)
		if m == nil {
			continue
		}
		def := mustMap(m["definition"])
		if def == nil {
			continue
		}
		t := stringField(def["type"])
		title := widgetTitle(def)
		if t == "group" {
			nested, _ := def["widgets"].([]any)
			*out = append(*out, widgetSummary{Kind: "group", Type: t, Title: title, Kids: countLeafWidgets(nested)})
			collectWidgetSummariesAt(nested, out)
			continue
		}
		*out = append(*out, widgetSummary{Kind: "widget", Type: t, Title: title})
	}
}

func countLeafWidgets(widgets []any) int {
	n := 0
	for _, raw := range widgets {
		m := mustMap(raw)
		if m == nil {
			continue
		}
		def := mustMap(m["definition"])
		if def == nil {
			continue
		}
		if stringField(def["type"]) == "group" {
			nested, _ := def["widgets"].([]any)
			n += countLeafWidgets(nested)
			continue
		}
		n++
	}
	return n
}

func summaryKey(s widgetSummary) string {
	return s.Kind + "\t" + s.Type + "\t" + s.Title
}

func countSummaries(items []widgetSummary) map[string]int {
	out := map[string]int{}
	for _, s := range items {
		out[summaryKey(s)]++
	}
	return out
}

func formatSummaryKey(key string, items []widgetSummary) string {
	for _, s := range items {
		if summaryKey(s) == key {
			return formatSummary(s)
		}
	}
	return key
}

func formatSummary(s widgetSummary) string {
	title := s.Title
	if title == "" {
		title = s.Type
	}
	switch s.Kind {
	case "group":
		line := "group: " + title
		if s.Kids > 0 {
			line += fmt.Sprintf("\n+ %d widgets", s.Kids)
		}
		return line
	default:
		if s.Title != "" {
			return fmt.Sprintf("widget %s: %s", s.Type, s.Title)
		}
		return "widget " + s.Type + ":"
	}
}

func templateVarDefaults(raw any) map[string]string {
	arr, _ := raw.([]any)
	out := map[string]string{}
	for _, item := range arr {
		m := mustMap(item)
		if m == nil {
			continue
		}
		name := stringField(m["name"])
		if name == "" {
			continue
		}
		out[name] = stringField(m["default"])
	}
	return out
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
