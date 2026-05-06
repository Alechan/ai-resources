package service

import (
	"github.com/alejandro-danos/jenkinsctl/internal/app"
	"github.com/alejandro-danos/jenkinsctl/internal/auth"
	"github.com/alejandro-danos/jenkinsctl/internal/jenkinsapi"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListJobs(t *testing.T) {
	mockResponse := `{"jobs":[{"name":"test-job","url":"http://jenkins/job/test-job/","color":"blue"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	cfg := &app.Config{Username: "u", APIToken: "t"}
	client := jenkinsapi.New(server.URL, auth.New(cfg))
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

	cfg := &app.Config{Username: "u", APIToken: "t"}
	client := jenkinsapi.New(server.URL, auth.New(cfg))
	svc := NewBuildService(client)

	build, err := svc.GetLastBuildStatus("test-job")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if build.Number != 123 || build.Result != "SUCCESS" {
		t.Errorf("unexpected build data: %+v", build)
	}
}
