package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
)

type logsRoundTripper func(*http.Request) (*http.Response, error)

func (f logsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type staticCookieProvider struct {
	cookies []*http.Cookie
}

func (s staticCookieProvider) Cookies() ([]*http.Cookie, error) { return s.cookies, nil }

func TestLogsQueryRun_IncludesHitCountAndWarningForHousekeepingRows(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: logsRoundTripper(func(req *http.Request) (*http.Response, error) {
			body := `{"hitCount":0,"result":{"events":[{"id":"1","columns":[null,"2026-05-26T12:03:57.505Z","i-host","albatross",null],"event":{"custom":{"log":{"retention":{"period":"7"}}}}}],"paging":{"after":""}}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}),
	}

	dd := datadogapi.NewClient(httpClient, "datadoghq.com", staticCookieProvider{
		cookies: []*http.Cookie{{Name: "dogweb", Value: "x"}, {Name: "dd_csrf_token", Value: "y"}},
	})

	svc := NewLogsQueryService(dd)
	got, err := svc.Run(context.Background(), LogsQueryInput{
		Query: "*", From: "now-1h", To: "now", Limit: 10,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got.HitCount != 0 {
		t.Fatalf("HitCount = %d, want 0", got.HitCount)
	}
	if len(got.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(got.Data))
	}
	if len(got.Warnings) == 0 {
		t.Fatalf("Warnings = %v, want at least one warning", got.Warnings)
	}
}

func TestLogsQueryRun_CountOnlyReturnsNoData(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: logsRoundTripper(func(req *http.Request) (*http.Response, error) {
			body := `{"hitCount":42,"result":{"events":[{"id":"1","columns":["info","2026-05-26T12:03:57.505Z","i-host","albatross","msg"]}],"paging":{"after":""}}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}),
	}

	dd := datadogapi.NewClient(httpClient, "datadoghq.com", staticCookieProvider{
		cookies: []*http.Cookie{{Name: "dogweb", Value: "x"}, {Name: "dd_csrf_token", Value: "y"}},
	})

	svc := NewLogsQueryService(dd)
	got, err := svc.Run(context.Background(), LogsQueryInput{
		Query: "*", From: "now-1h", To: "now", Limit: 10, CountOnly: true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.HitCount != 42 {
		t.Fatalf("HitCount = %d, want 42", got.HitCount)
	}
	if len(got.Data) != 0 {
		t.Fatalf("len(Data) = %d, want 0 for count-only mode", len(got.Data))
	}
}

var _ auth.CookieProvider = staticCookieProvider{}
