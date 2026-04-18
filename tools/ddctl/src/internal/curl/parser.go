package curl

import (
	"fmt"
	"regexp"
	"strings"
)

// cookieHeaderRe matches -H 'Cookie: ...' or -H "Cookie: ..." or --header 'Cookie: ...'
// with case-insensitive header name, capturing the cookie value.
var cookieHeaderRe = regexp.MustCompile(`(?i)(?:-H|--header)\s+['"]Cookie:\s*([^'"]+)['"]`)

// cookieFlagRe matches -b '...' or --cookie '...' (the curl cookie flag),
// capturing the cookie string value.
var cookieFlagRe = regexp.MustCompile(`(?:-b|--cookie)\s+['"]([^'"]+)['"]`)

// ExtractCookieHeader parses a cURL command and returns the cookie string.
// It handles both -H 'Cookie: ...' and -b '...' / --cookie '...' forms.
// Returns an error if neither form is found.
func ExtractCookieHeader(curlCmd string) (string, error) {
	// Normalize multi-line cURL (lines joined with " \\\n")
	normalized := strings.ReplaceAll(curlCmd, "\\\n", " ")

	if match := cookieHeaderRe.FindStringSubmatch(normalized); match != nil {
		return strings.TrimSpace(match[1]), nil
	}
	if match := cookieFlagRe.FindStringSubmatch(normalized); match != nil {
		return strings.TrimSpace(match[1]), nil
	}
	return "", fmt.Errorf("no Cookie header or -b flag found in cURL command")
}
