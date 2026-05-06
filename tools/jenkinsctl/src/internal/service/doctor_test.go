package service

import (
	"github.com/alejandro-danos/jenkinsctl/internal/app"
	"github.com/alejandro-danos/jenkinsctl/internal/auth"
	"github.com/alejandro-danos/jenkinsctl/internal/jenkinsapi"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckConnectivity_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &app.Config{Username: "u", APIToken: "t"}
	client := jenkinsapi.New(server.URL, auth.New(cfg))
	svc := NewDoctorService(client)

	if err := svc.CheckConnectivity(); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestCheckConnectivity_Fail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	cfg := &app.Config{Username: "u", APIToken: "t"}
	client := jenkinsapi.New(server.URL, auth.New(cfg))
	svc := NewDoctorService(client)

	if err := svc.CheckConnectivity(); err == nil {
		t.Fatal("expected error, got nil")
	}
}
