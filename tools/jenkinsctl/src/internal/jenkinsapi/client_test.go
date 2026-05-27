package jenkinsapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGet_SetsBasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "user" || p != "pass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL, "user", "pass")
	resp, err := client.Get("api/json")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestNew_SetsCredentials(t *testing.T) {
	client := New("http://example.com", "myuser", "mytoken")
	if client.baseURL != "http://example.com" {
		t.Errorf("expected baseURL 'http://example.com', got %q", client.baseURL)
	}
	if client.username != "myuser" {
		t.Errorf("expected username 'myuser', got %q", client.username)
	}
	if client.token != "mytoken" {
		t.Errorf("expected token 'mytoken', got %q", client.token)
	}
}
