package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/timeutil"
)

// EventsListInput holds parameters for an events list request.
type EventsListInput struct {
	From      string
	To        string
	Sources   string // comma-separated sources filter
	Tags      string // comma-separated tags filter
	Limit     int
	Cursor    string
	CountOnly bool
}

// Event represents a DataDog event from the Event Explorer.
type Event struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Text         string   `json:"text"`
	Timestamp    int64    `json:"timestamp"`
	Priority     string   `json:"priority"`
	AlertType    string   `json:"alert_type"`
	Host         string   `json:"host"`
	Source       string   `json:"source"`
	Tags         []string `json:"tags"`
	Service      string   `json:"service"`
	Status       string   `json:"status"`
	EventType    string   `json:"event_type"`
	AggregationKey string `json:"aggregation_key,omitempty"`
}

// EventsListResult is the response for listing events.
type EventsListResult struct {
	Events        []Event  `json:"events"`
	NextCursor    string   `json:"next_cursor,omitempty"`
	HitCount      int      `json:"hit_count"`
	ReturnedCount int      `json:"returned_count"`
	Warnings      []string `json:"warnings,omitempty"`
}

// v1 feed response types — same shape as logs-analytics but for type=feed.
type v1FeedResponse struct {
	Result struct {
		Events []v1FeedEvent `json:"events"`
		Paging struct {
			After string `json:"after"`
		} `json:"paging"`
	} `json:"result"`
	HitCount int `json:"hitCount"`
}

type v1FeedEvent struct {
	EventID string        `json:"event_id"`
	Columns []interface{} `json:"columns"`
	ID      string        `json:"id"`
	Event   *v1FeedBody   `json:"event"`
}

type v1FeedBody struct {
	Source             string         `json:"source"`
	Message            string         `json:"message"`
	Host               string         `json:"host"`
	Env                string         `json:"env"`
	Tags               []string       `json:"tags"`
	DiscoveryTimestamp int64          `json:"discovery_timestamp"`
	Custom             map[string]any `json:"custom"`
}

// EventsListService lists DataDog events using the browser Event Explorer endpoint.
// Uses /api/v1/logs-analytics/list?type=feed which accepts session-cookie auth.
type EventsListService struct {
	dd *datadogapi.Client
}

// NewEventsListService creates an EventsListService.
func NewEventsListService(dd *datadogapi.Client) *EventsListService {
	return &EventsListService{dd: dd}
}

// Run fetches events in the given time range using the Event Explorer endpoint.
func (s *EventsListService) Run(ctx context.Context, input EventsListInput) (EventsListResult, error) {
	fromMs, err := timeutil.ParseToUnixMs(input.From)
	if err != nil {
		return EventsListResult{}, fail.NewValidation(
			fmt.Sprintf("invalid --from value %q: %s", input.From, err),
			`use relative (now-1h, now-30m, now-2d) or Unix milliseconds`,
		)
	}
	toMs, err := timeutil.ParseToUnixMs(input.To)
	if err != nil {
		return EventsListResult{}, fail.NewValidation(
			fmt.Sprintf("invalid --to value %q: %s", input.To, err),
			`use relative (now-1h, now-30m, now-2d) or Unix milliseconds`,
		)
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}

	listBody := map[string]any{
		"columns": []map[string]any{
			{"field": map[string]any{"path": "source"}},
			{"field": map[string]any{"path": "message"}},
			{"field": map[string]any{"path": "date"}},
		},
		"sorts":                []map[string]any{{"time": map[string]any{"order": "desc"}}},
		"limit":                limit,
		"time":                 map[string]any{"from": fromMs, "to": toMs},
		"includeEvents":        !input.CountOnly,
		"includeEventContents": !input.CountOnly,
		"computeCount":         true,
		"indexes":              []string{"*"},
		"executionInfo":        map[string]any{},
	}

	// Build search query from sources/tags filters.
	query := buildEventsSearchQuery(input.Sources, input.Tags)
	if query != "" {
		listBody["search"] = map[string]any{"query": query}
	}

	if input.Cursor != "" {
		listBody["after"] = input.Cursor
	}

	body := map[string]any{
		"list": listBody,
	}

	var raw v1FeedResponse
	if err := s.dd.Post(ctx, "/api/v1/logs-analytics/list?type=feed", body, &raw); err != nil {
		return EventsListResult{}, err
	}

	result := EventsListResult{
		HitCount: raw.HitCount,
	}

	if !input.CountOnly {
		for _, ev := range raw.Result.Events {
			result.Events = append(result.Events, mapFeedEvent(ev))
		}
		result.ReturnedCount = len(result.Events)
		result.NextCursor = raw.Result.Paging.After
	}

	if raw.HitCount == 0 && len(raw.Result.Events) > 0 {
		result.Warnings = append(result.Warnings,
			"hitCount is 0 but events were returned; these may be housekeeping/non-matching rows")
	}

	return result, nil
}

// buildEventsSearchQuery constructs a DataDog search query string from sources and tags filters.
func buildEventsSearchQuery(sources, tags string) string {
	var parts []string
	if sources != "" {
		for _, src := range strings.Split(sources, ",") {
			src = strings.TrimSpace(src)
			if src != "" {
				parts = append(parts, "source:"+src)
			}
		}
	}
	if tags != "" {
		for _, tag := range strings.Split(tags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				parts = append(parts, tag)
			}
		}
	}
	return strings.Join(parts, " ")
}

// mapFeedEvent converts a raw v1FeedEvent to the public Event type.
func mapFeedEvent(ev v1FeedEvent) Event {
	out := Event{
		ID: ev.ID,
	}

	if ev.Event == nil {
		// Fall back to columns only.
		out.Source = colString(ev.Columns, 0)
		out.Text = colString(ev.Columns, 1)
		return out
	}

	fb := ev.Event
	out.Source = fb.Source
	out.Text = fb.Message
	out.Host = fb.Host
	out.Tags = fb.Tags
	out.Timestamp = fb.DiscoveryTimestamp

	// Extract fields from custom map.
	if fb.Custom != nil {
		out.Title = stringFromMap(fb.Custom, "title")
		out.Status = stringFromMap(fb.Custom, "status")
		out.AggregationKey = stringFromMap(fb.Custom, "aggregation_key")
		out.Service = stringFromMap(fb.Custom, "service")

		if evtMap, ok := fb.Custom["evt"].(map[string]any); ok {
			out.EventType = stringFromAny(evtMap["type"])
			if out.Status == "" {
				out.AlertType = stringFromAny(evtMap["category"])
			}
		}

		if ts, ok := fb.Custom["timestamp"].(float64); ok && out.Timestamp == 0 {
			out.Timestamp = int64(ts)
		}

		if out.Host == "" {
			out.Host = stringFromMap(fb.Custom, "hostname")
		}
	}

	// If source not in body, use column.
	if out.Source == "" {
		out.Source = colString(ev.Columns, 0)
	}
	if out.Text == "" {
		out.Text = colString(ev.Columns, 1)
	}

	return out
}

func colString(cols []interface{}, i int) string {
	if i < len(cols) && cols[i] != nil {
		s, _ := cols[i].(string)
		return s
	}
	return ""
}

func stringFromMap(m map[string]any, key string) string {
	return stringFromAny(m[key])
}

func stringFromAny(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}
