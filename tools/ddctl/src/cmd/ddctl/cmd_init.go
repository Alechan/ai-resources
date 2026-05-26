package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/curl"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

func runInitCmd(_ context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	curlVal := fs.String("curl", "", "cURL command copied from Chrome DevTools")
	cookieVal := fs.String("cookie", "", "raw Cookie header value")
	csrfVal := fs.String("csrf-token", "", "x-csrf-token value (extracted from cURL automatically if --curl is used)")
	clear := fs.Bool("clear", false, "delete stored credentials and exit")

	if err := fs.Parse(args); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl init [--curl <cmd>] [--cookie <str>] [--clear]"))
		return fail.CodeValidation
	}

	if *clear {
		if err := svcs.Auth.Delete(); err != nil {
			writeError(stderr, fail.NewAuth(err.Error(), ""))
			return fail.CodeAuth
		}
		fmt.Fprintln(stdout, "credentials cleared")
		return fail.CodeOK
	}

	if *curlVal == "" && *cookieVal == "" {
		writeError(stderr, fail.NewValidation("must provide --curl or --cookie", "usage: ddctl init --curl '<paste cURL here>'"))
		return fail.CodeValidation
	}
	if *curlVal != "" && *cookieVal != "" {
		writeError(stderr, fail.NewValidation("--curl and --cookie are mutually exclusive", "provide only one"))
		return fail.CodeValidation
	}

	cookieStr := *cookieVal
	if *curlVal != "" {
		var err error
		cookieStr, err = curl.ExtractCookieHeader(*curlVal)
		if err != nil {
			writeError(stderr, fail.NewValidation(err.Error(), "ensure the cURL command includes a Cookie header"))
			return fail.CodeValidation
		}
		// Auto-extract CSRF token from x-csrf-token header if not explicitly provided.
		if *csrfVal == "" {
			*csrfVal = curl.ExtractCSRFToken(*curlVal)
		}
	}

	// Append CSRF token as synthetic dd_csrf_token cookie so the API client can inject it.
	if *csrfVal != "" {
		cookieStr = cookieStr + "; dd_csrf_token=" + *csrfVal
	}

	sanitizedCookieStr, dropped := sanitizeCookieString(cookieStr)
	if len(dropped) > 0 {
		fmt.Fprintf(stderr, "warning: dropped invalid cookies: %s\n", strings.Join(dropped, ", "))
	}
	if err := validateInitAuthMaterial(sanitizedCookieStr); err != nil {
		writeError(stderr, fail.NewValidation(
			err.Error(),
			`provide a CSRF token and at least one of dogweb/dogwebu/_dd_s_v2 (try "ddctl init --curl '<fresh cURL>'")`,
		))
		return fail.CodeValidation
	}

	if err := svcs.Auth.Store(sanitizedCookieStr); err != nil {
		writeError(stderr, fail.NewAuth(err.Error(), "ensure you have access to the macOS Keychain"))
		return fail.CodeAuth
	}

	// Count cookies by loading them back via the same parser.
	cookies, err := svcs.Auth.Cookies()
	if err != nil {
		writeError(stderr, fail.NewAuth(err.Error(), ""))
		return fail.CodeAuth
	}
	count := len(cookies)

	if cfg.JSON {
		out := struct {
			Site          string `json:"site"`
			CookiesStored int    `json:"cookies_stored"`
		}{Site: cfg.Site, CookiesStored: count}
		enc := json.NewEncoder(stdout)
		if err := enc.Encode(out); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode init result", ""))
			return fail.CodeAPI
		}
		return fail.CodeOK
	}

	fmt.Fprintf(stdout, "stored %d cookies for %s\n", count, cfg.Site)
	fmt.Fprintln(stdout, `run "ddctl doctor" to verify connectivity`)
	return fail.CodeOK
}

func sanitizeCookieString(cookieStr string) (string, []string) {
	parts := strings.Split(cookieStr, ";")
	clean := make([]string, 0, len(parts))
	droppedSet := make(map[string]struct{})
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, "=")
		if idx <= 0 {
			droppedSet[part] = struct{}{}
			continue
		}
		name := strings.TrimSpace(part[:idx])
		value := strings.TrimSpace(part[idx+1:])
		if name == "" || value == "" || strings.Contains(value, `"`) {
			droppedSet[name] = struct{}{}
			continue
		}
		clean = append(clean, name+"="+value)
	}
	dropped := make([]string, 0, len(droppedSet))
	for name := range droppedSet {
		if name == "" {
			name = "<unknown>"
		}
		dropped = append(dropped, name)
	}
	sort.Strings(dropped)
	return strings.Join(clean, "; "), dropped
}

func validateInitAuthMaterial(cookieStr string) error {
	hasCSRF := false
	hasSession := false
	for _, part := range strings.Split(cookieStr, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, "=")
		if idx <= 0 {
			continue
		}
		name := strings.TrimSpace(part[:idx])
		switch name {
		case "dd_csrf_token", "_csrf":
			hasCSRF = true
		case "dogweb", "dogwebu", "_dd_s_v2":
			hasSession = true
		}
	}
	if !hasCSRF {
		return fmt.Errorf("missing CSRF token (dd_csrf_token/_csrf)")
	}
	if !hasSession {
		return fmt.Errorf("missing session cookie (dogweb/dogwebu/_dd_s_v2)")
	}
	return nil
}
