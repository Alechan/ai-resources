package curl_test

import (
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/curl"
)

func TestExtractCookieHeader_MultiLine(t *testing.T) {
	curlCmd := `curl 'https://app.datadoghq.com/api/v1/logs-indexes' \
  -H 'Accept: application/json' \
  -H 'Cookie: DD_S=abc123; dd_csrf_token=xyz456' \
  -H 'X-DD-PROXY-DIRECT-TO-BACKEND: true'`

	got, err := curl.ExtractCookieHeader(curlCmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "DD_S=abc123; dd_csrf_token=xyz456"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtractCookieHeader_NoCookie(t *testing.T) {
	curlCmd := `curl 'https://app.datadoghq.com/api/v1/logs-indexes' \
  -H 'Accept: application/json' \
  -H 'Content-Type: application/json'`

	_, err := curl.ExtractCookieHeader(curlCmd)
	if err == nil {
		t.Fatal("expected error for cURL with no Cookie header, got nil")
	}
}

func TestExtractCookieHeader_DoubleQuoted(t *testing.T) {
	curlCmd := `curl "https://app.datadoghq.com/api/v1/logs" ` +
		`-H "Accept: application/json" ` +
		`-H "Cookie: session=tok1; other=val2" ` +
		`-H "X-Custom: foo"`

	got, err := curl.ExtractCookieHeader(curlCmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "session=tok1") {
		t.Errorf("got %q, expected to contain session=tok1", got)
	}
}

func TestExtractCookieHeader_LongHeader(t *testing.T) {
	curlCmd := `curl 'https://app.datadoghq.com/api/v1/validate' \
  --header 'Cookie: DD_S=abc; dd_csrf=xyz; _dd_s=rum=1&expire=1234567890'`

	got, err := curl.ExtractCookieHeader(curlCmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "DD_S=abc") {
		t.Errorf("got %q, expected to contain DD_S=abc", got)
	}
}
