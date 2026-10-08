package capture

import (
	"net/url"
	"regexp"
	"strings"
)

const DefaultAccountKey = "0"

var (
	authUserPath     = regexp.MustCompile(`/u/(\d+)(?:/|$)`)
	longPathSegment  = regexp.MustCompile(`^[A-Za-z0-9_-]{20,}$`)
	accountPathToken = regexp.MustCompile(`/u/\d+`)
)

func AllowedHost(host string) bool {
	host = strings.ToLower(host)
	switch host {
	case "chat.google.com", "clients6.google.com", "chat.googleapis.com":
		return true
	default:
		return strings.HasSuffix(host, ".clients6.google.com")
	}
}

func AccountKey(rawURL, authUserHeader, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if parsed, err := url.Parse(rawURL); err == nil {
		if match := authUserPath.FindStringSubmatch(parsed.Path); len(match) == 2 {
			return match[1]
		}
	}
	if authUserHeader != "" {
		return authUserHeader
	}
	return DefaultAccountKey
}

func IsStaticPath(path string) bool {
	base := strings.ToLower(path)
	if i := strings.Index(base, "?"); i >= 0 {
		base = base[:i]
	}
	for _, ext := range []string{".js", ".mjs", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".map", ".webp"} {
		if strings.HasSuffix(base, ext) {
			return true
		}
	}
	return strings.Contains(base, "/static/") || strings.Contains(base, "/_/scs/")
}

func pathRank(path string) int {
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(lower, "/api/"), strings.Contains(lower, "/v1/"), strings.Contains(lower, "batchexecute"), strings.Contains(lower, "/$rpc/"):
		return 0
	default:
		return 1
	}
}

func NormalizePath(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	path := accountPathToken.ReplaceAllString(parsed.Path, "/u/ACCOUNT")
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if isRedactableSegment(segment) {
			segments[i] = "REDACTED_ID"
		}
	}
	return strings.Join(segments, "/")
}

func isRedactableSegment(segment string) bool {
	if !longPathSegment.MatchString(segment) {
		return false
	}
	if strings.Contains(segment, "_") && segment == strings.ToLower(segment) {
		return false
	}
	return true
}
