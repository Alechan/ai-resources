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

	cfg := app.NewConfig(opts.site, opts.timeout, opts.json, opts.debug)
	svcs := app.NewServices(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	switch cmd {
	case "init":
		return runInitCmd(ctx, svcs, cfg, cmdArgs, stdout, stderr)
	case "doctor":
		return runDoctorCmd(ctx, svcs, cfg, cmdArgs, stdout, stderr)
	case "logs-query":
		return runLogsQueryCmd(ctx, svcs, cfg, cmdArgs, stdout, stderr)
	case "monitors-list":
		return runMonitorsListCmd(ctx, svcs, cfg, cmdArgs, stdout, stderr)
	case "monitors-get":
		return runMonitorsGetCmd(ctx, svcs, cfg, cmdArgs, stdout, stderr)
	case "events-list":
		return runEventsListCmd(ctx, svcs, cfg, cmdArgs, stdout, stderr)
	case "help", "--help", "-h":
		printUsage(stdout)
		return fail.CodeOK
	default:
		err := fail.NewValidation("unknown command", "use one of: init, doctor, logs-query, monitors-list, monitors-get, events-list")
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
}

type rootOptions struct {
	site    string
	timeout time.Duration
	json    bool
	debug   bool
}

func parseRootArgs(args []string) (rootOptions, string, []string, error) {
	site := strings.TrimSpace(os.Getenv("DDCTL_SITE"))
	if site == "" {
		site = "datadoghq.com"
	}
	opts := rootOptions{
		site:    site,
		timeout: 30 * time.Second,
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
	fmt.Fprintln(w, "  init            Store DataDog session cookies from a cURL command or raw cookie string")
	fmt.Fprintln(w, "  doctor          Check credentials, DataDog auth, and reachability")
	fmt.Fprintln(w, "  logs-query      Query DataDog logs")
	fmt.Fprintln(w, "  monitors-list   List DataDog monitors")
	fmt.Fprintln(w, "  monitors-get    Get a specific DataDog monitor by ID")
	fmt.Fprintln(w, "  events-list     List DataDog events")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Global flags:")
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
