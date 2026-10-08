package service

import (
	"strings"
	"testing"
)

func TestExtractMessage_PrefersMsgOverBody(t *testing.T) {
	t.Parallel()

	// Given
	custom := map[string]any{
		"msg":  "upstream response payload",
		"body": `{"message":"Create resource failed: already exists"}`,
	}

	// When
	got := extractMessage(custom)

	// Then
	if got != "upstream response payload" {
		t.Fatalf("extractMessage() = %q", got)
	}
}

func TestExtractMessage_UsesErrorWhenOnlyError(t *testing.T) {
	t.Parallel()

	got := extractMessage(map[string]any{"error": "upstream api error: Bad Request"})
	if got != "upstream api error: Bad Request" {
		t.Fatalf("extractMessage() = %q", got)
	}
}

func TestProjectLogEventFields_BodyStatusCodeURL(t *testing.T) {
	t.Parallel()

	event := LogEvent{
		ID:        "evt-example-response",
		Timestamp: "2026-09-10T13:20:02.820Z",
		Status:    "info",
		Service:   "example-api",
		Host:      "i-host",
		Message:   "upstream response payload",
		Custom: map[string]any{
			"msg":         "upstream response payload",
			"body":        `{"message":"Create resource failed: already exists"}`,
			"method":      "POST",
			"status_code": float64(400),
			"url":         "https://api.example.com/v1/resources",
		},
	}

	got := ProjectLogEventFields(event, ParseLogFieldList("body,status_code,url"))
	if len(got) != 3 {
		t.Fatalf("ProjectLogEventFields() = %#v", got)
	}
	if !strings.Contains(got["body"].(string), "already exists") {
		t.Fatalf("body = %q", got["body"])
	}
}

func TestProjectLogEventFields_MsgFromCustom(t *testing.T) {
	t.Parallel()

	event := LogEvent{
		Message: "upstream response payload",
		Custom: map[string]any{
			"msg": "upstream response payload",
		},
	}

	got := ProjectLogEventFields(event, ParseLogFieldList("msg"))
	if got["msg"] != "upstream response payload" {
		t.Fatalf("msg = %#v", got["msg"])
	}
	if _, ok := got["message"]; ok {
		t.Fatalf("unexpected message key: %#v", got)
	}
}

func TestParseLogFieldList_TrimsWhitespace(t *testing.T) {
	t.Parallel()

	got := ParseLogFieldList(" body , status_code , url ")
	if len(got) != 3 || got[0] != "body" || got[2] != "url" {
		t.Fatalf("ParseLogFieldList() = %#v", got)
	}
}

func TestParseLogFieldList_StarMeansFull(t *testing.T) {
	t.Parallel()

	if ParseLogFieldList("*") != nil {
		t.Fatalf("ParseLogFieldList('*') = %#v, want nil", ParseLogFieldList("*"))
	}
}
