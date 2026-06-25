package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/curl"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
	"golang.org/x/term"
)

const initDocumentation = `To initialize ddctl with credentials from Chrome DevTools:

1. Open Chrome and log in to https://app.datadoghq.com/logs (Logs Explorer)
2. Open DevTools: Cmd+Option+I → Network tab
3. Find a POST request to /api/v1/logs-analytics/list
4. Right-click → Copy → Copy as cURL
5. Run: pbpaste | ddctl init

(Recommended: pbpaste pipes directly from clipboard, avoiding paste truncation)

Alternative: Save to file and run:
   ddctl init --curl-file ~/curl.txt

For more info:
   https://docs.datadoghq.com/api/latest/

`

func runInitCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	clear := fs.Bool("clear", false, "delete stored credentials and exit")
	curlFile := fs.String("curl-file", "", "path to file containing cURL command")

	if err := fs.Parse(args); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: pbpaste | ddctl init   (or: ddctl init --curl-file PATH, or: ddctl init --clear)"))
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

	if *curlFile == "" {
		// No flag provided; try reading from stdin
		return initFromStdin(ctx, svcs, cfg, os.Stdin, stdout, stderr)
	}

	// Read cURL from file
	curlData, err := os.ReadFile(*curlFile)
	if err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "ensure file exists and is readable"))
		return fail.CodeValidation
	}

	curlCmd := string(curlData)
	return initFromCurl(ctx, svcs, cfg, curlCmd, stdout, stderr)
}

// initFromStdin reads cURL from stdin and processes it.
func initFromStdin(ctx context.Context, svcs app.Services, cfg app.Config, stdin io.Reader, stdout, stderr io.Writer) int {
	return initFromStdinWithDetector(ctx, svcs, cfg, stdin, stdout, stderr, isTerminalInput)
}

func initFromStdinWithDetector(ctx context.Context, svcs app.Services, cfg app.Config, stdin io.Reader, stdout, stderr io.Writer, isTerminal func(io.Reader) bool) int {
	if isTerminal(stdin) {
		fmt.Fprint(stdout, initDocumentation)
		return fail.CodeOK
	}

	data, err := io.ReadAll(stdin)
	if err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "unable to read stdin"))
		return fail.CodeValidation
	}

	curlCmd := string(data)
	if strings.TrimSpace(curlCmd) == "" {
		fmt.Fprint(stdout, initDocumentation)
		return fail.CodeOK
	}

	return initFromCurl(ctx, svcs, cfg, curlCmd, stdout, stderr)
}

func isTerminalInput(stdin io.Reader) bool {
	file, ok := stdin.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

// initFromCurl extracts credentials from a cURL command and stores them.
func initFromCurl(ctx context.Context, svcs app.Services, cfg app.Config, curlCmd string, stdout, stderr io.Writer) int {
	// Extract cookie and CSRF token from cURL
	cookieStr, err := curl.ExtractCookieHeader(curlCmd)
	if err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "ensure the cURL command includes a Cookie header or -b flag"))
		return fail.CodeValidation
	}

	csrfToken := curl.ExtractCSRFToken(curlCmd)

	if csrfToken != "" {
		cookieStr = cookieStr + "; dd_csrf_token=" + csrfToken
	}

	// Sanitize and validate
	sanitized, dropped := sanitizeCookieString(cookieStr)
	if len(dropped) > 0 {
		fmt.Fprintf(stderr, "warning: dropped invalid cookies: %s\n", strings.Join(dropped, ", "))
	}

	if err := validateInitAuthMaterial(sanitized); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "provide a fresh cURL command from Chrome DevTools"))
		return fail.CodeValidation
	}

	// Store credentials
	if err := svcs.Auth.Store(sanitized); err != nil {
		writeError(stderr, fail.NewAuth(err.Error(), "ensure you have access to the macOS Keychain"))
		return fail.CodeAuth
	}

	// Show stored count and run doctor
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
		if encErr := enc.Encode(out); encErr != nil {
			writeError(stderr, fail.NewAPI(encErr.Error(), "unable to encode init result", ""))
			return fail.CodeAPI
		}
		return fail.CodeOK
	}

	fmt.Fprintf(stdout, "\nstored %d cookies for %s\n", count, cfg.Site)
	fmt.Fprintln(stdout, "verifying credentials…")
	return runDoctorCmd(ctx, svcs, cfg, nil, stdout, stderr)
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
