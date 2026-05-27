package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildStatusOutputStates(t *testing.T) {
	tests := []struct {
		name             string
		response         string
		wantState        string
		shouldContainURL bool
	}{
		{
			name:             "queued state",
			response:         `{"number":101,"result":"","url":"http://jenkins/job/test-job/101/","inQueue":true}`,
			wantState:        "queued",
			shouldContainURL: true,
		},
		{
			name:             "running state",
			response:         `{"number":102,"result":"","url":"http://jenkins/job/test-job/102/","building":true}`,
			wantState:        "running",
			shouldContainURL: true,
		},
		{
			name:             "blocked state",
			response:         `{"number":103,"result":"","url":"http://jenkins/job/test-job/103/","blocked":true}`,
			wantState:        "blocked",
			shouldContainURL: true,
		},
		{
			name:             "succeeded state",
			response:         `{"number":104,"result":"SUCCESS","url":"http://jenkins/job/test-job/104/"}`,
			wantState:        "succeeded",
			shouldContainURL: false,
		},
		{
			name:             "failed state",
			response:         `{"number":105,"result":"FAILURE","url":"http://jenkins/job/test-job/105/"}`,
			wantState:        "failed",
			shouldContainURL: true,
		},
		{
			name:             "aborted state",
			response:         `{"number":106,"result":"ABORTED","url":"http://jenkins/job/test-job/106/"}`,
			wantState:        "aborted",
			shouldContainURL: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tc.response))
			}))
			defer server.Close()

			url = server.URL
			user = "u"
			token = "t"

			var output bytes.Buffer
			buildStatusCmd.SetOut(&output)
			err := buildStatusCmd.RunE(buildStatusCmd, []string{"test-job"})
			if err != nil {
				t.Fatalf("expected success, got error: %v", err)
			}

			got := output.String()
			if !bytes.Contains(output.Bytes(), []byte("state="+tc.wantState)) {
				t.Fatalf("expected state %q, got %q", tc.wantState, got)
			}
			if tc.shouldContainURL && !bytes.Contains(output.Bytes(), []byte("url=http://jenkins/job/test-job/")) {
				t.Fatalf("expected URL in output for non-success state, got %q", got)
			}
			if !tc.shouldContainURL && bytes.Contains(output.Bytes(), []byte("url=")) {
				t.Fatalf("did not expect URL in succeeded output, got %q", got)
			}
		})
	}
}
