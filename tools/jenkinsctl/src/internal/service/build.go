package service

import (
	"encoding/json"
	"fmt"
	"github.com/alejandro-danos/jenkinsctl/internal/jenkinsapi"
	"net/http"
)

type Build struct {
	Number int    `json:"number"`
	Result string `json:"result"`
	URL    string `json:"url"`
}

type BuildService struct {
	client *jenkinsapi.Client
}

func NewBuildService(client *jenkinsapi.Client) *BuildService {
	return &BuildService{client: client}
}

func (s *BuildService) GetLastBuildStatus(jobName string) (*Build, error) {
	path := fmt.Sprintf("job/%s/lastBuild/api/json", jobName)
	resp, err := s.client.Get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var build Build
	if err := json.NewDecoder(resp.Body).Decode(&build); err != nil {
		return nil, err
	}
	return &build, nil
}
