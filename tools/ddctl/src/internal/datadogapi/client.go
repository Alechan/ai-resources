package datadogapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

// Client is an HTTP client for the DataDog API, authenticated via Chrome cookies.
type Client struct {
	httpClient *http.Client
	site       string
	cookies    auth.CookieProvider
}

// NewClient creates a new DataDog API client.
func NewClient(httpClient *http.Client, site string, cookies auth.CookieProvider) *Client {
	return &Client{httpClient: httpClient, site: site, cookies: cookies}
}

func (c *Client) baseURL() string {
	return "https://app." + c.site
}

// Get performs an authenticated GET request and decodes the JSON response into out.
func (c *Client) Get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL()+path, nil)
	if err != nil {
		return fail.MapNetworkOrAPI(err)
	}
	if err := c.addAuth(req); err != nil {
		return err
	}
	return c.do(req, out)
}

// Post performs an authenticated POST request with a JSON body and decodes the JSON response into out.
// If a CSRF token cookie is present, it is automatically injected as _authentication_token in the
// request body (required by DataDog's browser UI endpoints).
func (c *Client) Post(ctx context.Context, path string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fail.NewAPI("failed to marshal request body", "", err.Error())
	}
	// Inject _authentication_token from CSRF cookie if available.
	if csrf := c.csrfToken(); csrf != "" {
		var m map[string]any
		if json.Unmarshal(b, &m) == nil {
			m["_authentication_token"] = csrf
			if rb, err := json.Marshal(m); err == nil {
				b = rb
			}
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL()+path, bytes.NewReader(b))
	if err != nil {
		return fail.MapNetworkOrAPI(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if err := c.addAuth(req); err != nil {
		return err
	}
	return c.do(req, out)
}

// csrfToken returns the CSRF token value from stored cookies, or empty string if not found.
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

// Probe sends a GET to the given path and returns true if any HTTP response is received
// (regardless of status code). Used for reachability checks.
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

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fail.MapNetworkOrAPI(err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fail.NewAPI("failed to read response body", "", "")
	}
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
			string(bodyBytes),
		)
	}
	if out != nil {
		if err := json.Unmarshal(bodyBytes, out); err != nil {
			return fail.NewAPI("failed to decode response", "", string(bodyBytes))
		}
	}
	return nil
}
