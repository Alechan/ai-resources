package service

import (
	"encoding/json"
	"fmt"
	"github.com/alejandro-danos/jenkinsctl/internal/jenkinsapi"
	"net/http"
)

type Job struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Color string `json:"color"`
}

type JobListResponse struct {
	Jobs []Job `json:"jobs"`
}

type JobService struct {
	client *jenkinsapi.Client
}

func NewJobService(client *jenkinsapi.Client) *JobService {
	return &JobService{client: client}
}

func (s *JobService) ListJobs() ([]Job, error) {
	resp, err := s.client.Get("api/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result JobListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Jobs, nil
}
