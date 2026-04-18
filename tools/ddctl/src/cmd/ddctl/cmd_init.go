package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/curl"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

func runInitCmd(_ context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	curlVal := fs.String("curl", "", "cURL command copied from Chrome DevTools")
	cookieVal := fs.String("cookie", "", "raw Cookie header value")
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
	}

	if err := svcs.Auth.Store(cookieStr); err != nil {
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
