package datadogapi

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type staticCookies struct {
	cookies []*http.Cookie
}

func (s staticCookies) Cookies() ([]*http.Cookie, error) {
	return s.cookies, nil
}

func testClient(t *testing.T, rt http.RoundTripper) *Client {
	t.Helper()
	return NewClient(&http.Client{Transport: rt}, "datadoghq.com", staticCookies{
		cookies: []*http.Cookie{{Name: "dogweb", Value: "secret"}, {Name: "dd_csrf_token", Value: "csrf"}},
	})
}

func TestClient_Delete(t *testing.T) {
	t.Parallel()
	var method, path string
	c := testClient(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		method = req.Method
		path = req.URL.Path
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"deleted_dashboard_id":"abc-def-ghi"}`)),
		}, nil
	}))
	var out map[string]any
	if err := c.Delete(context.Background(), "/api/v1/dashboard/abc-def-ghi", &out); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if method != http.MethodDelete || path != "/api/v1/dashboard/abc-def-ghi" {
		t.Fatalf("request %s %s", method, path)
	}
}

func TestClient_PostInjectsCSRF(t *testing.T) {
	t.Parallel()
	var body string
	c := testClient(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(req.Body)
		body = string(b)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	}))
	if err := c.Post(context.Background(), "/api/v1/dashboard", map[string]any{"title": "x"}, &map[string]any{}); err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	if !strings.Contains(body, `"_authentication_token":"csrf"`) {
		t.Fatalf("body = %s", body)
	}
}

func TestDebugLogger_DoesNotPrintCookieValues(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	c := testClient(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	}))
	c.SetDebugLogger(NewDebugLogger(&buf))
	if err := c.Get(context.Background(), "/api/v1/dashboard/abc", &map[string]any{}); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "/api/v1/dashboard/abc") {
		t.Fatalf("debug = %q", out)
	}
	if strings.Contains(out, "secret") || strings.Contains(out, "dogweb") {
		t.Fatalf("debug leaked cookie: %q", out)
	}
}
