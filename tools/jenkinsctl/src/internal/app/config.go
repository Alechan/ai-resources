package app

import (
	"github.com/alejandro-danos/jenkinsctl/internal/fail"
	"os"
)

type Config struct {
	Username string
	APIToken string
}

func LoadConfig() (*Config, error) {
	username := os.Getenv("JENKINS_USERNAME")
	apiToken := os.Getenv("JENKINS_API_TOKEN")

	if username == "" || apiToken == "" {
		return nil, fail.New(fail.ErrConfig, "JENKINS_USERNAME and JENKINS_API_TOKEN must be set")
	}

	return &Config{
		Username: username,
		APIToken: apiToken,
	}, nil
}
