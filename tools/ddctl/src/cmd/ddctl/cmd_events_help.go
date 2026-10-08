package main

import (
	"fmt"
	"io"
)

func printEventsHelp(w io.Writer, args []string) {
	sub := firstNonHelpArg(args)
	switch sub {
	case "":
		fmt.Fprint(w, helpEvents)
	case "list":
		fmt.Fprint(w, helpEventsList)
	default:
		fmt.Fprintf(w, "Unknown events subcommand %q.\nValid choices: list.\n\n", sub)
		fmt.Fprint(w, helpEvents)
	}
}

const helpEvents = `Usage:
  ddctl events <subcommand> [flags]

Subcommands:
  list      List DataDog events in a time range

Use ddctl events list --help for flags and examples.
`

const helpEventsList = `Usage:
  ddctl events list [--from <time>] [--to <time>]

Flags:
  --from <time>            Start time (default: now-1h)
  --to <time>              End time (default: now)
  --sources <csv>          Filter by event sources
  --tags <csv>             Filter by tags (e.g. env:prod,service:api)
  --limit <n>              Max events per page or total with --all (default: 50)
  --all                    Auto-paginate until --limit or end of results
  --cursor <value>         Pagination cursor from a prior result
  --count-only             Return hit_count only

Exit codes:
  0  success
  2  validation / usage error
`
