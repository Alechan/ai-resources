package datadogapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

// Client is an HTTP client for the DataDog API, authenticated via Chrome cookies.
type Client struct {
	httpClient *http.Client
	site       string
	cookies    auth.CookieProvider
	debug      DebugLogger
}

// NewClient creates a new DataDog API client.
func NewClient(httpClient *http.Client, site string, cookies auth.CookieProvider) *Client {
	return &Client{httpClient: httpClient, site: site, cookies: cookies}
}

func (c *Client) SetDebugLogger(l DebugLogger) {
	c.debug = l
}

func (c *Client) baseURL() string {
	return "https://app." + c.site
}

// Get performs an authenticated GET request and decodes the JSON response into out.
func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.request(ctx, http.MethodGet, path, nil, out)
}

// Post performs an authenticated POST request with a JSON body and decodes the JSON response into out.
func (c *Client) Post(ctx context.Context, path string, body, out any) error {
	return c.request(ctx, http.MethodPost, path, body, out)
}

// Put performs an authenticated PUT request with a JSON body and decodes the JSON response into out.
func (c *Client) Put(ctx context.Context, path string, body, out any) error {
	return c.request(ctx, http.MethodPut, path, body, out)
}

// Delete performs an authenticated DELETE request and optionally decodes the JSON response into out.
func (c *Client) Delete(ctx context.Context, path string, out any) error {
	return c.request(ctx, http.MethodDelete, path, nil, out)
}

func (c *Client) request(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := c.marshalBody(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL()+path, reader)
	if err != nil {
		return fail.MapNetworkOrAPI(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if err := c.addAuth(req); err != nil {
		return err
	}
	started := time.Now()
	return c.do(req, out, started)
}

func (c *Client) marshalBody(body any) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fail.NewAPI("failed to marshal request body", "", err.Error())
	}
	if csrf := c.csrfToken(); csrf != "" {
		var m map[string]any
		if json.Unmarshal(b, &m) == nil {
			m["_authentication_token"] = csrf
			if rb, err := json.Marshal(m); err == nil {
				b = rb
			}
		}
	}
	return b, nil
}

func (c *Client) csrfToken() string {
	cookies, err := c.cookies.Cookies()
	if err != nil {
		return ""
	}
	for _, cookie := range cookies {
		if cookie.Name == "dd_csrf_token" || cookie.Name == "_csrf" {
			return cookie.Value
		}
	}
	return ""
}

func (c *Client) Probe(ctx context.Context, path string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL()+path, nil)
	if err != nil {
		return false
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}

func (c *Client) addAuth(req *http.Request) error {
	cookies, err := c.cookies.Cookies()
	if err != nil {
		return fail.NewAuth("failed to load Chrome cookies: "+err.Error(), "ensure Chrome has been used to visit app.datadoghq.com and that keychain access is granted")
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	for _, cookie := range cookies {
		if cookie.Name == "dd_csrf_token" || cookie.Name == "_csrf" {
			req.Header.Set("x-csrf-token", cookie.Value)
			break
		}
	}
	return nil
}

func (c *Client) do(req *http.Request, out any, started time.Time) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fail.MapNetworkOrAPI(err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fail.NewAPI("failed to read response body", "", "")
	}
	c.logRequest(req.Method, req.URL.Path, resp.StatusCode, started)
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fail.NewAuth(
			fmt.Sprintf("HTTP %d: authentication required", resp.StatusCode),
			"verify Chrome DataDog cookies are fresh; try visiting app.datadoghq.com",
		)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fail.NewAPI(
			fmt.Sprintf("HTTP %d", resp.StatusCode),
			"inspect API response",
			redactDetails(string(bodyBytes)),
		)
	}
	if out != nil && len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, out); err != nil {
			return fail.NewAPI("failed to decode response", "", redactDetails(string(bodyBytes)))
		}
	}
	return nil
}
