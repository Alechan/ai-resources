package service

import (
	"context"
	"fmt"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/timeutil"
)

// LogsQueryInput holds the parameters for a logs search request.
type LogsQueryInput struct {
	Query  string
	From   string
	To     string
	Limit  int
	Cursor string // pagination cursor from a previous result's NextCursor field
}

// LogEvent represents a single log event from DataDog.
type LogEvent struct {
	ID         string             `json:"id"`
	Attributes LogEventAttributes `json:"attributes"`
}

// LogEventAttributes holds the fields of a log event.
type LogEventAttributes struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Service   string `json:"service"`
	Status    string `json:"status"`
	Host      string `json:"host"`
}

// LogsQueryResult is the response from the DataDog logs search API.
type LogsQueryResult struct {
	Data       []LogEvent `json:"data"`
	NextCursor string     `json:"next_cursor,omitempty"`
}

// v1 internal response types — the browser UI endpoint structure.
type v1LogsResponse struct {
	Result struct {
		Events []v1LogEvent `json:"events"`
		Paging struct {
			After string `json:"after"`
		} `json:"paging"`
	} `json:"result"`
	HitCount int `json:"hitCount"`
}

type v1LogEvent struct {
	EventID string        `json:"event_id"`
	Columns []interface{} `json:"columns"`
	ID      string        `json:"id"`
}

// LogsQueryService queries DataDog logs.
type LogsQueryService struct {
	dd *datadogapi.Client
}

// NewLogsQueryService creates a LogsQueryService.
func NewLogsQueryService(dd *datadogapi.Client) *LogsQueryService {
	return &LogsQueryService{dd: dd}
}

// Run executes a logs query using DataDog's browser UI endpoint.
//
// Column layout returned by the API: [0]=status, [1]=timestamp, [2]=host, [3]=service, [4]=message
func (s *LogsQueryService) Run(ctx context.Context, input LogsQueryInput) (LogsQueryResult, error) {
	fromMs, err := timeutil.ParseToUnixMs(input.From)
	if err != nil {
		return LogsQueryResult{}, fail.NewValidation(
			fmt.Sprintf("invalid --from value %q: %s", input.From, err),
			`use relative (now-1h, now-30m, now-2d) or Unix milliseconds`,
		)
	}
	toMs, err := timeutil.ParseToUnixMs(input.To)
	if err != nil {
		return LogsQueryResult{}, fail.NewValidation(
			fmt.Sprintf("invalid --to value %q: %s", input.To, err),
			`use relative (now-1h, now-30m, now-2d) or Unix milliseconds`,
		)
	}

	listBody := map[string]any{
		"columns": []map[string]any{
			{"field": map[string]any{"path": "status_line"}},
			{"field": map[string]any{"path": "timestamp"}},
			{"field": map[string]any{"path": "host"}},
			{"field": map[string]any{"path": "service"}},
			{"field": map[string]any{"path": "content"}},
		},
		"sorts":                []map[string]any{{"time": map[string]any{"order": "desc"}}},
		"limit":                input.Limit,
		"time":                 map[string]any{"from": fromMs, "to": toMs},
		"includeEvents":        true,
		"includeEventContents": true,
		"computeCount":         false,
		"indexes":              []string{"*"},
		"executionInfo":        map[string]any{},
	}
	if input.Query != "" && input.Query != "*" {
		listBody["search"] = map[string]any{"query": input.Query}
	}
	if input.Cursor != "" {
		listBody["after"] = input.Cursor
	}

	body := map[string]any{
		"list":          listBody,
		"querySourceId": "logs_explorer",
	}

	var raw v1LogsResponse
	if err := s.dd.Post(ctx, "/api/v1/logs-analytics/list?type=logs", body, &raw); err != nil {
		return LogsQueryResult{}, err
	}

	result := LogsQueryResult{}
	for _, ev := range raw.Result.Events {
		attrs := LogEventAttributes{}
		if len(ev.Columns) > 0 && ev.Columns[0] != nil {
			attrs.Status, _ = ev.Columns[0].(string)
		}
		if len(ev.Columns) > 1 && ev.Columns[1] != nil {
			attrs.Timestamp, _ = ev.Columns[1].(string)
		}
		if len(ev.Columns) > 2 && ev.Columns[2] != nil {
			attrs.Host, _ = ev.Columns[2].(string)
		}
		if len(ev.Columns) > 3 && ev.Columns[3] != nil {
			attrs.Service, _ = ev.Columns[3].(string)
		}
		if len(ev.Columns) > 4 && ev.Columns[4] != nil {
			attrs.Message, _ = ev.Columns[4].(string)
		}
		result.Data = append(result.Data, LogEvent{
			ID:         ev.ID,
			Attributes: attrs,
		})
	}
	result.NextCursor = raw.Result.Paging.After
	return result, nil
}
