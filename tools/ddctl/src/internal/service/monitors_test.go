package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const testMonitorJSON = `{
  "name": "DELETE ME ddctl-dev test",
  "type": "metric alert",
  "query": "avg(last_5m):avg:system.cpu.user{*} > 100",
  "message": "test",
  "tags": ["env:test"],
  "options": {
    "thresholds": {
      "critical": 100
    }
  }
}`

func testMonitorFile(t *testing.T, raw string) string {
	t.Helper()
	return writeDashboardFile(t, raw)
}

func TestMonitorsValidate_ExtractsMetricQueryForPreflight(t *testing.T) {
	t.Parallel()

	var gotQuery string
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(req.URL.Path, "/api/v1/query"):
			gotQuery = req.URL.Query().Get("query")
			return jsonResponse(http.StatusOK, `{"status":"ok","series":[{"metric":"x","pointlist":[[1,1]]}]}`), nil
		case req.URL.Path == "/api/v1/monitor/validate" && req.Method == http.MethodPost:
			return jsonResponse(http.StatusOK, `{}`), nil
		default:
			t.Fatalf("unexpected %s %s", req.Method, req.URL.Path)
			return nil, nil
		}
	}))
	svc := NewMonitorsService(dd, NewMetricsQueryService(dd), "datadoghq.com")
	file := testMonitorFile(t, testMonitorJSON)
	got, err := svc.Validate(context.Background(), MonitorValidateInput{FilePath: file, From: "now-1h", To: "now"})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if gotQuery != "avg:system.cpu.user{*}" {
		t.Fatalf("preflight query = %q", gotQuery)
	}
	if !got.QueryValid {
		t.Fatal("expected query valid")
	}
}

func TestMonitorsValidate_EmptySeriesWarns(t *testing.T) {
	t.Parallel()

	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/api/v1/query") {
			return jsonResponse(http.StatusOK, `{"status":"ok","series":[]}`), nil
		}
		if req.URL.Path == "/api/v1/monitor/validate" {
			return jsonResponse(http.StatusOK, `{}`), nil
		}
		t.Fatalf("unexpected %s %s", req.Method, req.URL.Path)
		return nil, nil
	}))
	svc := NewMonitorsService(dd, NewMetricsQueryService(dd), "datadoghq.com")
	file := testMonitorFile(t, testMonitorJSON)
	got, err := svc.Validate(context.Background(), MonitorValidateInput{FilePath: file, From: "now-1h", To: "now"})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !got.QueryNoData || len(got.Warnings) == 0 {
		t.Fatalf("QueryNoData=%v warnings=%v", got.QueryNoData, got.Warnings)
	}
}

func TestMonitorsValidate_RemoteValidateCalled(t *testing.T) {
	t.Parallel()

	called := false
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v1/monitor/validate" && req.Method == http.MethodPost {
			called = true
			return jsonResponse(http.StatusOK, `{}`), nil
		}
		if strings.Contains(req.URL.Path, "/api/v1/query") {
			return jsonResponse(http.StatusOK, `{"status":"ok","series":[{"metric":"x","pointlist":[[1,1]]}]}`), nil
		}
		t.Fatalf("unexpected %s %s", req.Method, req.URL.Path)
		return nil, nil
	}))
	svc := NewMonitorsService(dd, NewMetricsQueryService(dd), "datadoghq.com")
	file := testMonitorFile(t, testMonitorJSON)
	if _, err := svc.Validate(context.Background(), MonitorValidateInput{FilePath: file, From: "now-1h", To: "now"}); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !called {
		t.Fatal("remote validate not called")
	}
}

func TestMonitorsCreate_DryRunNoPost(t *testing.T) {
	t.Parallel()

	postCalled := false
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPost && req.URL.Path == "/api/v1/monitor" {
			postCalled = true
		}
		return jsonResponse(http.StatusOK, `{}`), nil
	}))
	svc := NewMonitorsService(dd, NewMetricsQueryService(dd), "datadoghq.com")
	file := testMonitorFile(t, testMonitorJSON)
	got, err := svc.Create(context.Background(), MonitorMutationInput{
		FilePath:     file,
		SkipValidate: true,
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if postCalled {
		t.Fatal("POST /api/v1/monitor should not run on dry-run")
	}
	if got["dry_run"] != true {
		t.Fatalf("dry_run = %v", got["dry_run"])
	}
}

func TestMonitorsCreate_Muted_IncludesSilencedInPost(t *testing.T) {
	t.Parallel()

	var createBody string
	var paths []string
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		paths = append(paths, req.Method+" "+req.URL.Path)
		if req.URL.Path == "/api/v1/monitor" && req.Method == http.MethodPost {
			b, _ := io.ReadAll(req.Body)
			createBody = string(b)
			return jsonResponse(http.StatusOK, `{
  "id":12345,
  "name":"DELETE ME ddctl-dev test",
  "options":{"silenced":{"*":null}}
}`), nil
		}
		if req.URL.Path == "/api/v1/monitor/12345" && req.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, `{
  "id":12345,
  "name":"DELETE ME ddctl-dev test",
  "options":{"silenced":{"*":null}}
}`), nil
		}
		return jsonResponse(http.StatusOK, `{}`), nil
	}))
	svc := NewMonitorsService(dd, NewMetricsQueryService(dd), "datadoghq.com")
	file := testMonitorFile(t, testMonitorJSON)
	got, err := svc.Create(context.Background(), MonitorMutationInput{
		FilePath:     file,
		SkipValidate: true,
		Muted:        true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !strings.Contains(createBody, `"*":null`) && !strings.Contains(createBody, `"*": null`) {
		t.Fatalf("create body = %s", createBody)
	}
	if containsPath(paths, "POST /api/v1/monitor/12345/mute") {
		t.Fatalf("unexpected mute endpoint: %v", paths)
	}
	if got["muted"] != true || got["mute_scope"] != "*" {
		t.Fatalf("got = %#v", got)
	}
}

func TestMonitorsCreate_Muted_VerificationFails(t *testing.T) {
	t.Parallel()

	var paths []string
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		paths = append(paths, req.Method+" "+req.URL.Path)
		if req.URL.Path == "/api/v1/monitor" && req.Method == http.MethodPost {
			return jsonResponse(http.StatusOK, `{"id":12345,"name":"DELETE ME ddctl-dev test"}`), nil
		}
		if req.URL.Path == "/api/v1/monitor/12345" && req.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, `{
  "id":12345,
  "name":"DELETE ME ddctl-dev test",
  "options":{"silenced":{}}
}`), nil
		}
		if req.URL.Path == "/api/v1/monitor/12345/mute" {
			return jsonResponse(http.StatusOK, `{}`), nil
		}
		return jsonResponse(http.StatusOK, `{}`), nil
	}))
	svc := NewMonitorsService(dd, NewMetricsQueryService(dd), "datadoghq.com")
	file := testMonitorFile(t, testMonitorJSON)
	_, err := svc.Create(context.Background(), MonitorMutationInput{
		FilePath:     file,
		SkipValidate: true,
		Muted:        true,
	})
	if err == nil {
		t.Fatal("expected verification error")
	}
	if !strings.Contains(err.Error(), "atomic muted monitor creation was not achieved") {
		t.Fatalf("error = %v", err)
	}
	if !containsPath(paths, "POST /api/v1/monitor/12345/mute") {
		t.Fatalf("paths = %v", paths)
	}
}

func TestMonitorsCreate_Muted_DryRun(t *testing.T) {
	t.Parallel()

	svc := NewMonitorsService(nil, nil, "datadoghq.com")
	file := testMonitorFile(t, testMonitorJSON)
	got, err := svc.Create(context.Background(), MonitorMutationInput{
		FilePath:     file,
		SkipValidate: true,
		DryRun:       true,
		Muted:        true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got["muted"] != true || got["mute_scope"] != "*" {
		t.Fatalf("got = %#v", got)
	}
}

func TestMonitorsUpdate_PreservesGlobalMute(t *testing.T) {
	t.Parallel()

	var putBody string
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v1/monitor/12345" && req.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, `{
  "id": 12345,
  "name": "old",
  "type": "metric alert",
  "query": "avg(last_5m):avg:system.cpu.user{*} > 100",
  "message": "old",
  "modified": "2026-09-09T01:00:00Z",
  "options": {"silenced": {"*": null}, "thresholds": {"critical": 100}}
}`), nil
		}
		if req.URL.Path == "/api/v1/monitor/12345" && req.Method == http.MethodPut {
			b, _ := io.ReadAll(req.Body)
			putBody = string(b)
			return jsonResponse(http.StatusOK, `{"id":12345}`), nil
		}
		return jsonResponse(http.StatusOK, `{}`), nil
	}))
	svc := NewMonitorsService(dd, NewMetricsQueryService(dd), "datadoghq.com")
	file := testMonitorFile(t, testMonitorJSON)
	if _, err := svc.Update(context.Background(), MonitorMutationInput{
		FilePath:     file,
		ID:           12345,
		SkipValidate: true,
	}, true); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !strings.Contains(putBody, `"*":null`) && !strings.Contains(putBody, `"*": null`) {
		t.Fatalf("put body = %s", putBody)
	}
}

func TestMonitorsUpdate_RequiresReplaceAll(t *testing.T) {
	t.Parallel()

	svc := NewMonitorsService(nil, nil, "datadoghq.com")
	file := testMonitorFile(t, testMonitorJSON)
	_, err := svc.Update(context.Background(), MonitorMutationInput{
		FilePath: file,
		ID:       12345,
	}, false)
	if err == nil || !strings.Contains(err.Error(), "replace-all") {
		t.Fatalf("error = %v", err)
	}
}

func TestMonitorsUpdate_DryRunNoPut(t *testing.T) {
	t.Parallel()

	putCalled := false
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPut {
			putCalled = true
		}
		if req.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, `{
  "id": 12345,
  "name": "old",
  "type": "metric alert",
  "query": "avg(last_5m):avg:system.cpu.user{*} > 100",
  "message": "old",
  "modified": "2026-09-09T01:00:00Z"
}`), nil
		}
		return jsonResponse(http.StatusOK, `{}`), nil
	}))
	svc := NewMonitorsService(dd, NewMetricsQueryService(dd), "datadoghq.com")
	file := testMonitorFile(t, testMonitorJSON)
	got, err := svc.Update(context.Background(), MonitorMutationInput{
		FilePath:     file,
		ID:           12345,
		SkipValidate: true,
		DryRun:       true,
	}, true)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if putCalled {
		t.Fatal("PUT should not run on dry-run")
	}
	if got["dry_run"] != true {
		t.Fatalf("dry_run = %v", got["dry_run"])
	}
}

func TestMonitorsUpdate_IfUnmodifiedSinceMismatch(t *testing.T) {
	t.Parallel()

	putCalled := false
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPut {
			putCalled = true
		}
		return jsonResponse(http.StatusOK, `{
  "id": 12345,
  "name": "old",
  "type": "metric alert",
  "query": "avg(last_5m):avg:system.cpu.user{*} > 100",
  "message": "old",
  "modified": "2026-09-09T01:00:00Z"
}`), nil
	}))
	svc := NewMonitorsService(dd, NewMetricsQueryService(dd), "datadoghq.com")
	file := testMonitorFile(t, testMonitorJSON)
	_, err := svc.Update(context.Background(), MonitorMutationInput{
		FilePath:          file,
		ID:                12345,
		SkipValidate:      true,
		IfUnmodifiedSince: "2020-01-01T00:00:00Z",
	}, true)
	if err == nil || !strings.Contains(err.Error(), "if-unmodified-since") {
		t.Fatalf("error = %v", err)
	}
	if putCalled {
		t.Fatal("PUT should not run on mismatch")
	}
}

func TestMonitorsMute_WithUntil(t *testing.T) {
	t.Parallel()

	var body string
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v1/monitor/12345" && req.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, `{"id":12345,"name":"t","overall_state":"OK"}`), nil
		}
		if req.URL.Path == "/api/v1/monitor/12345/mute" {
			b, _ := io.ReadAll(req.Body)
			body = string(b)
			return jsonResponse(http.StatusOK, `{}`), nil
		}
		t.Fatalf("unexpected %s %s", req.Method, req.URL.Path)
		return nil, nil
	}))
	svc := NewMonitorsService(dd, nil, "datadoghq.com")
	until := time.Date(2026, 9, 9, 17, 0, 0, 0, time.UTC).Format(time.RFC3339)
	if _, err := svc.Mute(context.Background(), MonitorMuteInput{ID: 12345, Until: until}); err != nil {
		t.Fatalf("Mute() error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("body = %q err = %v", body, err)
	}
	if payload["end"] == nil {
		t.Fatalf("body = %q", body)
	}
}

func TestMonitorsUnmute_ProductionRequiresConfirm(t *testing.T) {
	t.Parallel()

	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{
  "id": 12345,
  "name": "prod monitor",
  "tags": ["env:prod"]
}`), nil
	}))
	svc := NewMonitorsService(dd, nil, "datadoghq.com")
	_, err := svc.Unmute(context.Background(), MonitorMuteInput{ID: 12345})
	if err == nil || !strings.Contains(err.Error(), "confirm") {
		t.Fatalf("error = %v", err)
	}
}

func TestMonitorsUnmute_NonProductionOk(t *testing.T) {
	t.Parallel()

	unmuteCalled := false
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v1/monitor/12345/unmute" {
			unmuteCalled = true
			return jsonResponse(http.StatusOK, `{}`), nil
		}
		return jsonResponse(http.StatusOK, `{
  "id": 12345,
  "name": "test monitor",
  "tags": ["env:test"],
  "overall_state": "OK"
}`), nil
	}))
	svc := NewMonitorsService(dd, nil, "datadoghq.com")
	if _, err := svc.Unmute(context.Background(), MonitorMuteInput{ID: 12345}); err != nil {
		t.Fatalf("Unmute() error = %v", err)
	}
	if !unmuteCalled {
		t.Fatal("unmute not called")
	}
}

func TestMonitorsGet_PreservesOptions(t *testing.T) {
	t.Parallel()

	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{
  "id": 12345,
  "name": "DELETE ME ddctl-dev test",
  "type": "metric alert",
  "query": "avg(last_5m):avg:system.cpu.user{*} > 100",
  "options": {"thresholds": {"critical": 100}}
}`), nil
	}))
	svc := NewMonitorsService(dd, nil, "datadoghq.com")
	got, err := svc.Get(context.Background(), 12345)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if _, ok := got["options"].(map[string]any); !ok {
		t.Fatalf("options missing: %#v", got)
	}
}

func TestMonitorsDelete_RequiresConfirm(t *testing.T) {
	t.Parallel()

	svc := NewMonitorsService(nil, nil, "datadoghq.com")
	_, err := svc.Delete(context.Background(), MonitorDeleteInput{ID: 12345, Confirm: "99999"})
	if err == nil || !strings.Contains(err.Error(), "confirm") {
		t.Fatalf("error = %v", err)
	}
}

func TestMonitorsDelete_CallsDeleteEndpoint(t *testing.T) {
	t.Parallel()

	deleteCalled := false
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		switch req.Method {
		case http.MethodGet:
			return jsonResponse(http.StatusOK, `{
  "id": 12345,
  "name": "DELETE ME ddctl-dev test",
  "url": "https://app.datadoghq.com/monitors/12345"
}`), nil
		case http.MethodDelete:
			deleteCalled = true
			if req.URL.Path != "/api/v1/monitor/12345" {
				t.Fatalf("path = %s", req.URL.Path)
			}
			return jsonResponse(http.StatusOK, `{}`), nil
		default:
			t.Fatalf("method = %s", req.Method)
			return nil, nil
		}
	}))
	svc := NewMonitorsService(dd, nil, "datadoghq.com")
	got, err := svc.Delete(context.Background(), MonitorDeleteInput{ID: 12345, Confirm: "12345"})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleteCalled {
		t.Fatal("DELETE not called")
	}
	if got["deleted"] != true {
		t.Fatalf("deleted = %v", got["deleted"])
	}
}

func containsPath(paths []string, want string) bool {
	for _, p := range paths {
		if p == want {
			return true
		}
	}
	return false
}
