package jenkinsapi

import (
	"fmt"
	"net/http"
)

type Client struct {
	baseURL  string
	username string
	token    string
	http     *http.Client
}

func New(baseURL, username, token string) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		token:    token,
		http: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (c *Client) Get(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.BuildURL(path), nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.username, c.token)
	return c.http.Do(req)
}

func (c *Client) BuildURL(path string) string {
	return fmt.Sprintf("%s/%s", c.baseURL, path)
}
