package googleapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDocsDocTabsSuccessFixture(t *testing.T) {
	payload := readFixture(t, "docs/doc_tabs_success_nested.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/documents/doc-1" {
			t.Fatalf("path = %q, want /v1/documents/doc-1", r.URL.Path)
		}
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := NewDocsClient(testHTTPClient(srv.URL))
	got, err := c.DocTabs(context.Background(), "tok", DocTabsRequest{ID: "doc-1"})
	if err != nil {
		t.Fatalf("DocTabs() error = %v", err)
	}
	if got["documentId"] != "doc-1" {
		t.Fatalf("documentId = %#v, want doc-1", got["documentId"])
	}
	tabs, ok := got["tabs"].([]any)
	if !ok || len(tabs) == 0 {
		t.Fatalf("tabs = %#v, want non-empty", got["tabs"])
	}
}

func TestDocsDocTabsStatusErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		fixture    string
		wantCat    string
	}{
		{name: "unauthorized auth", statusCode: http.StatusUnauthorized, fixture: "docs/error_401.json", wantCat: "auth"},
		{name: "forbidden scope", statusCode: http.StatusForbidden, fixture: "docs/error_403.json", wantCat: "scope"},
		{name: "server api", statusCode: http.StatusInternalServerError, fixture: "docs/error_500.json", wantCat: "api"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := readFixture(t, tt.fixture)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write(payload)
			}))
			defer srv.Close()

			c := NewDocsClient(testHTTPClient(srv.URL))
			_, err := c.DocTabs(context.Background(), "tok", DocTabsRequest{ID: "doc-1"})
			assertFailCategoryStrict(t, err, tt.wantCat)
		})
	}
}

func TestDocsDocTabsMalformedJSONMapsToAPI(t *testing.T) {
	payload := readFixture(t, "docs/malformed.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := NewDocsClient(testHTTPClient(srv.URL))
	_, err := c.DocTabs(context.Background(), "tok", DocTabsRequest{ID: "doc-1"})
	assertFailCategoryStrict(t, err, "api")
}

func TestDocsProbeStatusHandling(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantCat string
	}{
		{name: "not found accepted", status: http.StatusNotFound, wantCat: ""},
		{name: "ok accepted", status: http.StatusOK, wantCat: ""},
		{name: "unauthorized", status: http.StatusUnauthorized, wantCat: "auth"},
		{name: "forbidden", status: http.StatusForbidden, wantCat: "scope"},
		{name: "api", status: http.StatusInternalServerError, wantCat: "api"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(`{"error":"x"}`))
			}))
			defer srv.Close()

			c := NewDocsClient(testHTTPClient(srv.URL))
			err := c.Probe(context.Background(), "tok")
			if tt.wantCat == "" {
				if err != nil {
					t.Fatalf("Probe() error = %v, want nil", err)
				}
				return
			}
			assertFailCategoryStrict(t, err, tt.wantCat)
		})
	}
}

func TestDocsNetworkErrorMapsToNetwork(t *testing.T) {
	c := NewDocsClient(testHTTPClientWithErr(errors.New("dial tcp: no such host")))
	_, err := c.DocTabs(context.Background(), "tok", DocTabsRequest{ID: "doc-1"})
	assertFailCategoryStrict(t, err, "network")
}
