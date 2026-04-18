package curl

import (
	"fmt"
	"regexp"
	"strings"
)

// cookieHeaderRe matches -H 'Cookie: ...' or -H "Cookie: ..." or --header 'Cookie: ...'
// with case-insensitive header name, capturing the cookie value.
var cookieHeaderRe = regexp.MustCompile(`(?i)(?:-H|--header)\s+['"]Cookie:\s*([^'"]+)['"]`)

// ExtractCookieHeader parses a cURL command and returns the value of the Cookie header.
// Returns an error if no Cookie header is found.
func ExtractCookieHeader(curlCmd string) (string, error) {
	// Normalize multi-line cURL (lines joined with " \\\n")
	normalized := strings.ReplaceAll(curlCmd, "\\\n", " ")

	match := cookieHeaderRe.FindStringSubmatch(normalized)
	if match == nil {
		return "", fmt.Errorf("no Cookie header found in cURL command")
	}
	return strings.TrimSpace(match[1]), nil
}
