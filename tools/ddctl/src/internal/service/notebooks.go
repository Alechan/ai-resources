package service

import (
	"context"
	"fmt"
	"os"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

type NotebookGetInput struct {
	ID              string
	IncludeMetadata bool
}

type NotebookGetResult map[string]any

type NotebookMutationInput struct {
	FilePath          string
	Name              string
	Time              string
	ID                string
	SkipValidate      bool
	From              string
	To                string
	DryRun            bool
	ShowDiff          bool
	IfUnmodifiedSince string
}

type NotebookMutationResult map[string]any

type NotebookValidateInput struct {
	FilePath string
	From     string
	To       string
}

type NotebookValidateResult struct {
	QueryCount int      `json:"query_count"`
	Queries    []string `json:"queries"`
	Warnings   []string `json:"warnings"`
}

type NotebooksService struct {
	dd      *datadogapi.Client
	metrics *MetricsQueryService
	site    string
}

func NewNotebooksService(dd *datadogapi.Client, metrics *MetricsQueryService, site string) *NotebooksService {
	return &NotebooksService{dd: dd, metrics: metrics, site: site}
}

func (s *NotebooksService) Get(ctx context.Context, input NotebookGetInput) (NotebookGetResult, error) {
	if input.ID == "" {
		return nil, fail.NewValidation("missing notebook ID", "usage: ddctl notebooks get <id>")
	}
	path := fmt.Sprintf("/api/v1/notebooks/%s?include_metadata=%t", input.ID, input.IncludeMetadata)
	var out map[string]any
	if err := s.dd.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	attachNotebookURL(out, s.site, input.ID)
	return out, nil
}

func (s *NotebooksService) Create(ctx context.Context, input NotebookMutationInput) (NotebookMutationResult, error) {
	env, err := loadNotebookEnvelopeFromFile(input.FilePath)
	if err != nil {
		return nil, err
	}
	if !input.SkipValidate {
		if err := s.validatePrepared(ctx, env, input.From, input.To); err != nil {
			return nil, err
		}
	}
	payload, err := PrepareNotebookCreatePayload(env, input.Name, input.Time)
	if err != nil {
		return nil, err
	}
	if input.DryRun {
		attrs := mustMap(mustMap(payload["data"])["attributes"])
		name, _ := attrs["name"].(string)
		cells, _ := attrs["cells"].([]any)
		return NotebookMutationResult{
			"dry_run":    true,
			"name":       name,
			"cell_count": len(cells),
		}, nil
	}
	var out map[string]any
	if err := s.dd.Post(ctx, "/api/v1/notebooks", payload, &out); err != nil {
		return nil, err
	}
	attachNotebookURL(out, s.site, notebookIDFromEnvelope(out))
	return out, nil
}

func (s *NotebooksService) Update(ctx context.Context, input NotebookMutationInput, replaceAll bool) (NotebookMutationResult, error) {
	env, err := loadNotebookEnvelopeFromFile(input.FilePath)
	if err != nil {
		return nil, err
	}
	payload, err := PrepareNotebookUpdatePayload(env, input.ID, replaceAll)
	if err != nil {
		return nil, err
	}
	if !input.SkipValidate {
		if err := s.validatePrepared(ctx, env, input.From, input.To); err != nil {
			return nil, err
		}
	}

	snapshot := MutationSnapshotInput{
		IfUnmodifiedSince: input.IfUnmodifiedSince,
		DryRun:            input.DryRun,
		ShowDiff:          input.ShowDiff,
	}
	var semantic string
	if mutationNeedsRemoteSnapshot(snapshot) {
		current, err := s.Get(ctx, NotebookGetInput{ID: input.ID, IncludeMetadata: true})
		if err != nil {
			return nil, err
		}
		if err := assertModifiedAtMatches(notebookModifiedAt(current), input.IfUnmodifiedSince, "notebook"); err != nil {
			return nil, err
		}
		semantic, jsonDiff := computeMutationDiff(MutationDiffInput{
			Current:      map[string]any(current),
			Next:         payload,
			StripCurrent: stripNotebookIdentity,
			StripNext:    stripNotebookIdentity,
			Semantic:     SemanticDiffNotebookPayloads,
		})
		if input.DryRun {
			return NotebookMutationResult{
				"dry_run":   true,
				"id":        input.ID,
				"url":       notebookCanonicalURL(s.site, input.ID),
				"diff":      semantic,
				"json_diff": jsonDiff,
			}, nil
		}
	}

	path := fmt.Sprintf("/api/v1/notebooks/%s", input.ID)
	var out map[string]any
	if err := s.dd.Put(ctx, path, payload, &out); err != nil {
		return nil, err
	}
	attachNotebookURL(out, s.site, input.ID)
	if semantic != "" {
		out["diff"] = semantic
	}
	return out, nil
}

func (s *NotebooksService) Validate(ctx context.Context, input NotebookValidateInput) (NotebookValidateResult, error) {
	env, err := loadNotebookEnvelopeFromFile(input.FilePath)
	if err != nil {
		return NotebookValidateResult{}, err
	}
	queries, err := ExtractTimeseriesQueries(env)
	if err != nil {
		return NotebookValidateResult{}, err
	}
	result := NotebookValidateResult{
		QueryCount: len(queries),
		Queries:    queries,
	}

	for _, q := range queries {
		metricsResult, err := s.metrics.Run(ctx, MetricsQueryInput{
			Query: q,
			From:  input.From,
			To:    input.To,
		})
		if err != nil {
			return NotebookValidateResult{}, err
		}
		if len(metricsResult.Series) == 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("query returned no data: %s", q))
		}
	}
	return result, nil
}

func (s *NotebooksService) validatePrepared(ctx context.Context, env map[string]any, from, to string) error {
	if from == "" {
		from = "now-30d"
	}
	if to == "" {
		to = "now"
	}
	queries, err := ExtractTimeseriesQueries(env)
	if err != nil {
		return err
	}
	for _, q := range queries {
		if _, err := s.metrics.Run(ctx, MetricsQueryInput{
			Query: q,
			From:  from,
			To:    to,
		}); err != nil {
			return err
		}
	}
	return nil
}

func loadNotebookEnvelopeFromFile(path string) (map[string]any, error) {
	if path == "" {
		return nil, fail.NewValidation("--from-file is required", "provide a JSON file path")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fail.NewValidation("unable to read --from-file", err.Error())
	}
	return NormalizeNotebookEnvelope(raw)
}
