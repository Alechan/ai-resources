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
	case "logs":
		printLogsHelp(w, args)
	case "metrics":
		printMetricsHelp(w, args)
	case "events":
		printEventsHelp(w, args)
	case "notebooks":
		printNotebooksHelp(w, args)
	case "init":
		fmt.Fprint(w, helpInit)
	case "doctor":
		fmt.Fprint(w, helpDoctor)
	case "monitors":
		printMonitorsHelp(w, args)
	default:
		fmt.Fprintf(w, "Unknown command %q.\n\n", cmd)
		printUsage(w)
	}
}

const helpInit = `Usage:
  ddctl init [--curl-file <path>] [--clear]

Store DataDog session cookies in the macOS Keychain.

Prefer piping a copied cURL from Chrome DevTools:
  pbpaste | ddctl init

Or read cURL from a file:
  ddctl init --curl-file ~/curl.txt

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
