package service

import (
	"context"
	"fmt"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
)

// Monitor represents a DataDog monitor.
type Monitor struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	OverallState string   `json:"overall_state"`
	Tags         []string `json:"tags"`
	Message      string   `json:"message"`
	Query        string   `json:"query"`
}

// MonitorsListResult is the response for listing monitors.
type MonitorsListResult struct {
	Monitors []Monitor `json:"monitors"`
}

// MonitorsGetResult is the response for fetching a single monitor.
type MonitorsGetResult struct {
	Monitor Monitor `json:"monitor"`
}

// MonitorsListService lists DataDog monitors.
type MonitorsListService struct {
	dd *datadogapi.Client
}

// NewMonitorsListService creates a MonitorsListService.
func NewMonitorsListService(dd *datadogapi.Client) *MonitorsListService {
	return &MonitorsListService{dd: dd}
}

// Run fetches all monitors, paginating with page_size=100.
func (s *MonitorsListService) Run(ctx context.Context) (MonitorsListResult, error) {
	var all []Monitor
	page := 0
	for {
		path := fmt.Sprintf("/api/v1/monitor?with_downtimes=false&page=%d&page_size=100", page)
		var batch []Monitor
		if err := s.dd.Get(ctx, path, &batch); err != nil {
			return MonitorsListResult{}, err
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			break
		}
		page++
	}
	return MonitorsListResult{Monitors: all}, nil
}

// MonitorsGetService fetches a single DataDog monitor by ID.
type MonitorsGetService struct {
	dd *datadogapi.Client
}

// NewMonitorsGetService creates a MonitorsGetService.
func NewMonitorsGetService(dd *datadogapi.Client) *MonitorsGetService {
	return &MonitorsGetService{dd: dd}
}

// Run fetches a monitor by ID.
func (s *MonitorsGetService) Run(ctx context.Context, id int64) (MonitorsGetResult, error) {
	path := fmt.Sprintf("/api/v1/monitor/%d", id)
	var m Monitor
	if err := s.dd.Get(ctx, path, &m); err != nil {
		return MonitorsGetResult{}, err
	}
	return MonitorsGetResult{Monitor: m}, nil
}
