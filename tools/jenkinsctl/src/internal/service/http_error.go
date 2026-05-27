package service

import (
	"fmt"
	"net/http"
	"strings"
)

type HTTPError struct {
	Kind        string
	StatusCode  int
	URL         string
	AuthContext string
	Location    string
	Hint        string
}

func (e *HTTPError) Error() string {
	parts := []string{
		fmt.Sprintf("kind=%s", e.Kind),
		fmt.Sprintf("status=%d", e.StatusCode),
		fmt.Sprintf("url=%s", e.URL),
	}

	if e.AuthContext != "" {
		parts = append(parts, fmt.Sprintf("auth_context=%s", e.AuthContext))
	}
	if e.Location != "" {
		parts = append(parts, fmt.Sprintf("location=%s", e.Location))
	}
	if e.Hint != "" {
		parts = append(parts, fmt.Sprintf("hint=%s", e.Hint))
	}

	return strings.Join(parts, " ")
}

func classifyHTTPError(resp *http.Response, fallbackURL string) error {
	targetURL := fallbackURL
	if resp.Request != nil && resp.Request.URL != nil {
		targetURL = resp.Request.URL.String()
	}

	statusCode := resp.StatusCode
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &HTTPError{
			Kind:        "auth",
			StatusCode:  statusCode,
			URL:         targetURL,
			AuthContext: authContextLabel(targetURL),
			Hint:        "verify --url points to the intended Jenkins instance and regenerate the token for that same instance",
		}
	case statusCode == http.StatusNotFound:
		return &HTTPError{
			Kind:       "not_found",
			StatusCode: statusCode,
			URL:        targetURL,
			Hint:       "check the job path and use '/job/' separators for nested folders (e.g. folder/job/pipeline)",
		}
	case statusCode >= 300 && statusCode < 400:
		return &HTTPError{
			Kind:       "redirect",
			StatusCode: statusCode,
			URL:        targetURL,
			Location:   resp.Header.Get("Location"),
			Hint:       "use the final Jenkins base URL directly (avoid SSO/login redirect endpoints)",
		}
	default:
		return &HTTPError{
			Kind:       "unexpected",
			StatusCode: statusCode,
			URL:        targetURL,
			Hint:       "inspect Jenkins availability and permissions",
		}
	}
}

func authContextLabel(rawURL string) string {
	lowerURL := strings.ToLower(rawURL)
	switch {
	case strings.Contains(lowerURL, "acceptance"):
		return "acceptance"
	case strings.Contains(lowerURL, "prod"), strings.Contains(lowerURL, "production"):
		return "prod"
	case strings.Contains(lowerURL, "ci"):
		return "ci"
	default:
		return ""
	}
}
