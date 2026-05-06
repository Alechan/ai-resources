package auth

import (
	"github.com/alejandro-danos/jenkinsctl/internal/app"
)

type Provider struct {
	config *app.Config
}

func New(cfg *app.Config) *Provider {
	return &Provider{config: cfg}
}

func (p *Provider) BasicAuth() (string, string) {
	return p.config.Username, p.config.APIToken
}
