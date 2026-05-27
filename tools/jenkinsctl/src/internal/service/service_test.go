package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alejandro-danos/jenkinsctl/internal/jenkinsapi"
)

func TestListJobs(t *testing.T) {
	mockResponse := `{"jobs":[{"name":"test-job","url":"http://jenkins/job/test-job/","color":"blue"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := jenkinsapi.New(server.URL, "u", "t")
	svc := NewJobService(client)

	jobs, err := svc.ListJobs()
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(jobs) != 1 || jobs[0].Name != "test-job" {
		t.Errorf("unexpected job data: %+v", jobs)
	}
}

func TestGetLastBuildStatus(t *testing.T) {
	mockResponse := `{"number":123,"result":"SUCCESS","url":"http://jenkins/job/test-job/123/"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := jenkinsapi.New(server.URL, "u", "t")
	svc := NewBuildService(client)

	build, err := svc.GetLastBuildStatus("test-job")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if build.Number != 123 || build.Result != "SUCCESS" {
		t.Errorf("unexpected build data: %+v", build)
	}
}

func TestGetLastBuildStatus_NotFoundHasPathHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := jenkinsapi.New(server.URL, "u", "t")
	svc := NewBuildService(client)

	_, err := svc.GetLastBuildStatus("folder/pipeline")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	message := err.Error()
	if !strings.Contains(message, "kind=not_found") {
		t.Fatalf("expected not_found taxonomy, got: %s", message)
	}
	if !strings.Contains(message, "status=404") {
		t.Fatalf("expected status details, got: %s", message)
	}
	if !strings.Contains(message, "/job/") {
		t.Fatalf("expected /job/ path hint, got: %s", message)
	}
}

func TestGetLastBuildStatus_RedirectHasLocation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://login.example/sso")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	client := jenkinsapi.New(server.URL, "u", "t")
	svc := NewBuildService(client)

	_, err := svc.GetLastBuildStatus("test-job")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	message := err.Error()
	if !strings.Contains(message, "kind=redirect") {
		t.Fatalf("expected redirect taxonomy, got: %s", message)
	}
	if !strings.Contains(message, "location=https://login.example/sso") {
		t.Fatalf("expected redirect location, got: %s", message)
	}
}
