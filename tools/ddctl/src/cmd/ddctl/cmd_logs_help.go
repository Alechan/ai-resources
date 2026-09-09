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
	default:
		fmt.Fprintf(w, "Unknown logs subcommand %q.\nValid choices: query.\n\n", sub)
		fmt.Fprint(w, helpLogs)
	}
}

const helpLogs = `Usage:
  ddctl logs <subcommand> [flags]

Subcommands:
  query     Query DataDog logs

Use ddctl logs query --help for flags and examples.
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

Does not print raw session cookies.

Exit codes:
  0  success
  2  validation / usage error
  3  authentication failure
`
