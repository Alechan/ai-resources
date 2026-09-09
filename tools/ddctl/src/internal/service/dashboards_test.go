package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
)

type dashRoundTripper func(*http.Request) (*http.Response, error)

func (f dashRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func testDashClient(rt http.RoundTripper) *datadogapi.Client {
	return datadogapi.NewClient(&http.Client{Transport: rt}, "datadoghq.com", staticCookieProvider{
		cookies: []*http.Cookie{{Name: "dogweb", Value: "x"}, {Name: "dd_csrf_token", Value: "y"}},
	})
}

func writeDashboardFile(t *testing.T, raw string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dashboard.json")
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

const noteDashboardJSON = `{
  "title": "DELETE ME ddctl-dev",
  "layout_type": "ordered",
  "description": "throwaway",
  "template_variables": [],
  "widgets": [{"definition": {"type": "note", "content": "hello"}}],
  "id": "cec-7ix-73w",
  "url": "/dashboard/cec-7ix-73w/old",
  "author_handle": "dev@example.com",
  "created_at": "2026-09-09T00:00:00Z",
  "modified_at": "2026-09-09T01:00:00.000000+00:00"
}`

const metricDashboardJSON = `{
  "title": "DELETE ME ddctl-dev",
  "layout_type": "ordered",
  "widgets": [{"definition": {"type": "timeseries", "requests": [{"q": "avg:system.cpu.user{*}"}]}}]
}`

const logDashboardJSON = `{
  "title": "DELETE ME ddctl-dev",
  "layout_type": "ordered",
  "widgets": [{"definition": {"type": "log_stream", "query": "status:info"}}]
}`

func TestDashboardsGet_ReturnsBodyAndURL(t *testing.T) {
	t.Parallel()

	var gotPath string
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		gotPath = req.URL.Path
		body := `{
  "id": "cec-7ix-73w",
  "title": "DELETE ME ddctl-dev",
  "layout_type": "ordered",
  "template_variables": [{"name": "env"}],
  "modified_at": "2026-09-09T01:00:00.000000+00:00",
  "widgets": [{"definition": {"type": "note", "content": "x"}}]
}`
		return jsonResponse(http.StatusOK, body), nil
	}))
	svc := NewDashboardsService(dd, nil, nil, "datadoghq.com")
	got, err := svc.Get(context.Background(), DashboardGetInput{ID: "cec-7ix-73w"})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if gotPath != "/api/v1/dashboard/cec-7ix-73w" {
		t.Fatalf("path = %q", gotPath)
	}
	if got["title"] != "DELETE ME ddctl-dev" {
		t.Fatalf("title = %v", got["title"])
	}
	if _, ok := got["widgets"]; !ok {
		t.Fatal("widgets missing")
	}
	if _, ok := got["template_variables"]; !ok {
		t.Fatal("template_variables missing")
	}
	wantURL := "https://app.datadoghq.com/dashboard/cec-7ix-73w"
	if got["url"] != wantURL {
		t.Fatalf("url = %v, want %s", got["url"], wantURL)
	}
}

func TestDashboardsGet_MissingID(t *testing.T) {
	t.Parallel()
	svc := NewDashboardsService(nil, nil, nil, "datadoghq.com")
	_, err := svc.Get(context.Background(), DashboardGetInput{})
	if err == nil || !strings.Contains(err.Error(), "missing dashboard ID") {
		t.Fatalf("error = %v", err)
	}
}

func TestDashboardsCreate_PostsStrippedPayload(t *testing.T) {
	t.Parallel()

	var method, path, rawBody string
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		method = req.Method
		path = req.URL.Path
		b, _ := io.ReadAll(req.Body)
		rawBody = string(b)
		return jsonResponse(http.StatusOK, `{"id":"abc-def-ghi","title":"DELETE ME ddctl-dev copy"}`), nil
	}))
	svc := NewDashboardsService(dd, nil, nil, "datadoghq.com")
	file := writeDashboardFile(t, noteDashboardJSON)
	got, err := svc.Create(context.Background(), DashboardMutationInput{
		FilePath:     file,
		Title:        "DELETE ME ddctl-dev copy",
		SkipValidate: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if method != http.MethodPost || path != "/api/v1/dashboard" {
		t.Fatalf("request %s %s", method, path)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(rawBody), &payload); err != nil {
		t.Fatalf("body json: %v", err)
	}
	if payload["title"] != "DELETE ME ddctl-dev copy" {
		t.Fatalf("posted title = %v", payload["title"])
	}
	for _, k := range []string{"id", "url", "author_handle", "created_at", "modified_at"} {
		if _, ok := payload[k]; ok {
			t.Fatalf("posted body still has %s", k)
		}
	}
	if got["id"] != "abc-def-ghi" {
		t.Fatalf("id = %v", got["id"])
	}
	if got["url"] != "https://app.datadoghq.com/dashboard/abc-def-ghi" {
		t.Fatalf("url = %v", got["url"])
	}
}

func TestDashboardsCreate_RejectsInvalidFile(t *testing.T) {
	t.Parallel()
	called := false
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		called = true
		return jsonResponse(http.StatusOK, `{}`), nil
	}))
	svc := NewDashboardsService(dd, nil, nil, "datadoghq.com")
	file := writeDashboardFile(t, `{"title":"x","layout_type":"ordered","widgets":[]}`)
	_, err := svc.Create(context.Background(), DashboardMutationInput{FilePath: file, SkipValidate: true})
	if err == nil || !strings.Contains(err.Error(), "widgets") {
		t.Fatalf("error = %v", err)
	}
	if called {
		t.Fatal("HTTP should not be called for invalid payload")
	}
}

func TestDashboardsUpdate_PutsWhenReplaceAll(t *testing.T) {
	t.Parallel()
	var method, path string
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		method = req.Method
		path = req.URL.Path
		return jsonResponse(http.StatusOK, `{"id":"cec-7ix-73w","title":"DELETE ME ddctl-dev"}`), nil
	}))
	svc := NewDashboardsService(dd, nil, nil, "datadoghq.com")
	file := writeDashboardFile(t, noteDashboardJSON)
	_, err := svc.Update(context.Background(), DashboardMutationInput{
		FilePath:     file,
		ID:           "cec-7ix-73w",
		SkipValidate: true,
	}, true)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if method != http.MethodPut || path != "/api/v1/dashboard/cec-7ix-73w" {
		t.Fatalf("request %s %s", method, path)
	}
}

func TestDashboardsUpdate_DryRunDoesNotPut(t *testing.T) {
	t.Parallel()
	methods := []string{}
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		methods = append(methods, req.Method+" "+req.URL.Path)
		if req.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, `{
  "id":"cec-7ix-73w",
  "title":"old",
  "layout_type":"ordered",
  "modified_at":"2026-09-09T01:00:00Z",
  "widgets":[{"definition":{"type":"note","content":"x"}}]
}`), nil
		}
		t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
		return nil, nil
	}))
	svc := NewDashboardsService(dd, nil, nil, "datadoghq.com")
	file := writeDashboardFile(t, `{
  "title": "new",
  "layout_type": "ordered",
  "widgets": [{"definition": {"type": "note", "content": "hello"}}]
}`)
	got, err := svc.Update(context.Background(), DashboardMutationInput{
		FilePath:     file,
		ID:           "cec-7ix-73w",
		DryRun:       true,
		SkipValidate: true,
	}, true)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got["dry_run"] != true {
		t.Fatalf("dry_run = %v", got["dry_run"])
	}
	diff, _ := got["diff"].(string)
	if !strings.Contains(diff, "old") || !strings.Contains(diff, "new") {
		t.Fatalf("diff = %q", diff)
	}
	if len(methods) != 1 || !strings.HasPrefix(methods[0], "GET ") {
		t.Fatalf("methods = %v", methods)
	}
}

func TestDashboardsUpdate_ExpectedModifiedAtMismatch(t *testing.T) {
	t.Parallel()
	putCalled := false
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPut {
			putCalled = true
		}
		return jsonResponse(http.StatusOK, `{
  "id":"cec-7ix-73w",
  "title":"DELETE ME ddctl-dev",
  "modified_at":"2026-09-09T01:00:00Z",
  "layout_type":"ordered",
  "widgets":[{"definition":{"type":"note","content":"x"}}]
}`), nil
	}))
	svc := NewDashboardsService(dd, nil, nil, "datadoghq.com")
	file := writeDashboardFile(t, noteDashboardJSON)
	_, err := svc.Update(context.Background(), DashboardMutationInput{
		FilePath:           file,
		ID:                 "cec-7ix-73w",
		ExpectedModifiedAt: "2020-01-01T00:00:00Z",
		SkipValidate:       true,
	}, true)
	if err == nil || !strings.Contains(err.Error(), "modified_at") {
		t.Fatalf("error = %v", err)
	}
	if putCalled {
		t.Fatal("PUT should not run on mismatch")
	}
}

func TestDashboardsUpdate_ExpectedModifiedAtMatch(t *testing.T) {
	t.Parallel()
	putCalled := false
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPut {
			putCalled = true
			return jsonResponse(http.StatusOK, `{"id":"cec-7ix-73w","title":"DELETE ME ddctl-dev"}`), nil
		}
		return jsonResponse(http.StatusOK, `{
  "id":"cec-7ix-73w",
  "title":"DELETE ME ddctl-dev",
  "modified_at":"2026-09-09T01:00:00Z",
  "layout_type":"ordered",
  "widgets":[{"definition":{"type":"note","content":"x"}}]
}`), nil
	}))
	svc := NewDashboardsService(dd, nil, nil, "datadoghq.com")
	file := writeDashboardFile(t, noteDashboardJSON)
	_, err := svc.Update(context.Background(), DashboardMutationInput{
		FilePath:           file,
		ID:                 "cec-7ix-73w",
		ExpectedModifiedAt: "2026-09-09T01:00:00Z",
		SkipValidate:       true,
	}, true)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !putCalled {
		t.Fatal("PUT not called")
	}
}

func TestDashboardsValidate_RejectsMissingTitle(t *testing.T) {
	t.Parallel()
	svc := NewDashboardsService(nil, nil, nil, "datadoghq.com")
	file := writeDashboardFile(t, `{"layout_type":"ordered","widgets":[{"definition":{"type":"note","content":"x"}}]}`)
	_, err := svc.Validate(context.Background(), DashboardValidateInput{FilePath: file})
	if err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("error = %v", err)
	}
}

func TestDashboardsValidate_MetricEmptySeriesWarns(t *testing.T) {
	t.Parallel()
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/api/v1/query") {
			return jsonResponse(http.StatusOK, `{"status":"ok","series":[]}`), nil
		}
		t.Fatalf("unexpected path %s", req.URL.Path)
		return nil, nil
	}))
	metrics := NewMetricsQueryService(dd)
	svc := NewDashboardsService(dd, metrics, NewLogsQueryService(dd), "datadoghq.com")
	file := writeDashboardFile(t, metricDashboardJSON)
	got, err := svc.Validate(context.Background(), DashboardValidateInput{FilePath: file, From: "now-1h", To: "now"})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got.MetricsNoData != 1 {
		t.Fatalf("MetricsNoData = %d warnings=%v", got.MetricsNoData, got.Warnings)
	}
	if len(got.Warnings) == 0 {
		t.Fatal("expected no-data warning")
	}
}

func TestDashboardsValidate_AllowEmptySeriesWarns(t *testing.T) {
	t.Parallel()
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"status":"ok","series":[]}`), nil
	}))
	svc := NewDashboardsService(dd, NewMetricsQueryService(dd), NewLogsQueryService(dd), "datadoghq.com")
	file := writeDashboardFile(t, metricDashboardJSON)
	got, err := svc.Validate(context.Background(), DashboardValidateInput{
		FilePath:         file,
		From:             "now-1h",
		To:               "now",
		AllowEmptySeries: true,
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(got.Warnings) == 0 {
		t.Fatal("expected warnings")
	}
}

func TestDashboardsValidate_LogQueryCountOnly(t *testing.T) {
	t.Parallel()
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v1/logs-analytics/list" {
			t.Fatalf("path = %s", req.URL.Path)
		}
		return jsonResponse(http.StatusOK, `{"hitCount":0,"result":{"events":[],"paging":{"after":""}}}`), nil
	}))
	svc := NewDashboardsService(dd, NewMetricsQueryService(dd), NewLogsQueryService(dd), "datadoghq.com")
	file := writeDashboardFile(t, logDashboardJSON)
	got, err := svc.Validate(context.Background(), DashboardValidateInput{FilePath: file, From: "now-1h", To: "now"})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got.LogsNoData != 1 {
		t.Fatalf("LogsNoData = %d warnings=%v", got.LogsNoData, got.Warnings)
	}
}

func TestDashboardsValidate_SubstitutesTemplateVariables(t *testing.T) {
	t.Parallel()
	var gotQuery string
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		gotQuery = req.URL.Query().Get("query")
		return jsonResponse(http.StatusOK, `{"status":"ok","series":[{"metric":"x","pointlist":[[1,1]]}]}`), nil
	}))
	svc := NewDashboardsService(dd, NewMetricsQueryService(dd), NewLogsQueryService(dd), "datadoghq.com")
	file := writeDashboardFile(t, `{
  "title": "t",
  "layout_type": "ordered",
  "widgets": [{"definition": {"type": "timeseries", "title": "API", "requests": [{"q": "avg:system.cpu.user{kube_namespace:$environment.value}"}]}}]
}`)
	got, err := svc.Validate(context.Background(), DashboardValidateInput{
		FilePath:          file,
		From:              "now-1h",
		To:                "now",
		TemplateVariables: map[string]string{"environment": "acceptance"},
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !strings.Contains(gotQuery, "kube_namespace:acceptance") {
		t.Fatalf("preflight query = %q", gotQuery)
	}
	if got.MetricsValid != 1 {
		t.Fatalf("MetricsValid = %d", got.MetricsValid)
	}
}

func TestDashboardsCreate_DryRunDoesNotPost(t *testing.T) {
	t.Parallel()
	called := false
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		called = true
		return jsonResponse(http.StatusOK, `{}`), nil
	}))
	svc := NewDashboardsService(dd, nil, nil, "datadoghq.com")
	file := writeDashboardFile(t, noteDashboardJSON)
	got, err := svc.Create(context.Background(), DashboardMutationInput{
		FilePath:     file,
		Title:        "DELETE ME ddctl-dev copy",
		SkipValidate: true,
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if called {
		t.Fatal("HTTP should not be called for create --dry-run")
	}
	if got["dry_run"] != true {
		t.Fatalf("dry_run = %v", got["dry_run"])
	}
	if got["title"] != "DELETE ME ddctl-dev copy" {
		t.Fatalf("title = %v", got["title"])
	}
}

func TestDashboardsValidate_SkipsUnknownDataSource(t *testing.T) {
	t.Parallel()
	httpCalled := false
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		httpCalled = true
		return jsonResponse(http.StatusOK, `{"status":"ok","series":[]}`), nil
	}))
	svc := NewDashboardsService(dd, NewMetricsQueryService(dd), NewLogsQueryService(dd), "datadoghq.com")
	file := writeDashboardFile(t, `{
  "title": "t",
  "layout_type": "ordered",
  "widgets": [{"definition": {"type": "slo", "slo_id": "abc"}}]
}`)
	got, err := svc.Validate(context.Background(), DashboardValidateInput{FilePath: file, From: "now-1h", To: "now"})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(got.Skipped) == 0 {
		t.Fatal("expected skipped")
	}
	if len(got.Warnings) == 0 {
		t.Fatal("expected warnings")
	}
	if httpCalled {
		t.Fatal("should not preflight skipped widgets")
	}
}
