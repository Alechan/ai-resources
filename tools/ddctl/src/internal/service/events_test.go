package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
)

type eventsRoundTripper func(*http.Request) (*http.Response, error)

func (f eventsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newEventsClient(rt eventsRoundTripper) *datadogapi.Client {
	return datadogapi.NewClient(
		&http.Client{Transport: rt},
		"datadoghq.com",
		staticCookieProvider{
			cookies: []*http.Cookie{
				{Name: "dogweb", Value: "x"},
				{Name: "dd_csrf_token", Value: "csrf-tok"},
			},
		},
	)
}

func TestEventsListRun_ParsesEventsFromFeedEndpoint(t *testing.T) {
	t.Parallel()

	responseBody := `{
		"hitCount": 2,
		"result": {
			"events": [
				{
					"event_id": "AZ8DMcBg",
					"columns": ["containerd", "Task abc deleted with exit code 0", null],
					"id": "cursor-id-1",
					"event": {
						"source": "containerd",
						"message": "Task abc deleted with exit code 0",
						"host": "i-0b71680cf4d1f0409",
						"tags": ["service:tapir", "env:production"],
						"discovery_timestamp": 1782465020000,
						"custom": {
							"title": "Event on tasks from Containerd",
							"status": "info",
							"aggregation_key": "containerd:/tasks/delete",
							"service": "tapir",
							"hostname": "i-0b71680cf4d1f0409",
							"evt": {
								"type": "containerd",
								"category": "info"
							},
							"timestamp": 1782465020000
						}
					}
				},
				{
					"event_id": "AZ8DMbiQ",
					"columns": ["kubernetes", "Pod started", null],
					"id": "cursor-id-2",
					"event": {
						"source": "kubernetes",
						"message": "Pod started",
						"host": "i-0ce27f131c6df974a",
						"tags": ["service:albatross", "env:staging"],
						"discovery_timestamp": 1782465010000,
						"custom": {
							"title": "Pod lifecycle event",
							"status": "info",
							"service": "albatross",
							"evt": {
								"type": "kubernetes",
								"category": "info"
							}
						}
					}
				}
			],
			"paging": {
				"after": "next-page-cursor"
			}
		}
	}`

	rt := eventsRoundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(responseBody)),
		}, nil
	})

	svc := NewEventsListService(newEventsClient(rt))
	got, err := svc.Run(context.Background(), EventsListInput{
		From: "now-1h", To: "now", Limit: 50,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got.HitCount != 2 {
		t.Fatalf("HitCount = %d, want 2", got.HitCount)
	}
	if got.ReturnedCount != 2 {
		t.Fatalf("ReturnedCount = %d, want 2", got.ReturnedCount)
	}
	if got.NextCursor != "next-page-cursor" {
		t.Fatalf("NextCursor = %q, want %q", got.NextCursor, "next-page-cursor")
	}

	ev := got.Events[0]
	if ev.Source != "containerd" {
		t.Errorf("Events[0].Source = %q, want %q", ev.Source, "containerd")
	}
	if ev.Title != "Event on tasks from Containerd" {
		t.Errorf("Events[0].Title = %q, want %q", ev.Title, "Event on tasks from Containerd")
	}
	if ev.Host != "i-0b71680cf4d1f0409" {
		t.Errorf("Events[0].Host = %q, want %q", ev.Host, "i-0b71680cf4d1f0409")
	}
	if ev.Service != "tapir" {
		t.Errorf("Events[0].Service = %q, want %q", ev.Service, "tapir")
	}
	if ev.Status != "info" {
		t.Errorf("Events[0].Status = %q, want %q", ev.Status, "info")
	}
	if ev.Timestamp != 1782465020000 {
		t.Errorf("Events[0].Timestamp = %d, want %d", ev.Timestamp, int64(1782465020000))
	}
	if ev.Text != "Task abc deleted with exit code 0" {
		t.Errorf("Events[0].Text = %q, want %q", ev.Text, "Task abc deleted with exit code 0")
	}
	if ev.AggregationKey != "containerd:/tasks/delete" {
		t.Errorf("Events[0].AggregationKey = %q, want %q", ev.AggregationKey, "containerd:/tasks/delete")
	}

	ev2 := got.Events[1]
	if ev2.Source != "kubernetes" {
		t.Errorf("Events[1].Source = %q, want %q", ev2.Source, "kubernetes")
	}
	if ev2.Service != "albatross" {
		t.Errorf("Events[1].Service = %q, want %q", ev2.Service, "albatross")
	}
}

func TestEventsListRun_PostsToFeedEndpoint(t *testing.T) {
	t.Parallel()

	var capturedReq *http.Request
	var capturedBody map[string]any

	rt := eventsRoundTripper(func(req *http.Request) (*http.Response, error) {
		capturedReq = req
		bodyBytes, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(bodyBytes, &capturedBody)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"hitCount":0,"result":{"events":[],"paging":{"after":""}}}`)),
		}, nil
	})

	svc := NewEventsListService(newEventsClient(rt))
	_, err := svc.Run(context.Background(), EventsListInput{
		From: "now-1h", To: "now", Limit: 25,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify endpoint path.
	if capturedReq.URL.Path != "/api/v1/logs-analytics/list" {
		t.Errorf("URL.Path = %q, want %q", capturedReq.URL.Path, "/api/v1/logs-analytics/list")
	}
	if capturedReq.URL.Query().Get("type") != "feed" {
		t.Errorf("URL query type = %q, want %q", capturedReq.URL.Query().Get("type"), "feed")
	}
	if capturedReq.Method != http.MethodPost {
		t.Errorf("Method = %q, want %q", capturedReq.Method, http.MethodPost)
	}

	// Verify CSRF token injected.
	if capturedBody["_authentication_token"] != "csrf-tok" {
		t.Errorf("_authentication_token = %v, want %q", capturedBody["_authentication_token"], "csrf-tok")
	}

	// Verify limit is passed through.
	listMap, ok := capturedBody["list"].(map[string]any)
	if !ok {
		t.Fatalf("body.list is not a map")
	}
	if listMap["limit"] != float64(25) {
		t.Errorf("body.list.limit = %v, want 25", listMap["limit"])
	}
}

func TestEventsListRun_SourcesAndTagsBuildSearchQuery(t *testing.T) {
	t.Parallel()

	var capturedBody map[string]any

	rt := eventsRoundTripper(func(req *http.Request) (*http.Response, error) {
		bodyBytes, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(bodyBytes, &capturedBody)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"hitCount":0,"result":{"events":[],"paging":{"after":""}}}`)),
		}, nil
	})

	svc := NewEventsListService(newEventsClient(rt))
	_, err := svc.Run(context.Background(), EventsListInput{
		From:    "now-1h",
		To:      "now",
		Sources: "containerd,kubernetes",
		Tags:    "env:prod,service:tapir",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	listMap := capturedBody["list"].(map[string]any)
	searchMap, ok := listMap["search"].(map[string]any)
	if !ok {
		t.Fatalf("body.list.search is missing or not a map")
	}
	query, _ := searchMap["query"].(string)
	// Should contain all source and tag filters.
	for _, want := range []string{"source:containerd", "source:kubernetes", "env:prod", "service:tapir"} {
		if !strings.Contains(query, want) {
			t.Errorf("search query %q does not contain %q", query, want)
		}
	}
}

func TestEventsListRun_CountOnlyReturnsNoEvents(t *testing.T) {
	t.Parallel()

	rt := eventsRoundTripper(func(req *http.Request) (*http.Response, error) {
		body := `{"hitCount":150,"result":{"events":[{"event_id":"x","columns":["src","msg",null],"id":"id1","event":{"source":"src","message":"msg","tags":[],"custom":{}}}],"paging":{"after":""}}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})

	svc := NewEventsListService(newEventsClient(rt))
	got, err := svc.Run(context.Background(), EventsListInput{
		From: "now-1h", To: "now", CountOnly: true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.HitCount != 150 {
		t.Errorf("HitCount = %d, want 150", got.HitCount)
	}
	if len(got.Events) != 0 {
		t.Errorf("len(Events) = %d, want 0 in count-only mode", len(got.Events))
	}
}

func TestEventsListRun_WarnsOnHitCountZeroWithEvents(t *testing.T) {
	t.Parallel()

	rt := eventsRoundTripper(func(req *http.Request) (*http.Response, error) {
		body := `{"hitCount":0,"result":{"events":[{"event_id":"x","columns":["src","msg",null],"id":"id1","event":{"source":"src","message":"msg","tags":[],"custom":{}}}],"paging":{"after":""}}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})

	svc := NewEventsListService(newEventsClient(rt))
	got, err := svc.Run(context.Background(), EventsListInput{
		From: "now-1h", To: "now",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(got.Warnings) == 0 {
		t.Errorf("Warnings is empty, want housekeeping warning when hitCount=0 but events returned")
	}
}

func TestEventsListRun_Returns401AsAuthError(t *testing.T) {
	t.Parallel()

	rt := eventsRoundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"errors":["unauthorized"]}`)),
		}, nil
	})

	svc := NewEventsListService(newEventsClient(rt))
	_, err := svc.Run(context.Background(), EventsListInput{
		From: "now-1h", To: "now",
	})
	if err == nil {
		t.Fatal("Run() error = nil, want auth error on 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error = %q, want to contain '401'", err.Error())
	}
}

func TestEventsListRun_CursorPassedInRequest(t *testing.T) {
	t.Parallel()

	var capturedBody map[string]any

	rt := eventsRoundTripper(func(req *http.Request) (*http.Response, error) {
		bodyBytes, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(bodyBytes, &capturedBody)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"hitCount":0,"result":{"events":[],"paging":{"after":""}}}`)),
		}, nil
	})

	svc := NewEventsListService(newEventsClient(rt))
	_, err := svc.Run(context.Background(), EventsListInput{
		From: "now-1h", To: "now", Cursor: "my-cursor-value",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	listMap := capturedBody["list"].(map[string]any)
	if listMap["after"] != "my-cursor-value" {
		t.Errorf("body.list.after = %v, want %q", listMap["after"], "my-cursor-value")
	}
}

func TestEventsListRun_FallsBackToColumnsWhenEventBodyNil(t *testing.T) {
	t.Parallel()

	rt := eventsRoundTripper(func(req *http.Request) (*http.Response, error) {
		body := `{"hitCount":1,"result":{"events":[{"event_id":"ev1","columns":["docker","Container stopped",null],"id":"id1"}],"paging":{"after":""}}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})

	svc := NewEventsListService(newEventsClient(rt))
	got, err := svc.Run(context.Background(), EventsListInput{
		From: "now-1h", To: "now",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(got.Events) != 1 {
		t.Fatalf("len(Events) = %d, want 1", len(got.Events))
	}
	ev := got.Events[0]
	if ev.Source != "docker" {
		t.Errorf("Source = %q, want %q", ev.Source, "docker")
	}
	if ev.Text != "Container stopped" {
		t.Errorf("Text = %q, want %q", ev.Text, "Container stopped")
	}
}

func TestBuildEventsSearchQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		sources string
		tags    string
		want    string
	}{
		{name: "empty", sources: "", tags: "", want: ""},
		{name: "sources only", sources: "containerd,kubernetes", tags: "", want: "source:containerd source:kubernetes"},
		{name: "tags only", sources: "", tags: "env:prod,service:tapir", want: "env:prod service:tapir"},
		{name: "both", sources: "docker", tags: "env:staging", want: "source:docker env:staging"},
		{name: "whitespace trimmed", sources: " containerd , docker ", tags: " env:prod ", want: "source:containerd source:docker env:prod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildEventsSearchQuery(tt.sources, tt.tags)
			if got != tt.want {
				t.Errorf("buildEventsSearchQuery(%q, %q) = %q, want %q", tt.sources, tt.tags, got, tt.want)
			}
		})
	}
}
