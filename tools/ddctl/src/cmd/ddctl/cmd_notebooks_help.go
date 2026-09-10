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
  ddctl notebooks get <id> [--include-metadata] [--raw]

Positional arguments:
  id    Notebook ID

Flags:
  --include-metadata   Include metadata in the API request (default: true)
  --raw                With --json, emit the full Datadog response (default: concise summary)
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
  --raw                    With --json, emit the full Datadog response (default: concise summary)

Preflight resolves template variables ($name or $name.value) using defaults[0].
Timeseries requests may use legacy "q" or structured queries[] entries.

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
  --raw                              With --json, emit the full Datadog response (default: concise summary)

Mutates DataDog. PUT is full replacement; --replace-all is required.
`

const helpNotebooksValidate = `Usage:
  ddctl notebooks validate --from-file <path> [--from <time>] [--to <time>]

Validate notebook structure and preflight chart metric queries.
Does not change Datadog.

Structure checks cover all supported cell types (text and chart definitions).
Timeseries requests accept legacy "q" or structured queries[] entries.
Template variables ($name or $name.value) resolve to defaults[0] for preflight.

No-data in the selected window is a warning (exit 0), not invalidity.
`
