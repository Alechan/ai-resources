package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alejandro-danos/jenkinsctl/internal/jenkinsapi"
)

func TestCheckConnectivity_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := jenkinsapi.New(server.URL, "u", "t")
	svc := NewDoctorService(client)

	if err := svc.CheckConnectivity(); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestCheckConnectivity_Fail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := jenkinsapi.New(server.URL+"/acceptance", "u", "t")
	svc := NewDoctorService(client)

	err := svc.CheckConnectivity()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	message := err.Error()
	if !strings.Contains(message, "kind=auth") {
		t.Fatalf("expected auth taxonomy, got: %s", message)
	}
	if !strings.Contains(message, "status=401") {
		t.Fatalf("expected status code details, got: %s", message)
	}
	if !strings.Contains(message, "auth_context=acceptance") {
		t.Fatalf("expected auth context label, got: %s", message)
	}
	if !strings.Contains(message, "hint=") {
		t.Fatalf("expected remediation hint, got: %s", message)
	}
}
