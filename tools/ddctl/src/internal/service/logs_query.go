package service

import (
	"context"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
)

// LogsQueryInput holds the parameters for a logs search request.
type LogsQueryInput struct {
	Query string
	From  string
	To    string
	Limit int
}

// LogEvent represents a single log event from DataDog.
type LogEvent struct {
	ID         string             `json:"id"`
	Attributes LogEventAttributes `json:"attributes"`
}

// LogEventAttributes holds the fields of a log event.
type LogEventAttributes struct {
	Timestamp string   `json:"timestamp"`
	Message   string   `json:"message"`
	Service   string   `json:"service"`
	Status    string   `json:"status"`
	Tags      []string `json:"tags"`
}

// LogsQueryResult is the response from the DataDog logs search API.
type LogsQueryResult struct {
	Data []LogEvent `json:"data"`
}

// LogsQueryService queries DataDog logs.
type LogsQueryService struct {
	dd *datadogapi.Client
}

// NewLogsQueryService creates a LogsQueryService.
func NewLogsQueryService(dd *datadogapi.Client) *LogsQueryService {
	return &LogsQueryService{dd: dd}
}

// Run executes a logs query and returns the results.
func (s *LogsQueryService) Run(ctx context.Context, input LogsQueryInput) (LogsQueryResult, error) {
	body := map[string]any{
		"filter": map[string]any{
			"query": input.Query,
			"from":  input.From,
			"to":    input.To,
		},
		"sort": "timestamp",
		"page": map[string]any{
			"limit": input.Limit,
		},
	}

	var result LogsQueryResult
	if err := s.dd.Post(ctx, "/api/v2/logs/events/search", body, &result); err != nil {
		return LogsQueryResult{}, err
	}
	return result, nil
}
