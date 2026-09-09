package service

import (
	"context"
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
	FilePath             string
	Title                string
	ID                   string
	SkipValidate         bool
	AllowEmptySeries     bool
	From                 string
	To                   string
	DryRun               bool
	ExpectedModifiedAt   string
}

type DashboardMutationResult map[string]any

type DashboardValidateInput struct {
	FilePath         string
	From             string
	To               string
	AllowEmptySeries bool
}

type DashboardValidateResult struct {
	QueryCount       int               `json:"query_count"`
	Metrics          []string          `json:"metrics"`
	Logs             []string          `json:"logs"`
	Skipped          []DashboardQuery  `json:"skipped"`
	Warnings         []string          `json:"warnings"`
	AllowEmptySeries bool              `json:"allow_empty_series"`
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
		if _, err := s.validatePrepared(ctx, payload, input.From, input.To, input.AllowEmptySeries); err != nil {
			return nil, err
		}
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
		if _, err := s.validatePrepared(ctx, payload, input.From, input.To, input.AllowEmptySeries); err != nil {
			return nil, err
		}
	}

	if strings.TrimSpace(input.ExpectedModifiedAt) != "" || input.DryRun {
		current, err := s.Get(ctx, DashboardGetInput{ID: input.ID})
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(input.ExpectedModifiedAt) != "" {
			got := fmt.Sprint(current["modified_at"])
			if !modifiedAtMatches(got, input.ExpectedModifiedAt) {
				return nil, fail.NewValidation(
					"dashboard modified_at does not match --expected-modified-at",
					fmt.Sprintf("remote modified_at is %s", got),
				)
			}
		}
		if input.DryRun {
			left := cloneMap(map[string]any(current))
			right := cloneMap(payload)
			for _, k := range dashboardIdentityFields {
				delete(left, k)
				delete(right, k)
			}
			diff := DiffDashboardPayloads(left, right)
			return DashboardMutationResult{
				"dry_run": true,
				"id":      input.ID,
				"url":     dashboardCanonicalURL(s.site, input.ID),
				"diff":    diff,
			}, nil
		}
	}

	path := fmt.Sprintf("/api/v1/dashboard/%s", input.ID)
	var out map[string]any
	if err := s.dd.Put(ctx, path, payload, &out); err != nil {
		return nil, err
	}
	attachDashboardURL(out, s.site, input.ID)
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
	return s.validatePrepared(ctx, env, input.From, input.To, input.AllowEmptySeries)
}

func (s *DashboardsService) validatePrepared(ctx context.Context, env map[string]any, from, to string, allowEmpty bool) (DashboardValidateResult, error) {
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
		Skipped:          extracted.Skipped,
		AllowEmptySeries: allowEmpty,
	}
	for _, q := range extracted.Metrics {
		result.Metrics = append(result.Metrics, q.Query)
	}
	for _, q := range extracted.Logs {
		result.Logs = append(result.Logs, q.Query)
	}
	result.QueryCount = len(result.Metrics) + len(result.Logs)
	for _, q := range extracted.Skipped {
		msg := q.SkippedReason
		if q.WidgetType != "" {
			msg = q.WidgetType + ": " + q.SkippedReason
		}
		result.Warnings = append(result.Warnings, msg)
	}

	for _, q := range extracted.Metrics {
		metricsResult, err := s.metrics.Run(ctx, MetricsQueryInput{Query: q.Query, From: from, To: to})
		if err != nil {
			return DashboardValidateResult{}, err
		}
		if len(metricsResult.Series) == 0 {
			msg := fmt.Sprintf("query returned no data: %s", q.Query)
			if !allowEmpty {
				return DashboardValidateResult{}, fail.NewValidation(msg, "fix tags/wildcards or pass --allow-empty-series")
			}
			result.Warnings = append(result.Warnings, msg)
		}
	}
	for _, q := range extracted.Logs {
		logsResult, err := s.logs.Run(ctx, LogsQueryInput{
			Query:     q.Query,
			From:      from,
			To:        to,
			Limit:     1,
			CountOnly: true,
		})
		if err != nil {
			return DashboardValidateResult{}, err
		}
		if logsResult.HitCount == 0 {
			msg := fmt.Sprintf("query returned no data: %s", q.Query)
			if !allowEmpty {
				return DashboardValidateResult{}, fail.NewValidation(msg, "fix tags/wildcards or pass --allow-empty-series")
			}
			result.Warnings = append(result.Warnings, msg)
		}
	}
	return result, nil
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
