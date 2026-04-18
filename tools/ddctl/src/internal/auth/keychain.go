package auth

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

// CookieProvider returns HTTP cookies for use in requests.
type CookieProvider interface {
	Cookies() ([]*http.Cookie, error)
}

// KeychainProvider stores and loads DataDog session cookies in macOS Keychain.
type KeychainProvider struct {
	site string
}

// NewKeychainProvider creates a provider for the given DataDog site domain.
func NewKeychainProvider(site string) *KeychainProvider {
	return &KeychainProvider{site: site}
}

// Path returns a human-readable description of the credential location.
func (p *KeychainProvider) Path() string {
	return fmt.Sprintf("macOS Keychain (ddctl / %s)", p.site)
}

// Store saves the raw cookie string to the macOS Keychain.
func (p *KeychainProvider) Store(cookieStr string) error {
	cmd := exec.Command("security", "add-generic-password",
		"-s", "ddctl",
		"-a", p.site,
		"-w", cookieStr,
		"-U",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("keychain store: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Cookies loads the cookie string from Keychain and parses it into []*http.Cookie.
// Returns fail.NewAuth if no entry exists.
func (p *KeychainProvider) Cookies() ([]*http.Cookie, error) {
	out, err := exec.Command("security", "find-generic-password",
		"-s", "ddctl",
		"-a", p.site,
		"-w",
	).Output()
	if err != nil {
		return nil, fail.NewAuth(
			"no credentials found in Keychain",
			`run "ddctl init" to store your DataDog session`,
		)
	}
	cookieStr := strings.TrimSpace(string(out))
	return parseCookieString(cookieStr), nil
}

// Delete removes the Keychain entry.
func (p *KeychainProvider) Delete() error {
	cmd := exec.Command("security", "delete-generic-password",
		"-s", "ddctl",
		"-a", p.site,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("keychain delete: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// parseCookieString splits a raw Cookie header string into []*http.Cookie.
func parseCookieString(cookieStr string) []*http.Cookie {
	var cookies []*http.Cookie
	for _, part := range strings.Split(cookieStr, "; ") {
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
			Name:  part[:idx],
			Value: part[idx+1:],
		})
	}
	return cookies
}
