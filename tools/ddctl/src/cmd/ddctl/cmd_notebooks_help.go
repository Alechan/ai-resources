package main

import (
	"fmt"
	"io"
)

func printNotebooksHelp(w io.Writer, args []string) {
	sub := firstNonHelpArg(args)
	switch sub {
	case "":
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
		fmt.Fprintf(w, "Unknown notebooks subcommand %q.\nValid choices: get, create, update, validate.\n\n", sub)
		fmt.Fprint(w, helpNotebooks)
	}
}

const helpNotebooks = `Usage:
  ddctl notebooks <subcommand> [flags]

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

Flags:
  --from-file <path>       Notebook JSON file (required)
  --name <name>            Override attributes.name
  --time <live_span>       Override attributes.time.live_span
  --skip-validate          Skip metric query preflight
  --from <time>            Preflight window start (default: now-30d)
  --to <time>              Preflight window end (default: now)
  --dry-run                Validate and print summary without POST

Mutates DataDog. Do not run unless create was requested.
`

const helpNotebooksUpdate = `Usage:
  ddctl notebooks update <id> --from-file <path> --replace-all

Flags:
  --from-file <path>                 Notebook JSON file (required)
  --replace-all                      Required; PUT is full replacement
  --skip-validate                    Skip metric query preflight
  --from <time>                      Preflight window start (default: now-30d)
  --to <time>                        Preflight window end (default: now)
  --dry-run                          Validate and print diff without PUT
  --diff                             Include semantic diff on successful update
  --if-unmodified-since <rfc3339>    Abort if remote modified_at does not match

Mutates DataDog. PUT is full replacement; --replace-all is required.
`

const helpNotebooksValidate = `Usage:
  ddctl notebooks validate --from-file <path> [--from <time>] [--to <time>]

Validate notebook structure and preflight timeseries metric queries.
Does not change Datadog.

No-data in the selected window is a warning (exit 0), not invalidity.
`
