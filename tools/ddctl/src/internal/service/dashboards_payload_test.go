package service

import (
	"strings"
	"testing"
)

func TestNormalizeDashboardPayload(t *testing.T) {
	t.Parallel()

	minimal := `{
  "title": "DELETE ME ddctl-dev",
  "layout_type": "ordered",
  "widgets": [{"definition":{"type":"note","content":"x"}}]
}`

	tests := []struct {
		name    string
		raw     string
		wantErr string
		wantTitle string
	}{
		{
			name:      "raw object",
			raw:       minimal,
			wantTitle: "DELETE ME ddctl-dev",
		},
		{
			name:      "wrapped dashboard key",
			raw:       `{"dashboard":` + minimal + `}`,
			wantTitle: "DELETE ME ddctl-dev",
		},
		{
			name:    "invalid JSON",
			raw:     `{`,
			wantErr: "invalid JSON dashboard file",
		},
		{
			name:    "array",
			raw:     `[]`,
			wantErr: "invalid JSON dashboard file",
		},
		{
			name:    "notebook envelope",
			raw:     `{"data":{"type":"notebooks","attributes":{"name":"n"}}}`,
			wantErr: "notebook envelope is not a dashboard",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeDashboardPayload([]byte(tc.raw))
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("NormalizeDashboardPayload() error = nil, want %q", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeDashboardPayload() error = %v", err)
			}
			if title, _ := got["title"].(string); title != tc.wantTitle {
				t.Fatalf("title = %q, want %q", title, tc.wantTitle)
			}
		})
	}
}

func TestPrepareDashboardCreatePayload(t *testing.T) {
	t.Parallel()

	base := map[string]any{
		"title":       "Old title",
		"layout_type": "ordered",
		"widgets": []any{
			map[string]any{"definition": map[string]any{"type": "note", "content": "x"}},
		},
		"id":            "cec-7ix-73w",
		"url":           "/dashboard/cec-7ix-73w/old",
		"author_handle": "someone@example.com",
		"author_name":   "Someone",
		"created_at":    "2026-09-09T00:00:00Z",
		"modified_at":   "2026-09-09T01:00:00Z",
		"description":   "keep me",
	}

	t.Run("happy", func(t *testing.T) {
		t.Parallel()
		got, err := PrepareDashboardCreatePayload(cloneMap(base), "")
		if err != nil {
			t.Fatalf("PrepareDashboardCreatePayload() error = %v", err)
		}
		if got["title"] != "Old title" {
			t.Fatalf("title = %v", got["title"])
		}
		if got["description"] != "keep me" {
			t.Fatalf("description stripped")
		}
	})

	t.Run("title override", func(t *testing.T) {
		t.Parallel()
		got, err := PrepareDashboardCreatePayload(cloneMap(base), "DELETE ME ddctl-dev copy")
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if got["title"] != "DELETE ME ddctl-dev copy" {
			t.Fatalf("title = %v", got["title"])
		}
	})

	t.Run("strips identity", func(t *testing.T) {
		t.Parallel()
		got, err := PrepareDashboardCreatePayload(cloneMap(base), "")
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		for _, k := range dashboardIdentityFields {
			if _, ok := got[k]; ok {
				t.Fatalf("identity field %q still present", k)
			}
		}
	})

	t.Run("missing title", func(t *testing.T) {
		t.Parallel()
		in := cloneMap(base)
		delete(in, "title")
		_, err := PrepareDashboardCreatePayload(in, "")
		if err == nil || !strings.Contains(err.Error(), "title") {
			t.Fatalf("error = %v, want title", err)
		}
	})

	t.Run("missing widgets", func(t *testing.T) {
		t.Parallel()
		in := cloneMap(base)
		delete(in, "widgets")
		_, err := PrepareDashboardCreatePayload(in, "")
		if err == nil || !strings.Contains(err.Error(), "widgets") {
			t.Fatalf("error = %v, want widgets", err)
		}
	})

	t.Run("empty widgets", func(t *testing.T) {
		t.Parallel()
		in := cloneMap(base)
		in["widgets"] = []any{}
		_, err := PrepareDashboardCreatePayload(in, "")
		if err == nil || !strings.Contains(err.Error(), "widgets") {
			t.Fatalf("error = %v, want widgets", err)
		}
	})

	t.Run("missing layout", func(t *testing.T) {
		t.Parallel()
		in := cloneMap(base)
		delete(in, "layout_type")
		_, err := PrepareDashboardCreatePayload(in, "")
		if err == nil || !strings.Contains(err.Error(), "layout_type") {
			t.Fatalf("error = %v, want layout_type", err)
		}
	})

	t.Run("bad layout", func(t *testing.T) {
		t.Parallel()
		in := cloneMap(base)
		in["layout_type"] = "grid"
		_, err := PrepareDashboardCreatePayload(in, "")
		if err == nil || !strings.Contains(err.Error(), "layout_type") {
			t.Fatalf("error = %v, want layout_type", err)
		}
	})
}

func TestPrepareDashboardUpdatePayload(t *testing.T) {
	t.Parallel()

	valid := map[string]any{
		"title":       "Board",
		"layout_type": "ordered",
		"widgets": []any{
			map[string]any{"definition": map[string]any{"type": "note", "content": "x"}},
		},
	}

	t.Run("missing replace-all", func(t *testing.T) {
		t.Parallel()
		_, err := PrepareDashboardUpdatePayload(cloneMap(valid), "cec-7ix-73w", false)
		if err == nil || !strings.Contains(err.Error(), "--replace-all is required") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("happy", func(t *testing.T) {
		t.Parallel()
		got, err := PrepareDashboardUpdatePayload(cloneMap(valid), "cec-7ix-73w", true)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if got["id"] != "cec-7ix-73w" {
			t.Fatalf("id = %v", got["id"])
		}
	})

	t.Run("missing widgets", func(t *testing.T) {
		t.Parallel()
		in := cloneMap(valid)
		delete(in, "widgets")
		_, err := PrepareDashboardUpdatePayload(in, "cec-7ix-73w", true)
		if err == nil || !strings.Contains(err.Error(), "widgets") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestExtractDashboardQueries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		raw          string
		wantMetrics  int
		wantLogs     int
		wantSkipped  int
		wantMetricQ  string
		wantLogQ     string
		wantErr      string
	}{
		{
			name: "timeseries requests q",
			raw: `{
  "title":"t","layout_type":"ordered",
  "widgets":[{"definition":{"type":"timeseries","requests":[{"q":"avg:system.cpu.user{*}"}]}}]
}`,
			wantMetrics: 1,
			wantMetricQ: "avg:system.cpu.user{*}",
		},
		{
			name: "timeseries queries metrics",
			raw: `{
  "title":"t","layout_type":"ordered",
  "widgets":[{"definition":{"type":"timeseries","requests":[{"queries":[{"data_source":"metrics","name":"a","query":"avg:system.cpu.user{*}"}]}]}}]
}`,
			wantMetrics: 1,
			wantMetricQ: "avg:system.cpu.user{*}",
		},
		{
			name: "group containing timeseries",
			raw: `{
  "title":"t","layout_type":"ordered",
  "widgets":[{"definition":{"type":"group","layout_type":"ordered","widgets":[
    {"definition":{"type":"timeseries","requests":[{"q":"avg:system.cpu.user{*}"}]}}
  ]}}]
}`,
			wantMetrics: 1,
			wantMetricQ: "avg:system.cpu.user{*}",
		},
		{
			name: "log_stream",
			raw: `{
  "title":"t","layout_type":"ordered",
  "widgets":[{"definition":{"type":"log_stream","query":"status:error"}}]
}`,
			wantLogs: 1,
			wantLogQ: "status:error",
		},
		{
			name: "logs data_source",
			raw: `{
  "title":"t","layout_type":"ordered",
  "widgets":[{"definition":{"type":"timeseries","requests":[{"queries":[{"data_source":"logs","name":"a","query":"service:foo"}]}]}}]
}`,
			wantLogs: 1,
			wantLogQ: "service:foo",
		},
		{
			name: "note widget",
			raw: `{
  "title":"t","layout_type":"ordered",
  "widgets":[{"definition":{"type":"note","content":"hello"}}]
}`,
		},
		{
			name: "slo widget skipped",
			raw: `{
  "title":"t","layout_type":"ordered",
  "widgets":[{"definition":{"type":"slo","slo_id":"abc"}}]
}`,
			wantSkipped: 1,
		},
		{
			name: "rum formula skipped data_source",
			raw: `{
  "title":"t","layout_type":"ordered",
  "widgets":[{"definition":{"type":"timeseries","requests":[{"queries":[{"data_source":"rum","name":"a","query":"rum query"}],"formulas":[{"formula":"a"}]}]}}]
}`,
			wantSkipped: 1,
		},
		{
			name: "empty metrics query errors",
			raw: `{
  "title":"t","layout_type":"ordered",
  "widgets":[{"definition":{"type":"timeseries","requests":[{"queries":[{"data_source":"metrics","name":"a","query":""}]}]}}]
}`,
			wantErr: "invalid metric query entry",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env, err := NormalizeDashboardPayload([]byte(tc.raw))
			if err != nil {
				t.Fatalf("NormalizeDashboardPayload() error = %v", err)
			}
			got, err := ExtractDashboardQueries(env)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ExtractDashboardQueries() error = %v", err)
			}
			if len(got.Metrics) != tc.wantMetrics {
				t.Fatalf("metrics = %#v, want %d", got.Metrics, tc.wantMetrics)
			}
			if len(got.Logs) != tc.wantLogs {
				t.Fatalf("logs = %#v, want %d", got.Logs, tc.wantLogs)
			}
			if len(got.Skipped) != tc.wantSkipped {
				t.Fatalf("skipped = %#v, want %d", got.Skipped, tc.wantSkipped)
			}
			if tc.wantMetricQ != "" && got.Metrics[0].Query != tc.wantMetricQ {
				t.Fatalf("metrics[0].Query = %q", got.Metrics[0].Query)
			}
			if tc.wantLogQ != "" && got.Logs[0].Query != tc.wantLogQ {
				t.Fatalf("logs[0].Query = %q", got.Logs[0].Query)
			}
		})
	}
}

func TestDiffDashboardPayloads_NoChanges(t *testing.T) {
	t.Parallel()
	env := map[string]any{"title": "A", "layout_type": "ordered"}
	got := DiffDashboardPayloads(env, cloneMap(env))
	if !strings.Contains(got, "no changes") {
		t.Fatalf("diff = %q", got)
	}
}

func TestDiffDashboardPayloads_TitleChange(t *testing.T) {
	t.Parallel()
	current := map[string]any{"title": "A"}
	next := map[string]any{"title": "B"}
	got := DiffDashboardPayloads(current, next)
	if !strings.Contains(got, "--- current") || !strings.Contains(got, "+++ next") {
		t.Fatalf("diff = %q", got)
	}
	if !strings.Contains(got, `"A"`) || !strings.Contains(got, `"B"`) {
		t.Fatalf("diff missing titles: %q", got)
	}
}
