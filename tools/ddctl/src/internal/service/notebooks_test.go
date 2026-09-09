package service

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestNotebooksValidate_EmptySeriesWarns(t *testing.T) {
	t.Parallel()

	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/api/v1/query") {
			return jsonResponse(http.StatusOK, `{"status":"ok","series":[]}`), nil
		}
		t.Fatalf("unexpected path %s", req.URL.Path)
		return nil, nil
	}))
	svc := NewNotebooksService(dd, NewMetricsQueryService(dd), "datadoghq.com")
	file := writeDashboardFile(t, `{
  "attributes": {
    "name": "Notebook A",
    "time": {"live_span":"1w"},
    "cells": [{
      "type": "notebook_cells",
      "attributes": {
        "definition": {
          "type": "timeseries",
          "requests": [{
            "queries": [{
              "data_source": "metrics",
              "name": "query1",
              "query": "avg:system.cpu.user{*}"
            }]
          }]
        }
      }
    }]
  }
}`)
	got, err := svc.Validate(context.Background(), NotebookValidateInput{
		FilePath: file,
		From:     "now-1h",
		To:       "now",
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(got.Warnings) == 0 {
		t.Fatal("expected no-data warning")
	}
}

func TestNotebooksCreate_ValidateBeforePost(t *testing.T) {
	t.Parallel()

	var posted bool
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPost && strings.Contains(req.URL.Path, "/api/v1/notebooks") {
			posted = true
			return jsonResponse(http.StatusOK, `{"data":{"id":"123","type":"notebooks","attributes":{"name":"Notebook A","cells":[{}]}}}`), nil
		}
		if strings.Contains(req.URL.Path, "/api/v1/query") {
			return jsonResponse(http.StatusBadRequest, `{"errors":["bad query"]}`), nil
		}
		return jsonResponse(http.StatusOK, `{}`), nil
	}))
	svc := NewNotebooksService(dd, NewMetricsQueryService(dd), "datadoghq.com")
	file := writeDashboardFile(t, `{
  "attributes": {
    "name": "Notebook A",
    "time": {"live_span":"1w"},
    "cells": [{
      "type": "notebook_cells",
      "attributes": {
        "definition": {
          "type": "timeseries",
          "requests": [{
            "queries": [{
              "data_source": "metrics",
              "name": "query1",
              "query": "avg:system.cpu.user{*}"
            }]
          }]
        }
      }
    }]
  }
}`)
	_, err := svc.Create(context.Background(), NotebookMutationInput{FilePath: file})
	if err == nil {
		t.Fatal("expected validate failure")
	}
	if posted {
		t.Fatal("POST should not run when validate fails")
	}
}

func TestNotebooksCreate_SkipValidatePosts(t *testing.T) {
	t.Parallel()

	var posted bool
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPost && strings.Contains(req.URL.Path, "/api/v1/notebooks") {
			posted = true
			return jsonResponse(http.StatusOK, `{"data":{"id":"123","type":"notebooks","attributes":{"name":"Notebook A","cells":[{}]}}}`), nil
		}
		t.Fatalf("unexpected %s %s", req.Method, req.URL.Path)
		return nil, nil
	}))
	svc := NewNotebooksService(dd, NewMetricsQueryService(dd), "datadoghq.com")
	file := writeDashboardFile(t, `{
  "attributes": {
    "name": "Notebook A",
    "time": {"live_span":"1w"},
    "cells": [{
      "type": "notebook_cells",
      "attributes": {
        "definition": {
          "type": "timeseries",
          "requests": [{
            "queries": [{
              "data_source": "metrics",
              "name": "query1",
              "query": "avg:system.cpu.user{*}"
            }]
          }]
        }
      }
    }]
  }
}`)
	_, err := svc.Create(context.Background(), NotebookMutationInput{
		FilePath:     file,
		SkipValidate: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !posted {
		t.Fatal("expected POST when skip-validate is set")
	}
}

func TestNotebooksUpdate_IfUnmodifiedSinceAborts(t *testing.T) {
	t.Parallel()

	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet && strings.Contains(req.URL.Path, "/api/v1/notebooks/123") {
			return jsonResponse(http.StatusOK, `{
  "data": {
    "id": "123",
    "type": "notebooks",
    "meta": {"modified_at": "2026-09-09T02:00:00Z"},
    "attributes": {"name": "Old", "time": {"live_span":"1w"}, "cells": [{
      "type": "notebook_cells",
      "attributes": {"definition": {"type": "note", "content": "x"}}
    }]}
  }
}`), nil
		}
		if req.Method == http.MethodPut {
			t.Fatal("PUT should not run when modified_at mismatches")
		}
		return jsonResponse(http.StatusOK, `{}`), nil
	}))
	svc := NewNotebooksService(dd, NewMetricsQueryService(dd), "datadoghq.com")
	file := writeDashboardFile(t, `{
  "attributes": {
    "name": "Notebook A",
    "time": {"live_span":"1w"},
    "cells": [{
      "type": "notebook_cells",
      "attributes": {"definition": {"type": "note", "content": "x"}}
    }]
  }
}`)
	_, err := svc.Update(context.Background(), NotebookMutationInput{
		ID:                "123",
		FilePath:          file,
		SkipValidate:      true,
		IfUnmodifiedSince: "2026-09-09T01:00:00Z",
	}, true)
	if err == nil || !strings.Contains(err.Error(), "modified_at") {
		t.Fatalf("error = %v", err)
	}
}

func TestNotebooksUpdate_DryRunNoPut(t *testing.T) {
	t.Parallel()

	var put bool
	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet && strings.Contains(req.URL.Path, "/api/v1/notebooks/123") {
			return jsonResponse(http.StatusOK, `{
  "data": {
    "id": "123",
    "type": "notebooks",
    "attributes": {"name": "Old", "time": {"live_span":"1w"}, "cells": [{
      "type": "notebook_cells",
      "attributes": {"definition": {"type": "note", "content": "x"}}
    }]}
  }
}`), nil
		}
		if req.Method == http.MethodPut {
			put = true
		}
		return jsonResponse(http.StatusOK, `{}`), nil
	}))
	svc := NewNotebooksService(dd, NewMetricsQueryService(dd), "datadoghq.com")
	file := writeDashboardFile(t, `{
  "attributes": {
    "name": "Notebook B",
    "time": {"live_span":"1w"},
    "cells": [{
      "type": "notebook_cells",
      "attributes": {"definition": {"type": "note", "content": "y"}}
    }]
  }
}`)
	got, err := svc.Update(context.Background(), NotebookMutationInput{
		ID:           "123",
		FilePath:     file,
		SkipValidate: true,
		DryRun:       true,
	}, true)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if put {
		t.Fatal("PUT should not run on dry-run")
	}
	if dry, _ := got["dry_run"].(bool); !dry {
		t.Fatalf("result = %#v", got)
	}
}
