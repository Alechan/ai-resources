package main

import (
	"fmt"
	"io"
)

func printLogsHelp(w io.Writer, args []string) {
	sub := firstNonHelpArg(args)
	switch sub {
	case "":
		fmt.Fprint(w, helpLogs)
	case "query":
		fmt.Fprint(w, helpLogsQuery)
	case "export":
		fmt.Fprint(w, helpLogsExport)
	case "get":
		fmt.Fprint(w, helpLogsGet)
	default:
		fmt.Fprintf(w, "Unknown logs subcommand %q.\nValid choices: query, export, get.\n\n", sub)
		fmt.Fprint(w, helpLogs)
	}
}

const helpLogs = `Usage:
  ddctl logs <subcommand> [flags]

Subcommands:
  query     Query DataDog logs
  export    Export logs to NDJSON or CSV
  get       Fetch a single log event by ID

Use ddctl logs <subcommand> --help for flags and examples.
`

const helpLogsQuery = `Usage:
  ddctl logs query [--query <filter>] [--from <time>] [--to <time>] [--limit <n>]

Flags:
  --query, -q <filter>     Search query (default: *)
  --from <time>            Start time (default: now-1h)
  --to <time>              End time (default: now)
  --limit <n>              Max results per page or total with --all (default: 50)
  --all                    Auto-paginate until --limit or end of results
  --count-only             Return metadata and hit_count only
  --cursor <value>         Pagination cursor from a prior result
  --fields <list>          JSON field projection (requires --json)
  --verbose, -v            Include selected custom fields in text output
  --verbose-keys <list>    Custom keys to print with --verbose

JSON output includes full event.custom maps. Text mode stays one line per event unless --verbose.

Does not print raw session cookies.

Exit codes:
  0  success
  2  validation / usage error
  3  authentication failure
`

const helpLogsExport = `Usage:
  ddctl logs export [--query <filter>] [--from <time>] [--to <time>] [--output <path>]

Flags:
  --query, -q <filter>     Search query (default: *)
  --from <time>            Start time (default: now-1h)
  --to <time>              End time (default: now)
  --limit <n>              Max exported events (default: 1000)
  --format <ndjson|csv>    Export format (default: ndjson)
  --output, -o <path>      Output file (default: stdout)
  --fields <list>          Field projection (default: full event)

Prints a summary to stderr: exported N events (hit_count=M) to <path>
`

const helpLogsGet = `Usage:
  ddctl logs get <event_id> [--from <time>] [--to <time>]

Fetch one log event after spotting its ID in query results.
Uses @evt.id lookup against the logs-analytics list endpoint.

Flags:
  --from <time>            Start time (default: now-1h)
  --to <time>              End time (default: now)
  --fields <list>          JSON field projection (requires --json)
`
