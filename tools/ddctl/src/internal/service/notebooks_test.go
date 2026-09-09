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
	svc := NewNotebooksService(dd, NewMetricsQueryService(dd))
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

func TestNotebooksValidate_EmptySeriesWarnsWithAllowEmptyFlag(t *testing.T) {
	t.Parallel()

	dd := testDashClient(dashRoundTripper(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"status":"ok","series":[]}`), nil
	}))
	svc := NewNotebooksService(dd, NewMetricsQueryService(dd))
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
		FilePath:         file,
		From:             "now-1h",
		To:               "now",
		AllowEmptySeries: true,
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(got.Warnings) == 0 {
		t.Fatal("expected no-data warning")
	}
}
