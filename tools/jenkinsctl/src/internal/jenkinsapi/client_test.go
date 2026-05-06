package jenkinsapi

import (
	"github.com/alejandro-danos/jenkinsctl/internal/app"
	"github.com/alejandro-danos/jenkinsctl/internal/auth"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "user" || p != "pass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &app.Config{Username: "user", APIToken: "pass"}
	client := New(server.URL, auth.New(cfg))
	resp, err := client.Get("api/json")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
