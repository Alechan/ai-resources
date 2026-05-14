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
	FilePath string
	Name     string
	Time     string
	ID       string
}

type NotebookMutationResult map[string]any

type NotebookValidateInput struct {
	FilePath         string
	From             string
	To               string
	AllowEmptySeries bool
}

type NotebookValidateResult struct {
	QueryCount       int      `json:"query_count"`
	Queries          []string `json:"queries"`
	Warnings         []string `json:"warnings"`
	AllowEmptySeries bool     `json:"allow_empty_series"`
}

type NotebooksService struct {
	dd      *datadogapi.Client
	metrics *MetricsQueryService
}

func NewNotebooksService(dd *datadogapi.Client, metrics *MetricsQueryService) *NotebooksService {
	return &NotebooksService{dd: dd, metrics: metrics}
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
	return out, nil
}

func (s *NotebooksService) Create(ctx context.Context, input NotebookMutationInput) (NotebookMutationResult, error) {
	env, err := loadNotebookEnvelopeFromFile(input.FilePath)
	if err != nil {
		return nil, err
	}
	payload, err := PrepareNotebookCreatePayload(env, input.Name, input.Time)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := s.dd.Post(ctx, "/api/v1/notebooks", payload, &out); err != nil {
		return nil, err
	}
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
	path := fmt.Sprintf("/api/v1/notebooks/%s", input.ID)
	var out map[string]any
	if err := s.dd.Put(ctx, path, payload, &out); err != nil {
		return nil, err
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
		QueryCount:       len(queries),
		Queries:          queries,
		AllowEmptySeries: input.AllowEmptySeries,
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
			msg := fmt.Sprintf("query returned no data: %s", q)
			if !input.AllowEmptySeries {
				return NotebookValidateResult{}, fail.NewValidation(msg, "fix tags/wildcards or pass --allow-empty-series")
			}
			result.Warnings = append(result.Warnings, msg)
		}
	}
	return result, nil
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
