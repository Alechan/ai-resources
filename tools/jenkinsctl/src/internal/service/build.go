package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/alejandro-danos/jenkinsctl/internal/jenkinsapi"
)

type Build struct {
	Number   int    `json:"number"`
	Result   string `json:"result"`
	URL      string `json:"url"`
	Building bool   `json:"building"`
	InQueue  bool   `json:"inQueue"`
	Blocked  bool   `json:"blocked"`
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
		return nil, classifyHTTPError(resp, s.client.BuildURL(path))
	}

	var build Build
	if err := json.NewDecoder(resp.Body).Decode(&build); err != nil {
		return nil, err
	}
	return &build, nil
}

// --- status-by-ref support ---

type buildListResponse struct {
	Builds []buildRef `json:"builds"`
}

type buildRef struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

type buildDetails struct {
	Number  int           `json:"number"`
	Result  string        `json:"result"`
	URL     string        `json:"url"`
	Actions []buildAction `json:"actions"`
}

type buildAction struct {
	Parameters        []buildParam `json:"parameters,omitempty"`
	LastBuiltRevision *gitRevision `json:"lastBuiltRevision,omitempty"`
}

type buildParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type gitRevision struct {
	Branch []gitBranch `json:"branch"`
}

type gitBranch struct {
	Name string `json:"name"`
	SHA1 string `json:"SHA1"`
}

const maxBuildsToSearch = 20

// GetBuildByRef searches recent builds for one matching the given git ref.
// Returns nil if no matching build is found.
func (s *BuildService) GetBuildByRef(jobName, ref string) (*Build, error) {
	builds, err := s.listRecentBuilds(jobName, maxBuildsToSearch)
	if err != nil {
		return nil, err
	}

	for _, b := range builds {
		details, err := s.getBuildDetails(jobName, b.Number)
		if err != nil {
			continue
		}

		if matchesRef(details, ref) {
			result := details.Result
			if result == "" {
				result = "IN PROGRESS"
			}

			return &Build{
				Number: details.Number,
				Result: result,
				URL:    details.URL,
			}, nil
		}
	}

	return nil, nil
}

func (s *BuildService) listRecentBuilds(jobName string, max int) ([]buildRef, error) {
	path := fmt.Sprintf("job/%s/api/json?tree=builds[number,url]{0,%d}", jobName, max)
	resp, err := s.client.Get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, classifyHTTPError(resp, s.client.BuildURL(path))
	}

	var result buildListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Builds, nil
}

func (s *BuildService) getBuildDetails(jobName string, number int) (*buildDetails, error) {
	path := fmt.Sprintf("job/%s/%d/api/json", jobName, number)
	resp, err := s.client.Get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, classifyHTTPError(resp, s.client.BuildURL(path))
	}

	var details buildDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, err
	}

	return &details, nil
}

func matchesRef(details *buildDetails, ref string) bool {
	for _, a := range details.Actions {
		for _, p := range a.Parameters {
			if p.Name == "REF" && p.Value == ref {
				return true
			}
		}

		if a.LastBuiltRevision != nil {
			for _, b := range a.LastBuiltRevision.Branch {
				if b.Name == ref {
					return true
				}
			}
		}
	}

	return false
}

func (b *Build) State() string {
	switch b.Result {
	case "SUCCESS":
		return "succeeded"
	case "ABORTED":
		return "aborted"
	case "FAILURE", "UNSTABLE":
		return "failed"
	case "NOT_BUILT":
		return "blocked"
	}

	if b.Blocked {
		return "blocked"
	}
	if b.InQueue {
		return "queued"
	}
	if b.Building {
		return "running"
	}
	if b.Result == "" {
		return "running"
	}

	return "failed"
}
