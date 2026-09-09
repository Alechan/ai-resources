package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/output"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/service"
)

func TestRunDashboardsCmd_MissingSubcommand(t *testing.T) {
	t.Parallel()

	cfg := app.NewConfig("datadoghq.com", 10*time.Second, false, false)
	var stdout, stderr bytes.Buffer
	code := runDashboardsCmd(context.Background(), app.Services{}, cfg, nil, &stdout, &stderr)
	if code != fail.CodeValidation {
		t.Fatalf("code = %d, want %d; stderr=%s", code, fail.CodeValidation, stderr.String())
	}
	if !strings.Contains(stderr.String(), "get|list|search|validate|create|update|clone|delete") {
		t.Fatalf("stderr = %q, want usage listing subcommands", stderr.String())
	}
}

func TestRunDashboardsCmd_UnknownSubcommand(t *testing.T) {
	t.Parallel()

	cfg := app.NewConfig("datadoghq.com", 10*time.Second, false, false)
	var stdout, stderr bytes.Buffer
	code := runDashboardsCmd(context.Background(), app.Services{}, cfg, []string{"purge"}, &stdout, &stderr)
	if code != fail.CodeValidation {
		t.Fatalf("code = %d, want %d; stderr=%s", code, fail.CodeValidation, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown dashboards subcommand") {
		t.Fatalf("stderr = %q, want unknown dashboards subcommand", stderr.String())
	}
}

type cmdDashRT func(*http.Request) (*http.Response, error)

func (f cmdDashRT) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type cmdCookies struct {
	cookies []*http.Cookie
}

func (c cmdCookies) Cookies() ([]*http.Cookie, error) { return c.cookies, nil }

func testDashboardServices(t *testing.T, rt http.RoundTripper, jsonOut bool) (app.Services, app.Config) {
	t.Helper()
	dd := datadogapi.NewClient(&http.Client{Transport: rt}, "datadoghq.com", cmdCookies{
		cookies: []*http.Cookie{{Name: "dogweb", Value: "x"}, {Name: "dd_csrf_token", Value: "y"}},
	})
	metrics := service.NewMetricsQueryService(dd)
	logs := service.NewLogsQueryService(dd)
	return app.Services{
		Dashboards: service.NewDashboardsService(dd, metrics, logs, "datadoghq.com"),
		Output:     output.NewWriter(),
	}, app.NewConfig("datadoghq.com", 10*time.Second, jsonOut, false)
}

func TestRunDashboardsUpdate_RequiresReplaceAll(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "payload.json")
	err := os.WriteFile(file, []byte(`{
  "title": "DELETE ME ddctl-dev",
  "layout_type": "ordered",
  "widgets": [{"definition": {"type": "note", "content": "x"}}]
}`), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	svcs := app.Services{
		Dashboards: service.NewDashboardsService(nil, nil, nil, "datadoghq.com"),
	}
	cfg := app.NewConfig("datadoghq.com", 10*time.Second, false, false)
	var out, stderr bytes.Buffer
	code := runDashboardsUpdateCmd(
		context.Background(),
		svcs,
		cfg,
		[]string{"cec-7ix-73w", "--from-file", file},
		&out,
		&stderr,
	)
	if code != fail.CodeValidation {
		t.Fatalf("code = %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--replace-all is required") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestRunDashboardsGet_TextAndJSON(t *testing.T) {
	t.Parallel()
	body := `{
  "id": "cec-7ix-73w",
  "title": "DELETE ME ddctl-dev",
  "layout_type": "ordered",
  "widgets": [{"definition": {"type": "note", "content": "hello"}}]
}`
	rt := cmdDashRT(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})

	svcs, cfg := testDashboardServices(t, rt, false)
	var stdout, stderr bytes.Buffer
	code := runDashboardsGetCmd(context.Background(), svcs, cfg, []string{"cec-7ix-73w"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"ID:", "Title:", "URL:", "cec-7ix-73w", "DELETE ME ddctl-dev"} {
		if !strings.Contains(out, want) {
			t.Fatalf("text output missing %q:\n%s", want, out)
		}
	}

	svcsJSON, cfgJSON := testDashboardServices(t, rt, true)
	stdout.Reset()
	stderr.Reset()
	code = runDashboardsGetCmd(context.Background(), svcsJSON, cfgJSON, []string{"cec-7ix-73w"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("json code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"widgets"`) {
		t.Fatalf("json missing widgets: %s", stdout.String())
	}
}

func TestRunDashboardsValidate_Text(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "payload.json")
	err := os.WriteFile(file, []byte(`{
  "title": "DELETE ME ddctl-dev",
  "layout_type": "ordered",
  "widgets": [{"definition": {"type": "note", "content": "hello"}}]
}`), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	svcs, cfg := testDashboardServices(t, cmdDashRT(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected HTTP %s %s", req.Method, req.URL.Path)
		return nil, nil
	}), false)
	var stdout, stderr bytes.Buffer
	code := runDashboardsValidateCmd(context.Background(), svcs, cfg, []string{"--from-file", file}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "dashboard structure: valid") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}
