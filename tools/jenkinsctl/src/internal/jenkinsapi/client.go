package jenkinsapi

import (
	"fmt"
	"github.com/alejandro-danos/jenkinsctl/internal/auth"
	"net/http"
)

type Client struct {
	baseURL string
	auth    *auth.Provider
	http    *http.Client
}

func New(baseURL string, authProvider *auth.Provider) *Client {
	return &Client{
		baseURL: baseURL,
		auth:    authProvider,
		http:    &http.Client{},
	}
}

func (c *Client) Get(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", c.baseURL, path), nil)
	if err != nil {
		return nil, err
	}
	u, t := c.auth.BasicAuth()
	req.SetBasicAuth(u, t)
	return c.http.Do(req)
}
