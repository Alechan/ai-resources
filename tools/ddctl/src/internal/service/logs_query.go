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
	Event   *v1EventBody  `json:"event"`
}

// v1EventBody is the full log event body returned in the "event" field.
// Columns are often null; this carries the reliable values.
type v1EventBody struct {
	Status    string         `json:"status"`
	Timestamp string         `json:"timestamp"`
	Host      string         `json:"host"`
	Service   string         `json:"service"`
	Custom    map[string]any `json:"custom"`
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
// Column values are used when present; event body fields are used as fallback.
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
		attrs := extractEventAttrs(ev)
		result.Data = append(result.Data, LogEvent{
			ID:         ev.ID,
			Attributes: attrs,
		})
	}
	result.NextCursor = raw.Result.Paging.After
	return result, nil
}

// extractEventAttrs builds LogEventAttributes from a v1 event, preferring
// column values but falling back to the event body when columns are null.
func extractEventAttrs(ev v1LogEvent) LogEventAttributes {
	colStr := func(i int) string {
		if i < len(ev.Columns) && ev.Columns[i] != nil {
			s, _ := ev.Columns[i].(string)
			return s
		}
		return ""
	}

	attrs := LogEventAttributes{
		Status:    colStr(0),
		Timestamp: colStr(1),
		Host:      colStr(2),
		Service:   colStr(3),
		Message:   colStr(4),
	}

	// Fall back to event body fields when columns are null.
	if ev.Event != nil {
		if attrs.Status == "" {
			attrs.Status = ev.Event.Status
		}
		if attrs.Timestamp == "" {
			attrs.Timestamp = ev.Event.Timestamp
		}
		if attrs.Host == "" {
			attrs.Host = ev.Event.Host
		}
		if attrs.Service == "" {
			attrs.Service = ev.Event.Service
		}
		if attrs.Message == "" {
			attrs.Message = extractMessage(ev.Event.Custom)
		}
	}
	return attrs
}

// extractMessage finds the log message from the custom fields map.
// Different services use different field names.
func extractMessage(custom map[string]any) string {
	for _, key := range []string{"message", "msg", "MESSAGE", "log", "body", "text"} {
		if v, ok := custom[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}
