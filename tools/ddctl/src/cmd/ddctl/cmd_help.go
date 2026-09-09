package main

import (
	"fmt"
	"io"
	"strings"
)

func printCommandHelp(w io.Writer, cmd string, args []string) {
	switch cmd {
	case "", "help":
		printUsage(w)
	case "dashboards":
		printDashboardsHelp(w, args)
	case "notebooks":
		printNotebooksHelp(w, args)
	case "init":
		fmt.Fprint(w, helpInit)
	case "doctor":
		fmt.Fprint(w, helpDoctor)
	case "logs-query":
		fmt.Fprint(w, helpLogsQuery)
	case "monitors":
		printMonitorsHelp(w, args)
	case "events-list":
		fmt.Fprint(w, helpEventsList)
	case "metrics-query":
		fmt.Fprint(w, helpMetricsQuery)
	default:
		fmt.Fprintf(w, "Unknown command %q.\n\n", cmd)
		printUsage(w)
	}
}

func printNotebooksHelp(w io.Writer, args []string) {
	sub := firstNonHelpArg(args)
	switch sub {
	case "", "help":
		fmt.Fprint(w, helpNotebooks)
	case "get":
		fmt.Fprint(w, helpNotebooksGet)
	case "create":
		fmt.Fprint(w, helpNotebooksCreate)
	case "update":
		fmt.Fprint(w, helpNotebooksUpdate)
	case "validate":
		fmt.Fprint(w, helpNotebooksValidate)
	default:
		fmt.Fprintf(w, "Unknown notebooks subcommand %q.\n\n", sub)
		fmt.Fprint(w, helpNotebooks)
	}
}

const helpInit = `Usage:
  ddctl init [--curl <curl>] [--cookie <cookie>] [--csrf-token <token>]

Store DataDog session cookies in the macOS Keychain.

Prefer piping a copied cURL:
  pbpaste | ddctl init

Exit codes:
  0  success
  2  validation / usage error
`

const helpDoctor = `Usage:
  ddctl doctor

Check Keychain credentials, DataDog reachability, and a sample auth query.

Exit codes:
  0  all checks passed
  3  authentication failure
  4  network failure
  5  API failure
`

const helpLogsQuery = `Usage:
  ddctl logs-query --query <filter> [--from <time>] [--to <time>] [--limit <n>]

Query DataDog logs. Does not print raw session cookies.

Exit codes:
  0  success
  2  validation / usage error
  3  authentication failure
`

const helpEventsList = `Usage:
  ddctl events-list [--from <time>] [--to <time>]

List DataDog events in a time range.

Exit codes:
  0  success
  2  validation / usage error
`

const helpMetricsQuery = `Usage:
  ddctl metrics-query --query <query> [--from <time>] [--to <time>]

Query DataDog timeseries metrics.

Exit codes:
  0  success
  2  validation / usage error
`

const helpNotebooks = `Usage:
  ddctl notebooks <get|create|update|validate> [flags]

Subcommands:
  get       Fetch a notebook by ID
  create    Create a notebook from JSON
  update    Replace a notebook (requires --replace-all)
  validate  Check payload and preflight metric queries

Use ddctl notebooks <subcommand> --help for flags and examples.
`

const helpNotebooksGet = `Usage:
  ddctl notebooks get <id> [--include-metadata]

Positional arguments:
  id    Notebook ID
`

const helpNotebooksCreate = `Usage:
  ddctl notebooks create --from-file <path> [--name <name>] [--time <live_span>]

Mutates DataDog. Do not run unless create was requested.
`

const helpNotebooksUpdate = `Usage:
  ddctl notebooks update <id> --from-file <path> --replace-all

Mutates DataDog. PUT is full replacement; --replace-all is required.
`

const helpNotebooksValidate = `Usage:
  ddctl notebooks validate --from-file <path> [--from <time>] [--to <time>] [--allow-empty-series]

Validate notebook structure and preflight timeseries metric queries.
Does not change Datadog.

No-data in the selected window is a warning (exit 0), not invalidity.
--allow-empty-series is accepted for compatibility; it does not change exit code.
`

func firstNonHelpArg(args []string) string {
	for _, a := range args {
		if a == "-h" || a == "--help" || a == "help" {
			continue
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		return a
	}
	return ""
}
