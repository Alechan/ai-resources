package service

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteLogsCSV_MatchesManualExportShape(t *testing.T) {
	t.Parallel()

	events := []LogEvent{{
		Timestamp: "2026-09-10T13:20:02.820Z",
		Host:      "i-host",
		Service:   "beaver-api",
		Custom: map[string]any{
			"msg":  "dixa response payload",
			"body": `{"message":"EmailExists"}`,
		},
	}}

	var buf bytes.Buffer
	if err := WriteLogsCSV(events, nil, &buf); err != nil {
		t.Fatalf("WriteLogsCSV() error = %v", err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "Date,Host,Service,Content\n") {
		t.Fatalf("header = %q", out)
	}
	if !strings.Contains(out, "EmailExists") {
		t.Fatalf("content missing body: %q", out)
	}
}

func TestLogEventToMap_IncludesCustomAndTags(t *testing.T) {
	t.Parallel()

	got := LogEventToMap(LogEvent{
		ID:      "evt-1",
		Message: "hello",
		Custom:  map[string]any{"email": "a@b.com"},
		Tags:    []string{"kube_namespace:acceptance"},
	})
	if got["custom"] == nil || got["tags"] == nil {
		t.Fatalf("LogEventToMap() = %#v", got)
	}
}
