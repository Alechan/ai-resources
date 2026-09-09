package main

import (
	"context"
	"encoding/json"
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
		writeError(stderr, err, app.Config{JSON: opts.json})
		return fail.ExitCode(err)
	}
	if opts.help {
		printCommandHelp(stdout, cmd, cmdArgs)
		return fail.CodeOK
	}
	if cmd == "help" {
		if len(cmdArgs) == 0 {
			printUsage(stdout)
			return fail.CodeOK
		}
		printCommandHelp(stdout, cmdArgs[0], cmdArgs[1:])
		return fail.CodeOK
	}
	if cmd == "" {
		printUsage(stderr)
		return fail.CodeValidation
	}

	cfg := app.NewConfig(opts.site, opts.timeout, opts.json, opts.debug)
	svcs := app.NewServices(cfg)
	if cfg.Debug {
		svcs.SetDebug(stderr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	switch cmd {
	case "init":
		return runInitCmd(ctx, svcs, cfg, cmdArgs, stdout, stderr)
	case "doctor":
		return runDoctorCmd(ctx, svcs, cfg, cmdArgs, stdout, stderr)
	case "logs":
		return runLogsCmd(ctx, svcs, cfg, cmdArgs, stdout, stderr)
	case "metrics":
		return runMetricsCmd(ctx, svcs, cfg, cmdArgs, stdout, stderr)
	case "events":
		return runEventsCmd(ctx, svcs, cfg, cmdArgs, stdout, stderr)
	case "logs-query":
		return rejectRemovedCommand(stderr, cfg, "logs-query", "ddctl logs query")
	case "metrics-query":
		return rejectRemovedCommand(stderr, cfg, "metrics-query", "ddctl metrics query")
	case "events-list":
		return rejectRemovedCommand(stderr, cfg, "events-list", "ddctl events list")
	case "monitors":
		return runMonitorsCmd(ctx, svcs, cfg, cmdArgs, stdout, stderr)
	case "notebooks":
		return runNotebooksCmd(ctx, svcs, cfg, cmdArgs, stdout, stderr)
	case "dashboards":
		return runDashboardsCmd(ctx, svcs, cfg, cmdArgs, stdout, stderr)
	default:
		err := fail.NewValidation("unknown command", "use one of: init, doctor, logs, metrics, events, monitors, notebooks, dashboards")
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
}

func rejectRemovedCommand(stderr io.Writer, cfg app.Config, oldCmd, newUsage string) int {
	writeError(stderr, fail.NewValidation("removed command "+oldCmd, "use: "+newUsage), cfg)
	return fail.CodeValidation
}

type rootOptions struct {
	site    string
	timeout time.Duration
	json    bool
	debug   bool
	help    bool
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
			opts.help = true
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
	fmt.Fprintln(w, "  init            Store DataDog session cookies from stdin or --curl-file")
	fmt.Fprintln(w, "  doctor          Check credentials, DataDog auth, and reachability")
	fmt.Fprintln(w, "  logs            Query DataDog logs (ddctl logs query)")
	fmt.Fprintln(w, "  metrics         Query DataDog metrics (ddctl metrics query)")
	fmt.Fprintln(w, "  events          List DataDog events (ddctl events list)")
	fmt.Fprintln(w, "  monitors        Manage DataDog monitors (list/get/validate/create/update/mute/unmute/delete)")
	fmt.Fprintln(w, "  notebooks       Manage DataDog notebooks (get/create/update/validate)")
	fmt.Fprintln(w, "  dashboards      Manage DataDog dashboards (get/list/search/create/update/validate/clone/delete)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Global flags:")
	fmt.Fprintln(w, "  --site <domain>        DataDog site domain (default: datadoghq.com)")
	fmt.Fprintln(w, "                           Env override: DDCTL_SITE")
	fmt.Fprintln(w, "  --timeout <duration>   Timeout per command (default: 30s)")
	fmt.Fprintln(w, "  --json                 JSON output")
	fmt.Fprintln(w, "  --debug                Debug logging")
}

func writeError(w io.Writer, err error, cfg app.Config) {
	e := fail.AsError(err)
	if cfg.JSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(e.Envelope())
		return
	}
	if e.Action == "" {
		fmt.Fprintf(w, "Error [%s]: %s\n", e.Category, e.Message)
	} else {
		fmt.Fprintf(w, "Error [%s]: %s\nAction: %s\n", e.Category, e.Message, e.Action)
	}
	if cfg.Debug && e.Details != "" {
		fmt.Fprintf(w, "Details: %s\n", e.Details)
	}
}
