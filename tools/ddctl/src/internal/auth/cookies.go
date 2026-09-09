package auth

import (
	"net/http"
	"strings"
)

// JoinCookiePairs joins name=value pairs into a Cookie header string.
func JoinCookiePairs(pairs []string) string {
	return strings.Join(pairs, "; ")
}

// ParseCookieString splits a raw Cookie header string into []*http.Cookie.
func ParseCookieString(cookieStr string) []*http.Cookie {
	var cookies []*http.Cookie
	for _, part := range strings.Split(cookieStr, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, "=")
		if idx < 0 {
			cookies = append(cookies, &http.Cookie{Name: part})
			continue
		}
		cookies = append(cookies, &http.Cookie{
			Name:  strings.TrimSpace(part[:idx]),
			Value: strings.TrimSpace(part[idx+1:]),
		})
	}
	return cookies
}
