package main

import (
	"fmt"
	"io"
)

func printMetricsHelp(w io.Writer, args []string) {
	sub := firstNonHelpArg(args)
	switch sub {
	case "":
		fmt.Fprint(w, helpMetrics)
	case "query":
		fmt.Fprint(w, helpMetricsQuery)
	default:
		fmt.Fprintf(w, "Unknown metrics subcommand %q.\nValid choices: query.\n\n", sub)
		fmt.Fprint(w, helpMetrics)
	}
}

const helpMetrics = `Usage:
  ddctl metrics <subcommand> [flags]

Subcommands:
  query     Query DataDog timeseries metrics

Use ddctl metrics query --help for flags and examples.
`

const helpMetricsQuery = `Usage:
  ddctl metrics query --query <query> [--from <time>] [--to <time>]

Flags:
  --query, -q <query>      Metrics query (required)
  --from <time>            Start time (default: now-1h)
  --to <time>              End time (default: now)
  --raw                    Include full pointlist in JSON output

Exit codes:
  0  success
  2  validation / usage error
`
