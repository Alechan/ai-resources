package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

type MonitorSummary struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	OverallState string   `json:"overall_state"`
	Tags         []string `json:"tags"`
	URL          string   `json:"url"`
	Modified     string   `json:"modified,omitempty"`
}

type MonitorsListResult struct {
	Monitors []MonitorSummary `json:"monitors"`
}

type MonitorGetResult map[string]any

type MonitorMutationInput struct {
	FilePath           string
	ID                 int64
	SkipValidate       bool
	From               string
	To                 string
	DryRun             bool
	ShowDiff           bool
	IfUnmodifiedSince  string
	Muted              bool
}

type MonitorValidateInput struct {
	FilePath string
	From     string
	To       string
}

type MonitorValidateResult struct {
	Structure     string   `json:"structure"`
	Query         string   `json:"query"`
	QueryValid    bool     `json:"query_valid"`
	QueryNoData   bool     `json:"query_no_data"`
	RemoteValid   bool     `json:"remote_valid"`
	Warnings      []string `json:"warnings"`
}

type MonitorMuteInput struct {
	ID      int64
	Until   string
	Confirm string
}

type MonitorDeleteInput struct {
	ID      int64
	Confirm string
}

type MonitorsService struct {
	dd      *datadogapi.Client
	metrics *MetricsQueryService
	site    string
}

func NewMonitorsService(dd *datadogapi.Client, metrics *MetricsQueryService, site string) *MonitorsService {
	return &MonitorsService{dd: dd, metrics: metrics, site: site}
}

func (s *MonitorsService) List(ctx context.Context, tagFilter string) (MonitorsListResult, error) {
	var all []MonitorSummary
	page := 0
	for {
		path := fmt.Sprintf("/api/v1/monitor?with_downtimes=true&page=%d&page_size=100", page)
		var batch []map[string]any
		if err := s.dd.Get(ctx, path, &batch); err != nil {
			return MonitorsListResult{}, err
		}
		for _, raw := range batch {
			id := monitorIDFrom(raw)
			if id <= 0 {
				continue
			}
			name, _ := raw["name"].(string)
			mtype, _ := raw["type"].(string)
			state, _ := raw["overall_state"].(string)
			tags := monitorTags(raw)
			if tagFilter != "" && !containsTag(tags, tagFilter) {
				continue
			}
			all = append(all, MonitorSummary{
				ID:           id,
				Name:         name,
				Type:         mtype,
				OverallState: state,
				Tags:         tags,
				URL:          monitorCanonicalURL(s.site, id),
				Modified:     monitorModifiedAt(raw),
			})
		}
		if len(batch) < 100 {
			break
		}
		page++
	}
	return MonitorsListResult{Monitors: all}, nil
}

func containsTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}

func (s *MonitorsService) Get(ctx context.Context, id int64) (MonitorGetResult, error) {
	if id <= 0 {
		return nil, fail.NewValidation("missing monitor ID", "usage: ddctl monitors get <id>")
	}
	path := fmt.Sprintf("/api/v1/monitor/%d", id)
	var out map[string]any
	if err := s.dd.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	attachMonitorURL(out, s.site, id)
	return out, nil
}

func (s *MonitorsService) Validate(ctx context.Context, input MonitorValidateInput) (MonitorValidateResult, error) {
	env, err := loadMonitorFromFile(input.FilePath)
	if err != nil {
		return MonitorValidateResult{}, err
	}
	if err := assertMonitorRequired(env); err != nil {
		return MonitorValidateResult{}, err
	}
	from, to := input.From, input.To
	if from == "" {
		from = "now-24h"
	}
	if to == "" {
		to = "now"
	}
	result := MonitorValidateResult{
		Structure: "valid",
		Query:     monitorQueryFromPayload(env),
	}
	if _, ok := env["options"].(map[string]any); !ok {
		result.Warnings = append(result.Warnings, "missing options block")
	}
	metricsResult, err := s.metrics.Run(ctx, MetricsQueryInput{
		Query: metricQueryForPreflight(result.Query),
		From:  from,
		To:    to,
	})
	if err != nil {
		return MonitorValidateResult{}, fail.NewResourceValidation("monitor", "invalid query: "+err.Error(), "fix the monitor query")
	}
	result.QueryValid = true
	if len(metricsResult.Series) == 0 {
		result.QueryNoData = true
		result.Warnings = append(result.Warnings, "query valid; series currently absent in selected window")
	}
	payload, err := PrepareMonitorCreatePayload(env)
	if err != nil {
		return MonitorValidateResult{}, err
	}
	var remote map[string]any
	if err := s.dd.Post(ctx, "/api/v1/monitor/validate", payload, &remote); err != nil {
		return MonitorValidateResult{}, err
	}
	result.RemoteValid = true
	return result, nil
}

func (s *MonitorsService) Create(ctx context.Context, input MonitorMutationInput) (MonitorGetResult, error) {
	env, err := loadMonitorFromFile(input.FilePath)
	if err != nil {
		return nil, err
	}
	payload, err := PrepareMonitorCreatePayload(env)
	if err != nil {
		return nil, err
	}
	if !input.SkipValidate {
		if _, err := s.Validate(ctx, MonitorValidateInput{FilePath: input.FilePath, From: input.From, To: input.To}); err != nil {
			return nil, err
		}
	}
	if input.DryRun {
		name, _ := payload["name"].(string)
		return MonitorGetResult{"dry_run": true, "name": name, "type": payload["type"]}, nil
	}
	var out map[string]any
	if err := s.dd.Post(ctx, "/api/v1/monitor", payload, &out); err != nil {
		return nil, err
	}
	id := monitorIDFrom(out)
	attachMonitorURL(out, s.site, id)
	if input.Muted && id > 0 {
		if err := s.mute(ctx, id, ""); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *MonitorsService) Update(ctx context.Context, input MonitorMutationInput, replaceAll bool) (MonitorGetResult, error) {
	env, err := loadMonitorFromFile(input.FilePath)
	if err != nil {
		return nil, err
	}
	payload, err := PrepareMonitorUpdatePayload(env, input.ID, replaceAll)
	if err != nil {
		return nil, err
	}
	if !input.SkipValidate {
		if _, err := s.Validate(ctx, MonitorValidateInput{FilePath: input.FilePath, From: input.From, To: input.To}); err != nil {
			return nil, err
		}
	}
	var semantic string
	if mutationNeedsRemoteSnapshot(MutationSnapshotInput{
		IfUnmodifiedSince: input.IfUnmodifiedSince,
		DryRun:            input.DryRun,
		ShowDiff:          input.ShowDiff,
	}) {
		current, err := s.Get(ctx, input.ID)
		if err != nil {
			return nil, err
		}
		if err := assertModifiedAtMatches(monitorModifiedAt(current), input.IfUnmodifiedSince, "monitor"); err != nil {
			return nil, err
		}
		stripIdentity := func(m map[string]any) {
			for _, k := range monitorIdentityFields {
				delete(m, k)
			}
		}
		semantic, _ = computeMutationDiff(MutationDiffInput{
			Current:      map[string]any(current),
			Next:         payload,
			StripCurrent: stripIdentity,
			StripNext:    stripIdentity,
			Semantic:     func(left, right map[string]any) string { return DiffMonitorPayloads(left, right) },
			JSONDiff:     func(left, right map[string]any) string { return DiffMonitorPayloads(left, right) },
		})
		if input.DryRun {
			return MonitorGetResult{
				"dry_run": true,
				"id":      input.ID,
				"url":     monitorCanonicalURL(s.site, input.ID),
				"diff":    semantic,
			}, nil
		}
	}
	path := fmt.Sprintf("/api/v1/monitor/%d", input.ID)
	var out map[string]any
	if err := s.dd.Put(ctx, path, payload, &out); err != nil {
		return nil, err
	}
	attachMonitorURL(out, s.site, input.ID)
	if semantic != "" {
		out["diff"] = semantic
	}
	return out, nil
}

func (s *MonitorsService) Mute(ctx context.Context, input MonitorMuteInput) (MonitorGetResult, error) {
	if input.ID <= 0 {
		return nil, fail.NewValidation("missing monitor ID", "usage: ddctl monitors mute <id>")
	}
	current, err := s.Get(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if err := s.mute(ctx, input.ID, input.Until); err != nil {
		return nil, err
	}
	out, err := s.Get(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	out["previous_overall_state"] = current["overall_state"]
	return out, nil
}

func (s *MonitorsService) Unmute(ctx context.Context, input MonitorMuteInput) (MonitorGetResult, error) {
	if input.ID <= 0 {
		return nil, fail.NewValidation("missing monitor ID", "usage: ddctl monitors unmute <id>")
	}
	current, err := s.Get(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	tags := monitorTags(current)
	if isProductionMonitor(tags) {
		confirm := strings.TrimSpace(input.Confirm)
		want := strconv.FormatInt(input.ID, 10)
		if confirm != want {
			return nil, fail.NewValidation(
				"production monitor requires --confirm <id>",
				fmt.Sprintf("pass --confirm %s to unmute tags %v", want, tags),
			)
		}
	}
	path := fmt.Sprintf("/api/v1/monitor/%d/unmute", input.ID)
	var out map[string]any
	if err := s.dd.Post(ctx, path, map[string]any{}, &out); err != nil {
		return nil, err
	}
	return s.Get(ctx, input.ID)
}

func (s *MonitorsService) Delete(ctx context.Context, input MonitorDeleteInput) (MonitorGetResult, error) {
	if input.ID <= 0 {
		return nil, fail.NewValidation("missing monitor ID", "usage: ddctl monitors delete <id> --confirm <id>")
	}
	want := strconv.FormatInt(input.ID, 10)
	if strings.TrimSpace(input.Confirm) != want {
		return nil, fail.NewValidation("--confirm must equal monitor ID", fmt.Sprintf("pass --confirm %s", want))
	}
	current, err := s.Get(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/api/v1/monitor/%d", input.ID)
	var deleted map[string]any
	if err := s.dd.Delete(ctx, path, &deleted); err != nil {
		return nil, err
	}
	name, _ := current["name"].(string)
	url, _ := current["url"].(string)
	return MonitorGetResult{
		"id":      input.ID,
		"name":    name,
		"url":     url,
		"deleted": true,
	}, nil
}

func (s *MonitorsService) mute(ctx context.Context, id int64, until string) error {
	body := map[string]any{}
	if strings.TrimSpace(until) != "" {
		ts, err := parseFlexibleTime(until)
		if err != nil {
			return fail.NewValidation("invalid --until value", "use RFC3339 timestamp")
		}
		body["end"] = ts.Unix()
	}
	path := fmt.Sprintf("/api/v1/monitor/%d/mute", id)
	return s.dd.Post(ctx, path, body, &map[string]any{})
}

func loadMonitorFromFile(path string) (map[string]any, error) {
	if path == "" {
		return nil, fail.NewValidation("--from-file is required", "provide a JSON file path")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fail.NewValidation("unable to read --from-file", err.Error())
	}
	return NormalizeMonitorPayload(raw)
}
