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

// Client is an HTTP client for the DataDog API, authenticated via Keychain session cookies.
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
		b, err := c.marshalBody(path, body)
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

func (c *Client) marshalBody(path string, body any) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fail.NewAPI("failed to marshal request body", "", err.Error())
	}
	if !pathNeedsCSRF(path) {
		return b, nil
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
		return fail.NewAuth(
			"failed to load Keychain session credentials: "+err.Error(),
			fmt.Sprintf("run ddctl init and ensure Keychain access for app.%s", c.site),
		)
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
	maxAttempts := 1
	if req.Method == http.MethodGet {
		maxAttempts = 3
	}

	var bodyBytes []byte
	status := 0
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 250 * time.Millisecond)
			if c.debug != nil {
				c.debug.Printf("retry %s %s after HTTP %d (attempt %d)", req.Method, req.URL.Path, status, attempt+1)
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fail.MapNetworkOrAPI(err)
		}
		bodyBytes, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fail.NewAPI("failed to read response body", "", "")
		}
		status = resp.StatusCode
		c.logRequest(req.Method, req.URL.Path, status, started)

		if req.Method == http.MethodGet && (status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable) && attempt < maxAttempts-1 {
			continue
		}
		break
	}

	if status == 401 || status == 403 {
		return fail.NewAuth(
			fmt.Sprintf("HTTP %d: authentication required", status),
			fmt.Sprintf("refresh Keychain session credentials with ddctl init (app.%s)", c.site),
		)
	}
	if status < 200 || status >= 300 {
		return apiErrorFromResponse(status, string(bodyBytes))
	}
	if out != nil && len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, out); err != nil {
			return fail.NewAPI("failed to decode response", "", redactResponseBody(string(bodyBytes)))
		}
	}
	return nil
}
