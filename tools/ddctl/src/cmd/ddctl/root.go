package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

func Execute(args []string, stdout, stderr io.Writer) int {
	opts, cmd, cmdArgs, err := parseRootArgs(args)
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	if cmd == "" {
		printUsage(stderr)
		return fail.CodeValidation
	}

	cfg := app.NewConfig(opts.cookiesPath, opts.site, opts.timeout, opts.json, opts.debug)
	svcs := app.NewServices(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	switch cmd {
	case "doctor":
		return runDoctorCmd(ctx, svcs, cfg, cmdArgs, stdout, stderr)
	case "logs-query":
		return runLogsQueryCmd(ctx, svcs, cfg, cmdArgs, stdout, stderr)
	case "help", "--help", "-h":
		printUsage(stdout)
		return fail.CodeOK
	default:
		err := fail.NewValidation("unknown command", "use one of: doctor, logs-query")
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
}

type rootOptions struct {
	cookiesPath string
	site        string
	timeout     time.Duration
	json        bool
	debug       bool
}

func parseRootArgs(args []string) (rootOptions, string, []string, error) {
	cookiesPath := strings.TrimSpace(os.Getenv("DDCTL_COOKIES_PATH"))
	if cookiesPath == "" {
		cookiesPath = "~/Library/Application Support/Google/Chrome/Default/Cookies"
	}
	site := strings.TrimSpace(os.Getenv("DDCTL_SITE"))
	if site == "" {
		site = "datadoghq.com"
	}
	opts := rootOptions{
		cookiesPath: cookiesPath,
		site:        site,
		timeout:     30 * time.Second,
	}
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-h" || a == "--help":
			return opts, "help", nil, nil
		case a == "--json":
			opts.json = true
		case a == "--debug":
			opts.debug = true
		case a == "--cookies-path":
			if i+1 >= len(args) {
				return opts, "", nil, fail.NewValidation("missing value for --cookies-path", "provide a valid path to the Chrome Cookies file")
			}
			i++
			opts.cookiesPath = args[i]
		case strings.HasPrefix(a, "--cookies-path="):
			opts.cookiesPath = strings.TrimPrefix(a, "--cookies-path=")
		case a == "--site":
			if i+1 >= len(args) {
				return opts, "", nil, fail.NewValidation("missing value for --site", "provide a DataDog site domain like datadoghq.com")
			}
			i++
			opts.site = args[i]
		case strings.HasPrefix(a, "--site="):
			opts.site = strings.TrimPrefix(a, "--site=")
		case a == "--timeout":
			if i+1 >= len(args) {
				return opts, "", nil, fail.NewValidation("missing value for --timeout", "provide a duration like 30s")
			}
			i++
			d, err := time.ParseDuration(args[i])
			if err != nil {
				return opts, "", nil, fail.NewValidation("invalid --timeout value", "provide a duration like 30s")
			}
			opts.timeout = d
		case strings.HasPrefix(a, "--timeout="):
			d, err := time.ParseDuration(strings.TrimPrefix(a, "--timeout="))
			if err != nil {
				return opts, "", nil, fail.NewValidation("invalid --timeout value", "provide a duration like 30s")
			}
			opts.timeout = d
		default:
			rest = append(rest, a)
		}
	}
	if len(rest) == 0 {
		return opts, "", nil, nil
	}
	return opts, rest[0], rest[1:], nil
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: ddctl [global flags] <command> [flags]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  doctor       Check Chrome cookies, DataDog auth, and reachability")
	fmt.Fprintln(w, "  logs-query   Query DataDog logs")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Global flags:")
	fmt.Fprintln(w, "  --cookies-path <path>  Path to Chrome Cookies SQLite file")
	fmt.Fprintln(w, "                           Env override: DDCTL_COOKIES_PATH")
	fmt.Fprintln(w, "                           Default: ~/Library/Application Support/Google/Chrome/Default/Cookies")
	fmt.Fprintln(w, "  --site <domain>        DataDog site domain (default: datadoghq.com)")
	fmt.Fprintln(w, "                           Env override: DDCTL_SITE")
	fmt.Fprintln(w, "  --timeout <duration>   Timeout per command (default: 30s)")
	fmt.Fprintln(w, "  --json                 JSON output")
	fmt.Fprintln(w, "  --debug                Debug logging")
}

func writeError(w io.Writer, err error) {
	var e *fail.Error
	if errors.As(err, &e) {
		if e.Action == "" {
			fmt.Fprintf(w, "Error [%s]: %s\n", e.Category, e.Message)
			return
		}
		fmt.Fprintf(w, "Error [%s]: %s\nAction: %s\n", e.Category, e.Message, e.Action)
		return
	}
	fmt.Fprintf(w, "Error: %s\n", strings.TrimSpace(err.Error()))
}
