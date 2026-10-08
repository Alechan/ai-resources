package search

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

const (
	syntheticAt     = "synthetic-at-token-value"
	syntheticBlob36 = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	syntheticFreq   = `[[["SBNmJb","[null,null,null,\"nps\",null,\"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\"]",null,"1"]]]`
)

func TestRewriteQueryReplacesInnerSlotThree(t *testing.T) {
	// Given
	body := "f.req=" + url.QueryEscape(syntheticFreq) + "&at=" + url.QueryEscape(syntheticAt)

	// When
	got, err := RewriteQuery(body, "hello")

	// Then
	if err != nil {
		t.Fatal(err)
	}
	form, err := url.ParseQuery(got)
	if err != nil {
		t.Fatal(err)
	}
	if form.Get("at") != syntheticAt {
		t.Fatalf("at changed: %q", form.Get("at"))
	}
	raw := form.Get("f.req")
	if strings.Contains(raw, "nps") || !strings.Contains(raw, "hello") {
		t.Fatalf("f.req = %s", raw)
	}
	if !strings.Contains(raw, syntheticBlob36) {
		t.Fatal("non-query slot changed")
	}
	inner := innerPayload(t, raw)
	if got, ok := inner[3].(string); !ok || got != "hello" {
		t.Fatalf("query slot = %#v", inner[3])
	}
	if got, ok := inner[5].(string); !ok || got != syntheticBlob36 {
		t.Fatalf("blob slot = %#v", inner[5])
	}
}

func TestRewriteQueryRejectsEmpty(t *testing.T) {
	if _, err := RewriteQuery("f.req="+url.QueryEscape(syntheticFreq), "  "); err == nil {
		t.Fatal("expected empty query error")
	}
}

func TestRewriteQueryRequiresFreq(t *testing.T) {
	if _, err := RewriteQuery("at="+url.QueryEscape(syntheticAt), "hello"); err == nil || !strings.Contains(err.Error(), "f.req") {
		t.Fatalf("err = %v", err)
	}
}

func innerPayload(t *testing.T, raw string) []any {
	t.Helper()
	var root []any
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		t.Fatal(err)
	}
	call := root[0].([]any)[0].([]any)
	var inner []any
	if err := json.Unmarshal([]byte(call[1].(string)), &inner); err != nil {
		t.Fatal(err)
	}
	return inner
}
