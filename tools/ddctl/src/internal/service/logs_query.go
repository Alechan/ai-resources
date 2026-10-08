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
	Query     string
	From      string
	To        string
	Limit     int
	Cursor    string
	CountOnly bool
}

// LogEvent represents a single log event from DataDog.
type LogEvent struct {
	ID        string         `json:"id"`
	Timestamp string         `json:"timestamp"`
	Status    string         `json:"status"`
	Service   string         `json:"service"`
	Host      string         `json:"host"`
	Message   string         `json:"message"`
	Custom    map[string]any `json:"custom,omitempty"`
	Tags      []string       `json:"tags,omitempty"`
}

// LogsQueryResult is the response from the DataDog logs search API.
type LogsQueryResult struct {
	Data          []LogEvent `json:"data,omitempty"`
	NextCursor    string     `json:"next_cursor,omitempty"`
	HitCount      int        `json:"hit_count"`
	Warnings      []string   `json:"warnings,omitempty"`
	Truncated     bool       `json:"truncated,omitempty"`
	ReturnedCount int        `json:"returned_count,omitempty"`
	Limit         int        `json:"limit,omitempty"`
}

// LogsGetInput fetches a single log event by ID.
type LogsGetInput struct {
	ID    string
	From  string
	To    string
	Query string
}

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
	EventID string       `json:"event_id"`
	Columns []any        `json:"columns"`
	ID      string       `json:"id"`
	Event   *v1EventBody `json:"event"`
}

type v1EventBody struct {
	Status    string         `json:"status"`
	Timestamp string         `json:"timestamp"`
	Host      string         `json:"host"`
	Service   string         `json:"service"`
	Tags      []string       `json:"tags"`
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
func (s *LogsQueryService) Run(ctx context.Context, input LogsQueryInput) (LogsQueryResult, error) {
	fromMs, toMs, err := parseLogsWindow(input.From, input.To)
	if err != nil {
		return LogsQueryResult{}, err
	}
	if input.Limit <= 0 {
		input.Limit = 50
	}

	raw, err := s.postLogsList(ctx, input, fromMs, toMs)
	if err != nil {
		return LogsQueryResult{}, err
	}
	return buildLogsQueryResult(raw, input.CountOnly), nil
}

// RunAll paginates until the cursor ends or maxTotal is reached.
func (s *LogsQueryService) RunAll(ctx context.Context, input LogsQueryInput, maxTotal int) (LogsQueryResult, error) {
	if maxTotal <= 0 {
		maxTotal = 1000
	}
	const pageSize = 50

	var allEvents []LogEvent
	warningsSet := make(map[string]struct{})
	hitCount := 0
	truncated := false
	cursor := input.Cursor

	for {
		remaining := maxTotal - len(allEvents)
		if remaining <= 0 {
			truncated = true
			break
		}
		batchLimit := pageSize
		if remaining < batchLimit {
			batchLimit = remaining
		}
		batchInput := input
		batchInput.Limit = batchLimit
		batchInput.Cursor = cursor
		result, err := s.Run(ctx, batchInput)
		if err != nil {
			return LogsQueryResult{}, err
		}
		if result.HitCount > hitCount {
			hitCount = result.HitCount
		}
		for _, warning := range result.Warnings {
			warningsSet[warning] = struct{}{}
		}
		allEvents = append(allEvents, result.Data...)
		if result.NextCursor == "" || len(result.Data) == 0 {
			if len(allEvents) >= maxTotal && hitCount > len(allEvents) {
				truncated = true
			}
			break
		}
		cursor = result.NextCursor
	}

	warnings := make([]string, 0, len(warningsSet))
	for warning := range warningsSet {
		warnings = append(warnings, warning)
	}
	out := LogsQueryResult{
		Data:          allEvents,
		HitCount:      hitCount,
		Warnings:      warnings,
		ReturnedCount: len(allEvents),
	}
	if truncated {
		out.Truncated = true
		out.Limit = maxTotal
	}
	return out, nil
}

// Get fetches a single log event by ID.
func (s *LogsQueryService) Get(ctx context.Context, input LogsGetInput) (LogEvent, error) {
	if input.ID == "" {
		return LogEvent{}, fail.NewValidation("missing log event ID", "usage: ddctl logs get <event_id> --from <time> --to <time>")
	}
	query := input.Query
	if query == "" {
		query = fmt.Sprintf("@evt.id:%s", input.ID)
	}
	result, err := s.Run(ctx, LogsQueryInput{
		Query: query,
		From:  input.From,
		To:    input.To,
		Limit: 50,
	})
	if err != nil {
		return LogEvent{}, err
	}
	var matches []LogEvent
	for _, event := range result.Data {
		if event.ID == input.ID {
			matches = append(matches, event)
		}
	}
	switch len(matches) {
	case 0:
		return LogEvent{}, fail.NewValidation(
			fmt.Sprintf("log event %q not found", input.ID),
			"widen --from/--to or verify the event ID from logs query output",
		)
	case 1:
		return matches[0], nil
	default:
		return LogEvent{}, fail.NewValidation(
			fmt.Sprintf("log event %q matched %d events", input.ID, len(matches)),
			"narrow the time window or refine the lookup query",
		)
	}
}

func parseLogsWindow(from, to string) (int64, int64, error) {
	fromMs, err := timeutil.ParseToUnixMs(from)
	if err != nil {
		return 0, 0, fail.NewValidation(
			fmt.Sprintf("invalid --from value %q: %s", from, err),
			`use relative (now-1h, now-30m, now-2d) or Unix milliseconds`,
		)
	}
	toMs, err := timeutil.ParseToUnixMs(to)
	if err != nil {
		return 0, 0, fail.NewValidation(
			fmt.Sprintf("invalid --to value %q: %s", to, err),
			`use relative (now-1h, now-30m, now-2d) or Unix milliseconds`,
		)
	}
	return fromMs, toMs, nil
}

func (s *LogsQueryService) postLogsList(ctx context.Context, input LogsQueryInput, fromMs, toMs int64) (v1LogsResponse, error) {
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
		"includeEvents":        !input.CountOnly,
		"includeEventContents": !input.CountOnly,
		"computeCount":         true,
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
		return v1LogsResponse{}, err
	}
	return raw, nil
}

func buildLogsQueryResult(raw v1LogsResponse, countOnly bool) LogsQueryResult {
	result := LogsQueryResult{HitCount: raw.HitCount}
	if !countOnly {
		for _, ev := range raw.Result.Events {
			result.Data = append(result.Data, mapLogEvent(ev))
		}
		result.ReturnedCount = len(result.Data)
		result.NextCursor = raw.Result.Paging.After
	}
	if raw.HitCount == 0 && len(raw.Result.Events) > 0 {
		result.Warnings = append(result.Warnings,
			"hitCount is 0 but events were returned; these may be housekeeping/non-matching rows")
	}
	return result
}

func mapLogEvent(ev v1LogEvent) LogEvent {
	colStr := func(i int) string {
		if i < len(ev.Columns) && ev.Columns[i] != nil {
			s, _ := ev.Columns[i].(string)
			return s
		}
		return ""
	}

	event := LogEvent{ID: ev.ID}
	event.Status = colStr(0)
	event.Timestamp = colStr(1)
	event.Host = colStr(2)
	event.Service = colStr(3)

	if ev.Event != nil {
		if event.Status == "" {
			event.Status = ev.Event.Status
		}
		if event.Timestamp == "" {
			event.Timestamp = ev.Event.Timestamp
		}
		if event.Host == "" {
			event.Host = ev.Event.Host
		}
		if event.Service == "" {
			event.Service = ev.Event.Service
		}
		if len(ev.Event.Tags) > 0 {
			event.Tags = append([]string(nil), ev.Event.Tags...)
		}
		if len(ev.Event.Custom) > 0 {
			event.Custom = cloneCustomMap(ev.Event.Custom)
		}
	}

	message := extractMessage(event.Custom)
	if message == "" {
		message = colStr(4)
	}
	event.Message = message
	return event
}

func cloneCustomMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func extractMessage(custom map[string]any) string {
	for _, key := range []string{"message", "msg", "MESSAGE", "error", "text", "body", "log"} {
		if v, ok := custom[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}
