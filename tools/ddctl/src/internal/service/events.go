package service

import (
	"context"
	"fmt"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/timeutil"
)

// EventsListInput holds parameters for an events list request.
type EventsListInput struct {
	From    string
	To      string
	Sources string // comma-separated sources filter
	Tags    string // comma-separated tags filter
}

// Event represents a DataDog event.
type Event struct {
	ID        int64    `json:"id"`
	Title     string   `json:"title"`
	Text      string   `json:"text"`
	DateHappened int64 `json:"date_happened"`
	Priority  string   `json:"priority"`
	AlertType string   `json:"alert_type"`
	Host      string   `json:"host"`
	Tags      []string `json:"tags"`
	URL       string   `json:"url"`
}

// EventsListResult is the response for listing events.
type EventsListResult struct {
	Events []Event `json:"events"`
}

type eventsAPIResponse struct {
	Events []Event `json:"events"`
}

// EventsListService lists DataDog events.
// Note: uses /api/v1/events which may require different auth in some DataDog configurations.
// If you receive HTTP 401, the browser may use a different internal endpoint for this resource.
type EventsListService struct {
	dd *datadogapi.Client
}

// NewEventsListService creates an EventsListService.
func NewEventsListService(dd *datadogapi.Client) *EventsListService {
	return &EventsListService{dd: dd}
}

// Run fetches events in the given time range.
func (s *EventsListService) Run(ctx context.Context, input EventsListInput) (EventsListResult, error) {
	fromSec, err := timeutil.ParseToUnixSec(input.From)
	if err != nil {
		return EventsListResult{}, fail.NewValidation(
			fmt.Sprintf("invalid --from value %q: %s", input.From, err),
			`use relative (now-1h, now-30m, now-2d) or Unix seconds`,
		)
	}
	toSec, err := timeutil.ParseToUnixSec(input.To)
	if err != nil {
		return EventsListResult{}, fail.NewValidation(
			fmt.Sprintf("invalid --to value %q: %s", input.To, err),
			`use relative (now-1h, now-30m, now-2d) or Unix seconds`,
		)
	}

	path := fmt.Sprintf("/api/v1/events?start=%d&end=%d", fromSec, toSec)
	if input.Sources != "" {
		path += "&sources=" + input.Sources
	}
	if input.Tags != "" {
		path += "&tags=" + input.Tags
	}

	var raw eventsAPIResponse
	if err := s.dd.Get(ctx, path, &raw); err != nil {
		return EventsListResult{}, err
	}
	return EventsListResult{Events: raw.Events}, nil
}
