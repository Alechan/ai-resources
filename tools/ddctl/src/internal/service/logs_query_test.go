package service

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
)

//go:embed testdata/logs_beaver_dixa_response.json
var logsBeaverDixaResponseFixture []byte

//go:embed testdata/logs_housekeeping_hitcount_zero.json
var logsHousekeepingHitCountZeroFixture []byte

type logsRoundTripper func(*http.Request) (*http.Response, error)

func (f logsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type staticCookieProvider struct {
	cookies []*http.Cookie
}

func (s staticCookieProvider) Cookies() ([]*http.Cookie, error) { return s.cookies, nil }

func newLogsTestService(t *testing.T, body string) *LogsQueryService {
	t.Helper()
	httpClient := &http.Client{
		Transport: logsRoundTripper(func(req *http.Request) (*http.Response, error) {
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
	return NewLogsQueryService(dd)
}

func TestLogsQueryRun_PreservesCustomMap(t *testing.T) {
	t.Parallel()

	svc := newLogsTestService(t, string(logsBeaverDixaResponseFixture))
	got, err := svc.Run(context.Background(), LogsQueryInput{
		Query: "*", From: "now-1h", To: "now", Limit: 10,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(got.Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2", len(got.Data))
	}
	body, _ := got.Data[0].Custom["body"].(string)
	if !strings.Contains(body, "EmailExists") {
		t.Fatalf("custom.body = %q", body)
	}
	payload, ok := got.Data[1].Custom["payload"].(map[string]any)
	if !ok || payload["Email"] != "user@example.com" {
		t.Fatalf("custom.payload = %#v", got.Data[1].Custom["payload"])
	}
}

func TestLogsQueryRun_MessagePrefersMsgOverBody(t *testing.T) {
	t.Parallel()

	svc := newLogsTestService(t, string(logsBeaverDixaResponseFixture))
	got, err := svc.Run(context.Background(), LogsQueryInput{
		Query: "*", From: "now-1h", To: "now", Limit: 10,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.Data[0].Message != "dixa response payload" {
		t.Fatalf("Message = %q", got.Data[0].Message)
	}
}

func TestLogsQueryRun_FieldsProjection(t *testing.T) {
	t.Parallel()

	svc := newLogsTestService(t, string(logsBeaverDixaResponseFixture))
	got, err := svc.Run(context.Background(), LogsQueryInput{
		Query: "*", From: "now-1h", To: "now", Limit: 10,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	projected := ProjectLogEventFields(got.Data[0], ParseLogFieldList("body,status_code,url"))
	if len(projected) != 3 {
		t.Fatalf("projected = %#v", projected)
	}
}

func TestLogsQueryRun_IncludesHitCountAndWarningForHousekeepingRows(t *testing.T) {
	t.Parallel()

	svc := newLogsTestService(t, string(logsHousekeepingHitCountZeroFixture))
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

	svc := newLogsTestService(t, string(logsBeaverDixaResponseFixture))
	got, err := svc.Run(context.Background(), LogsQueryInput{
		Query: "*", From: "now-1h", To: "now", Limit: 10, CountOnly: true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.HitCount != 2 {
		t.Fatalf("HitCount = %d, want 2", got.HitCount)
	}
	if len(got.Data) != 0 {
		t.Fatalf("len(Data) = %d, want 0 for count-only mode", len(got.Data))
	}
}

func TestLogsExport_WritesNDJSON(t *testing.T) {
	t.Parallel()

	svc := newLogsTestService(t, string(logsBeaverDixaResponseFixture))
	dir := t.TempDir()
	path := filepath.Join(dir, "out.ndjson")
	result, err := svc.ExportLogsToFile(context.Background(), LogsExportInput{
		Query:  "*",
		From:   "now-1h",
		To:     "now",
		Limit:  1000,
		Format: "ndjson",
	}, path)
	if err != nil {
		t.Fatalf("ExportLogsToFile() error = %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
	if len(lines) != result.ReturnedCount {
		t.Fatalf("lines = %d, returned_count = %d", len(lines), result.ReturnedCount)
	}
}

func TestLogsGet_ReturnsSingleEvent(t *testing.T) {
	t.Parallel()

	svc := newLogsTestService(t, string(logsBeaverDixaResponseFixture))
	got, err := svc.Get(context.Background(), LogsGetInput{
		ID:   "evt-dixa-response",
		From: "now-1h",
		To:   "now",
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != "evt-dixa-response" {
		t.Fatalf("ID = %q", got.ID)
	}
}

func TestVerboseCustomLines_ShowsBodyMethodStatusCodeURL(t *testing.T) {
	t.Parallel()

	event := LogEvent{
		Message: "dixa response payload",
		Custom: map[string]any{
			"msg":         "dixa response payload",
			"body":        `{"message":"Create EndUser failed with [errors:EmailExists]"}`,
			"method":      "POST",
			"status_code": float64(400),
			"url":         "https://dev.dixa.io/v1/endusers",
		},
	}
	lines := VerboseCustomLines(event, nil)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"body=", "method=POST", "status_code=", "url="} {
		if !strings.Contains(joined, want) {
			t.Fatalf("lines = %q, missing %q", joined, want)
		}
	}
}

var _ auth.CookieProvider = staticCookieProvider{}
