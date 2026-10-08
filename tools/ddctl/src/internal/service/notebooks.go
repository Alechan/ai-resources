package service

import (
	"context"
	"fmt"
	"os"
	"strings"

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
	Force             bool
}

type NotebookMutationResult map[string]any

type NotebookValidateInput struct {
	FilePath string
	From     string
	To       string
}

type NotebookQueryReport struct {
	CellIndex    int    `json:"cell_index"`
	RequestIndex int    `json:"request_index"`
	Original     string `json:"original"`
	Resolved     string `json:"resolved"`
	NoData       bool   `json:"no_data,omitempty"`
}

type NotebookValidateResult struct {
	Structure  string                `json:"structure"`
	QueryCount int                   `json:"query_count"`
	Queries    []NotebookQueryReport `json:"queries"`
	Warnings   []string              `json:"warnings"`
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
		prepared := cloneNotebookEnvelope(env)
		if _, err := PrepareNotebookSchema(prepared); err != nil {
			return nil, err
		}
		return NotebookMutationResult{
			"dry_run":    true,
			"name":       notebookName(prepared),
			"cell_count": notebookCellCount(prepared),
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

	needRemote := input.DryRun || input.ShowDiff || !input.Force
	var semantic string
	if needRemote {
		current, err := s.Get(ctx, NotebookGetInput{ID: input.ID, IncludeMetadata: true})
		if err != nil {
			return nil, err
		}
		if !input.Force {
			expected := strings.TrimSpace(input.IfUnmodifiedSince)
			if expected == "" {
				expected = notebookRevisionFromEnvelope(env)
			}
			if err := assertNotebookRevisionMatches(notebookModifiedAt(current), expected, input.ID); err != nil {
				return nil, err
			}
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
	return s.validateEnvelope(ctx, env, input.From, input.To)
}

func (s *NotebooksService) validatePrepared(ctx context.Context, env map[string]any, from, to string) error {
	_, err := s.validateEnvelope(ctx, env, from, to)
	return err
}

func (s *NotebooksService) validateEnvelope(ctx context.Context, env map[string]any, from, to string) (NotebookValidateResult, error) {
	if from == "" {
		from = "now-30d"
	}
	if to == "" {
		to = "now"
	}
	warnings, err := PrepareNotebookSchema(env)
	if err != nil {
		return NotebookValidateResult{}, err
	}
	queries, err := ExtractNotebookMetricQueries(env)
	if err != nil {
		return NotebookValidateResult{}, err
	}
	result := NotebookValidateResult{
		Structure:  "valid",
		QueryCount: len(queries),
		Warnings:   append([]string(nil), warnings...),
	}
	for _, q := range queries {
		report := NotebookQueryReport{
			CellIndex:    q.CellIndex,
			RequestIndex: q.RequestIndex,
			Original:     q.Original,
			Resolved:     q.Resolved,
		}
		metricsResult, err := s.metrics.Run(ctx, MetricsQueryInput{
			Query: q.Resolved,
			From:  from,
			To:    to,
		})
		if err != nil {
			return NotebookValidateResult{}, fail.NewResourceValidation(
				"notebook",
				fmt.Sprintf("cell[%d] request[%d] invalid query: %s", q.CellIndex, q.RequestIndex, err.Error()),
				"fix the metric query or template variable defaults",
			)
		}
		if len(metricsResult.Series) == 0 {
			report.NoData = true
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"cell[%d] request[%d] query returned no data: original=%q resolved=%q",
				q.CellIndex, q.RequestIndex, q.Original, q.Resolved,
			))
		}
		result.Queries = append(result.Queries, report)
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
