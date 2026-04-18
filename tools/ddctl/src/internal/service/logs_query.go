package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
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
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Service   string `json:"service"`
	Status    string `json:"status"`
	Host      string `json:"host"`
}

// LogsQueryResult is the response from the DataDog logs search API.
type LogsQueryResult struct {
	Data []LogEvent `json:"data"`
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
	fromMs, err := parseTimeToMs(input.From)
	if err != nil {
		return LogsQueryResult{}, fail.NewValidation(
			fmt.Sprintf("invalid --from value %q: %s", input.From, err),
			`use relative (now-1h, now-30m, now-2d) or Unix milliseconds`,
		)
	}
	toMs, err := parseTimeToMs(input.To)
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
		"sorts":               []map[string]any{{"time": map[string]any{"order": "desc"}}},
		"limit":               input.Limit,
		"time":                map[string]any{"from": fromMs, "to": toMs},
		"includeEvents":       true,
		"includeEventContents": true,
		"computeCount":        false,
		"indexes":             []string{"*"},
		"executionInfo":       map[string]any{},
	}
	if input.Query != "" && input.Query != "*" {
		listBody["search"] = map[string]any{"query": input.Query}
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
	return result, nil
}

// parseTimeToMs converts a time expression to Unix milliseconds.
// Accepts: "now", "now-1h", "now-30m", "now-2d", Unix ms integers, or ISO-8601.
func parseTimeToMs(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "now" {
		return time.Now().UnixMilli(), nil
	}
	if strings.HasPrefix(s, "now-") {
		suffix := s[4:]
		var n int64
		var unit string
		for i, c := range suffix {
			if c < '0' || c > '9' {
				n, _ = strconv.ParseInt(suffix[:i], 10, 64)
				unit = suffix[i:]
				break
			}
		}
		if n == 0 {
			return 0, fmt.Errorf("invalid relative duration %q", s)
		}
		var d time.Duration
		switch unit {
		case "m":
			d = time.Duration(n) * time.Minute
		case "h":
			d = time.Duration(n) * time.Hour
		case "d":
			d = time.Duration(n) * 24 * time.Hour
		default:
			return 0, fmt.Errorf("unknown unit %q (use m, h, d)", unit)
		}
		return time.Now().Add(-d).UnixMilli(), nil
	}
	// Try Unix milliseconds
	if ms, err := strconv.ParseInt(s, 10, 64); err == nil {
		return ms, nil
	}
	// Try ISO-8601
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0, fmt.Errorf("cannot parse %q as relative, Unix ms, or RFC3339", s)
	}
	return t.UnixMilli(), nil
}
