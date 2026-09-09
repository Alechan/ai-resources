package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

type DashboardGetInput struct {
	ID string
}

type DashboardGetResult map[string]any

type DashboardMutationInput struct {
	FilePath           string
	Title              string
	ID                 string
	SkipValidate       bool
	AllowEmptySeries   bool
	From               string
	To                 string
	DryRun             bool
	ShowDiff           bool
	IfUnmodifiedSince string
	TemplateVariables  map[string]string
}

type DashboardMutationResult map[string]any

type DashboardValidateInput struct {
	FilePath          string
	From              string
	To                string
	AllowEmptySeries  bool
	TemplateVariables map[string]string
}

type DashboardValidateResult struct {
	Structure        string           `json:"structure"`
	QueryCount       int              `json:"query_count"`
	MetricsValid     int              `json:"metrics_valid"`
	MetricsNoData    int              `json:"metrics_no_data"`
	LogsValid        int              `json:"logs_valid"`
	LogsNoData       int              `json:"logs_no_data"`
	MonitorsValid    int              `json:"monitors_valid"`
	Metrics          []string         `json:"metrics"`
	Logs             []string         `json:"logs"`
	Monitors         []string         `json:"monitors,omitempty"`
	Skipped          []DashboardQuery `json:"skipped"`
	Warnings         []string         `json:"warnings"`
	AllowEmptySeries bool             `json:"allow_empty_series"`
}

type DashboardsService struct {
	dd      *datadogapi.Client
	metrics *MetricsQueryService
	logs    *LogsQueryService
	site    string
}

func NewDashboardsService(dd *datadogapi.Client, metrics *MetricsQueryService, logs *LogsQueryService, site string) *DashboardsService {
	return &DashboardsService{dd: dd, metrics: metrics, logs: logs, site: site}
}

func (s *DashboardsService) Get(ctx context.Context, input DashboardGetInput) (DashboardGetResult, error) {
	id := strings.TrimSpace(input.ID)
	if id == "" {
		return nil, fail.NewValidation("missing dashboard ID", "usage: ddctl dashboards get <id>")
	}
	path := fmt.Sprintf("/api/v1/dashboard/%s", id)
	var out map[string]any
	if err := s.dd.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	attachDashboardURL(out, s.site, id)
	return out, nil
}

func (s *DashboardsService) Create(ctx context.Context, input DashboardMutationInput) (DashboardMutationResult, error) {
	env, err := loadDashboardEnvelopeFromFile(input.FilePath)
	if err != nil {
		return nil, err
	}
	payload, err := PrepareDashboardCreatePayload(env, input.Title)
	if err != nil {
		return nil, err
	}
	if !input.SkipValidate {
		if _, err := s.validatePrepared(ctx, payload, input.From, input.To, input.AllowEmptySeries, input.TemplateVariables); err != nil {
			return nil, err
		}
	}
	if input.DryRun {
		widgets, _ := payload["widgets"].([]any)
		title, _ := payload["title"].(string)
		return DashboardMutationResult{
			"dry_run":      true,
			"title":        title,
			"layout_type":  payload["layout_type"],
			"widget_count": len(widgets),
		}, nil
	}
	var out map[string]any
	if err := s.dd.Post(ctx, "/api/v1/dashboard", payload, &out); err != nil {
		return nil, err
	}
	id := dashboardIDFrom(out)
	attachDashboardURL(out, s.site, id)
	return out, nil
}

func (s *DashboardsService) Update(ctx context.Context, input DashboardMutationInput, replaceAll bool) (DashboardMutationResult, error) {
	env, err := loadDashboardEnvelopeFromFile(input.FilePath)
	if err != nil {
		return nil, err
	}
	payload, err := PrepareDashboardUpdatePayload(env, input.ID, replaceAll)
	if err != nil {
		return nil, err
	}
	if !input.SkipValidate {
		if _, err := s.validatePrepared(ctx, payload, input.From, input.To, input.AllowEmptySeries, input.TemplateVariables); err != nil {
			return nil, err
		}
	}

	needGet := strings.TrimSpace(input.IfUnmodifiedSince) != "" || input.DryRun || input.ShowDiff
	var semantic string
	if needGet {
		current, err := s.Get(ctx, DashboardGetInput{ID: input.ID})
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(input.IfUnmodifiedSince) != "" {
			got := fmt.Sprint(current["modified_at"])
			if !modifiedAtMatches(got, input.IfUnmodifiedSince) {
				return nil, fail.NewValidation(
					"dashboard modified_at does not match --if-unmodified-since",
					fmt.Sprintf("remote modified_at is %s", got),
				)
			}
		}
		left := cloneMap(map[string]any(current))
		right := cloneMap(payload)
		for _, k := range dashboardIdentityFields {
			delete(left, k)
			delete(right, k)
		}
		semantic = SemanticDiffDashboardPayloads(left, right)
		if input.DryRun {
			return DashboardMutationResult{
				"dry_run":   true,
				"id":        input.ID,
				"url":       dashboardCanonicalURL(s.site, input.ID),
				"diff":      semantic,
				"json_diff": DiffDashboardPayloads(left, right),
			}, nil
		}
	}

	out, err := s.putDashboard(ctx, input.ID, payload)
	if err != nil {
		return nil, err
	}
	if semantic != "" {
		out["diff"] = semantic
	}
	return out, nil
}

func (s *DashboardsService) putDashboard(ctx context.Context, id string, payload map[string]any) (DashboardMutationResult, error) {
	path := fmt.Sprintf("/api/v1/dashboard/%s", id)
	var out map[string]any
	if err := s.dd.Put(ctx, path, payload, &out); err != nil {
		return nil, err
	}
	attachDashboardURL(out, s.site, id)
	return out, nil
}

func (s *DashboardsService) Validate(ctx context.Context, input DashboardValidateInput) (DashboardValidateResult, error) {
	env, err := loadDashboardEnvelopeFromFile(input.FilePath)
	if err != nil {
		return DashboardValidateResult{}, err
	}
	if err := assertDashboardRequired(env); err != nil {
		return DashboardValidateResult{}, err
	}
	return s.validatePrepared(ctx, env, input.From, input.To, input.AllowEmptySeries, input.TemplateVariables)
}

func (s *DashboardsService) validatePrepared(ctx context.Context, env map[string]any, from, to string, allowEmpty bool, vars map[string]string) (DashboardValidateResult, error) {
	if from == "" {
		from = "now-30d"
	}
	if to == "" {
		to = "now"
	}
	extracted, err := ExtractDashboardQueries(env)
	if err != nil {
		return DashboardValidateResult{}, err
	}
	result := DashboardValidateResult{
		Structure:        "valid",
		Skipped:          extracted.Skipped,
		AllowEmptySeries: allowEmpty,
	}
	for _, q := range extracted.Metrics {
		result.Metrics = append(result.Metrics, q.Query)
	}
	for _, q := range extracted.Logs {
		result.Logs = append(result.Logs, q.Query)
	}
	for _, q := range extracted.Monitors {
		result.Monitors = append(result.Monitors, q.Query)
	}
	result.QueryCount = len(result.Metrics) + len(result.Logs)
	for _, q := range extracted.Skipped {
		result.Warnings = append(result.Warnings, skippedWarning(q))
	}

	for _, q := range extracted.Metrics {
		query := applyTemplateVariables(q.Query, vars)
		for _, unresolved := range unresolvedTemplateVariables(query) {
			result.Warnings = append(result.Warnings, queryWarning(q, "unresolved template variable "+unresolved))
		}
		metricsResult, err := s.metrics.Run(ctx, MetricsQueryInput{Query: query, From: from, To: to})
		if err != nil {
			return DashboardValidateResult{}, wrapQueryError(q, err)
		}
		if len(metricsResult.Series) == 0 {
			result.MetricsNoData++
			result.Warnings = append(result.Warnings, queryWarning(q, "no data in selected window"))
			continue
		}
		result.MetricsValid++
	}
	for _, q := range extracted.Logs {
		query := applyTemplateVariables(q.Query, vars)
		for _, unresolved := range unresolvedTemplateVariables(query) {
			result.Warnings = append(result.Warnings, queryWarning(q, "unresolved template variable "+unresolved))
		}
		logsResult, err := s.logs.Run(ctx, LogsQueryInput{
			Query:     query,
			From:      from,
			To:        to,
			Limit:     1,
			CountOnly: true,
		})
		if err != nil {
			return DashboardValidateResult{}, wrapQueryError(q, err)
		}
		if logsResult.HitCount == 0 {
			result.LogsNoData++
			result.Warnings = append(result.Warnings, queryWarning(q, "no data in selected window"))
			continue
		}
		result.LogsValid++
	}
	for _, q := range extracted.Monitors {
		path := fmt.Sprintf("/api/v1/monitor/%d", q.MonitorID)
		var mon map[string]any
		if err := s.dd.Get(ctx, path, &mon); err != nil {
			return DashboardValidateResult{}, wrapQueryError(q, err)
		}
		result.MonitorsValid++
	}
	_ = allowEmpty
	return result, nil
}

func skippedWarning(q DashboardQuery) string {
	msg := q.SkippedReason
	if q.WidgetType != "" {
		msg = q.WidgetType + ": " + q.SkippedReason
	}
	if q.WidgetTitle != "" {
		return fmt.Sprintf("widget %q: %s", q.WidgetTitle, msg)
	}
	return msg
}

func queryWarning(q DashboardQuery, msg string) string {
	if q.WidgetTitle != "" {
		return fmt.Sprintf("widget %q: %s", q.WidgetTitle, msg)
	}
	if q.WidgetIndex != "" {
		return fmt.Sprintf("widget %s: %s", q.WidgetIndex, msg)
	}
	return msg
}

func wrapQueryError(q DashboardQuery, err error) error {
	var e *fail.Error
	if errors.As(err, &e) && (e.Category == "auth" || e.Category == "network") {
		return err
	}
	return fail.NewQueryValidation(
		"dashboard",
		q.WidgetIndex,
		q.WidgetTitle,
		q.QueryIndex,
		queryWarning(q, "invalid query: "+err.Error()),
		"fix the query on the named widget",
	)
}

func (s *DashboardsService) List(ctx context.Context, limit int) ([]map[string]any, error) {
	var body json.RawMessage
	if err := s.dd.Get(ctx, "/api/v1/dashboard", &body); err != nil {
		return nil, err
	}
	raw, err := parseDashboardListItems(body)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		id := dashboardIDFrom(item)
		if id == "" {
			continue
		}
		entry := map[string]any{
			"id":    id,
			"title": item["title"],
			"url":   dashboardCanonicalURL(s.site, id),
		}
		if author, ok := item["author_handle"]; ok {
			entry["author_handle"] = author
		}
		if modified, ok := item["modified_at"]; ok {
			entry["modified_at"] = modified
		}
		if tags, ok := item["tags"]; ok {
			entry["tags"] = tags
		}
		out = append(out, entry)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func parseDashboardListItems(body []byte) ([]map[string]any, error) {
	var direct []map[string]any
	if err := json.Unmarshal(body, &direct); err == nil {
		return direct, nil
	}
	var wrapped struct {
		Dashboards []map[string]any `json:"dashboards"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fail.NewAPI("failed to decode dashboard list", "check API response shape", "")
	}
	return wrapped.Dashboards, nil
}

func (s *DashboardsService) Search(ctx context.Context, titleSubstr, tag string, limit int) ([]map[string]any, error) {
	all, err := s.List(ctx, 0)
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for _, item := range all {
		if titleSubstr != "" {
			title, _ := item["title"].(string)
			if !strings.Contains(strings.ToLower(title), strings.ToLower(titleSubstr)) {
				continue
			}
		}
		if tag != "" && !dashboardHasTag(item["tags"], tag) {
			continue
		}
		out = append(out, item)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func dashboardHasTag(raw any, want string) bool {
	tags, ok := raw.([]any)
	if !ok {
		return false
	}
	for _, item := range tags {
		if s, ok := item.(string); ok && s == want {
			return true
		}
	}
	return false
}

func (s *DashboardsService) Clone(ctx context.Context, sourceID, title string, input DashboardMutationInput) (DashboardMutationResult, error) {
	source, err := s.Get(ctx, DashboardGetInput{ID: sourceID})
	if err != nil {
		return nil, err
	}
	file, err := writeTempDashboard(source)
	if err != nil {
		return nil, err
	}
	defer os.Remove(file)
	input.FilePath = file
	input.Title = title
	return s.Create(ctx, input)
}

func writeTempDashboard(payload map[string]any) (string, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return "", fail.NewValidation("unable to serialize dashboard", err.Error())
	}
	f, err := os.CreateTemp("", "ddctl-dashboard-*.json")
	if err != nil {
		return "", fail.NewValidation("unable to create temp file", err.Error())
	}
	defer f.Close()
	if _, err := f.Write(b); err != nil {
		return "", fail.NewValidation("unable to write temp file", err.Error())
	}
	return f.Name(), nil
}

type DashboardDeleteInput struct {
	ID      string
	Confirm string
}

func (s *DashboardsService) Delete(ctx context.Context, input DashboardDeleteInput) (map[string]any, error) {
	id := strings.TrimSpace(input.ID)
	if id == "" {
		return nil, fail.NewValidation("missing dashboard ID", "usage: ddctl dashboards delete <id> --confirm <id>")
	}
	if strings.TrimSpace(input.Confirm) != id {
		return nil, fail.NewValidation("--confirm must equal dashboard ID", fmt.Sprintf("pass --confirm %s", id))
	}
	current, err := s.Get(ctx, DashboardGetInput{ID: id})
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/api/v1/dashboard/%s", id)
	var deleted map[string]any
	if err := s.dd.Delete(ctx, path, &deleted); err != nil {
		return nil, err
	}
	return map[string]any{
		"id":      id,
		"title":   current["title"],
		"url":     current["url"],
		"deleted": true,
	}, nil
}

func loadDashboardEnvelopeFromFile(path string) (map[string]any, error) {
	if path == "" {
		return nil, fail.NewValidation("--from-file is required", "provide a JSON file path")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fail.NewValidation("unable to read --from-file", err.Error())
	}
	return NormalizeDashboardPayload(raw)
}

func attachDashboardURL(payload map[string]any, site, id string) {
	if payload == nil {
		return
	}
	if id == "" {
		id = dashboardIDFrom(payload)
	}
	if id == "" {
		return
	}
	payload["url"] = dashboardCanonicalURL(site, id)
}

func dashboardCanonicalURL(site, id string) string {
	if site == "" {
		site = "datadoghq.com"
	}
	return fmt.Sprintf("https://app.%s/dashboard/%s", site, id)
}

func dashboardIDFrom(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	switch v := payload["id"].(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	default:
		if v != nil {
			return fmt.Sprint(v)
		}
	}
	return ""
}

func modifiedAtMatches(got, expected string) bool {
	got = strings.TrimSpace(got)
	expected = strings.TrimSpace(expected)
	if got == expected {
		return true
	}
	gt, gerr := parseFlexibleTime(got)
	et, eerr := parseFlexibleTime(expected)
	if gerr != nil || eerr != nil {
		return false
	}
	return gt.Equal(et)
}

func parseFlexibleTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000000+00:00",
		"2006-01-02T15:04:05Z",
	}
	var last error
	for _, f := range formats {
		ts, err := time.Parse(f, s)
		if err == nil {
			return ts, nil
		}
		last = err
	}
	return time.Time{}, last
}
