package service

import (
	"context"
	"fmt"
	"math"
	"net/url"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/timeutil"
)

// MetricsQueryInput holds parameters for a metrics query.
type MetricsQueryInput struct {
	Query string
	From  string
	To    string
}

// MetricsSeries represents a single time series result.
type MetricsSeries struct {
	Metric    string      `json:"metric"`
	Scope     string      `json:"scope"`
	Pointlist [][]float64 `json:"pointlist"` // [[timestamp_ms, value], ...]
	Start     float64     `json:"start"`
	End       float64     `json:"end"`
	Interval  int         `json:"interval"`
	// Derived stats — populated by the service, not the API.
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	Avg  float64 `json:"avg"`
	Last float64 `json:"last"`
}

// MetricsQueryResult is the response for a metrics query.
type MetricsQueryResult struct {
	Query  string          `json:"query"`
	Series []MetricsSeries `json:"series"`
}

type metricsAPIResponse struct {
	Status string          `json:"status"`
	Series []MetricsSeries `json:"series"`
	Error  string          `json:"error"`
}

// MetricsQueryService queries DataDog timeseries metrics.
type MetricsQueryService struct {
	dd *datadogapi.Client
}

// NewMetricsQueryService creates a MetricsQueryService.
func NewMetricsQueryService(dd *datadogapi.Client) *MetricsQueryService {
	return &MetricsQueryService{dd: dd}
}

// Run executes a metrics query and computes per-series summary stats.
func (s *MetricsQueryService) Run(ctx context.Context, input MetricsQueryInput) (MetricsQueryResult, error) {
	fromSec, err := timeutil.ParseToUnixSec(input.From)
	if err != nil {
		return MetricsQueryResult{}, fail.NewValidation(
			fmt.Sprintf("invalid --from value %q: %s", input.From, err),
			`use relative (now-1h, now-30m, now-2d) or Unix seconds`,
		)
	}
	toSec, err := timeutil.ParseToUnixSec(input.To)
	if err != nil {
		return MetricsQueryResult{}, fail.NewValidation(
			fmt.Sprintf("invalid --to value %q: %s", input.To, err),
			`use relative (now-1h, now-30m, now-2d) or Unix seconds`,
		)
	}

	path := fmt.Sprintf("/api/v1/query?query=%s&from=%d&to=%d",
		url.QueryEscape(input.Query), fromSec, toSec)

	var raw metricsAPIResponse
	if err := s.dd.Get(ctx, path, &raw); err != nil {
		return MetricsQueryResult{}, err
	}
	if raw.Error != "" {
		return MetricsQueryResult{}, fail.NewAPI(raw.Error, "check your metrics query syntax", "")
	}

	for i := range raw.Series {
		computeStats(&raw.Series[i])
	}

	return MetricsQueryResult{Query: input.Query, Series: raw.Series}, nil
}

func computeStats(s *MetricsSeries) {
	if len(s.Pointlist) == 0 {
		return
	}
	min := math.MaxFloat64
	max := -math.MaxFloat64
	sum := 0.0
	count := 0
	for _, pt := range s.Pointlist {
		if len(pt) < 2 {
			continue
		}
		v := pt[1]
		if math.IsNaN(v) {
			continue
		}
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
		count++
	}
	if count == 0 {
		return
	}
	s.Min = min
	s.Max = max
	s.Avg = sum / float64(count)
	// last non-NaN point
	for i := len(s.Pointlist) - 1; i >= 0; i-- {
		if len(s.Pointlist[i]) >= 2 && !math.IsNaN(s.Pointlist[i][1]) {
			s.Last = s.Pointlist[i][1]
			break
		}
	}
}
