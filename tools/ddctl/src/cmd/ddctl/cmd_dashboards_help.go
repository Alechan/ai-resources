package main

import (
	"fmt"
	"io"
)

func printDashboardsHelp(w io.Writer, args []string) {
	sub := firstNonHelpArg(args)
	switch sub {
	case "":
		fmt.Fprint(w, helpDashboards)
	case "get":
		fmt.Fprint(w, helpDashboardsGet)
	case "validate":
		fmt.Fprint(w, helpDashboardsValidate)
	case "create":
		fmt.Fprint(w, helpDashboardsCreate)
	case "update":
		fmt.Fprint(w, helpDashboardsUpdate)
	default:
		fmt.Fprintf(w, "Unknown dashboards subcommand %q.\nValid choices: get, validate, create, update.\n\n", sub)
		fmt.Fprint(w, helpDashboards)
	}
}

const helpDashboards = `Usage:
  ddctl dashboards <subcommand> [flags]

Subcommands:
  get       Fetch a dashboard by ID
  validate  Check payload structure and preflight queries (no writes)
  create    Create a dashboard from JSON
  update    Replace a dashboard (requires --replace-all)

Use ddctl dashboards <subcommand> --help for flags, defaults, exit codes, and examples.

Global flags (all subcommands):
  --json                 Machine-readable JSON output
  --site <domain>        DataDog site (default: datadoghq.com)
  --timeout <duration>   Timeout (default: 30s)
  --debug                Debug logging (never prints cookies or CSRF tokens)

Exit codes:
  0  success (including --help)
  2  validation / usage error
  3  authentication failure
  4  network failure
  5  API failure
`

const helpDashboardsGet = `Usage:
  ddctl dashboards get <id>

Fetch the complete dashboard object from GET /api/v1/dashboard/{id}.

Positional arguments:
  id    Dashboard ID (for example bx7-nsy-pm5)

Flags:
  (none beyond global flags)

Output:
  Text: ID, title, layout, widget count, URL
  JSON (--json): raw Datadog object, with canonical url injected.
  The JSON is reusable by validate, create, and update.

Input JSON shape: not applicable (ID only).

Exit codes:
  0  success
  2  missing ID or invalid usage
  3  authentication failure
  5  API failure (unknown ID, etc.)

Example:
  ddctl --json dashboards get bx7-nsy-pm5 > dashboard.json
`

const helpDashboardsValidate = `Usage:
  ddctl dashboards validate --from-file <path> [flags]

Validate dashboard structure and execute embedded metric/log queries.
Does not change Datadog.

Flags:
  --from-file <path>                 Dashboard JSON payload (required)
  --from <time>                      Query window start (default: now-30d)
  --to <time>                        Query window end (default: now)
  --template-variable <name=value>   Substitute $name.value in queries (repeatable)
  --allow-empty-series               Accepted for compatibility; no-data is always a warning

Input JSON shape:
  A raw dashboard object from dashboards get, or {"dashboard":{...}}.
  Required fields: title, layout_type (ordered|free), non-empty widgets.

Query support:
  metrics, logs (including search.query), formulas (structural),
  monitor IDs on alert widgets. Other data sources warn and skip.

No-data in the selected window is a warning (exit 0), not invalidity.

Exit codes:
  0  structure valid (warnings allowed)
  2  malformed payload or invalid query
  3  authentication failure
  4  network failure
  5  API failure

Example:
  ddctl dashboards validate --from-file dashboard.json --from now-4h
  ddctl dashboards validate --from-file dashboard.json --template-variable environment=acceptance --from now-4h
`

const helpDashboardsCreate = `Usage:
  ddctl dashboards create --from-file <path> [flags]

Create a dashboard via POST /api/v1/dashboard.

This mutates Datadog. Do not run unless create was requested.

Flags:
  --from-file <path>                 Dashboard JSON payload (required)
  --title <title>                    Override title
  --dry-run                          Validate and print summary; do not POST
  --skip-validate                    Skip query preflight (structure still checked)
  --from <time>                      Query window start (default: now-30d)
  --to <time>                        Query window end (default: now)
  --template-variable <name=value>   Substitute $name.value during validate (repeatable)
  --allow-empty-series               Accepted for compatibility; no-data is always a warning

Input JSON shape:
  Raw dashboard object or {"dashboard":{...}}.
  Identity fields from a previous get (id, url, author_*, created_at, modified_at) are stripped.
  Other fields (widgets, layout, formulas, template_variables, notify_list) are preserved.

Exit codes:
  0  created, or dry-run success
  2  invalid payload or flags
  3  authentication failure
  5  API failure

Example:
  ddctl --json dashboards create --from-file survey-dashboard.json
  ddctl dashboards create --from-file survey-dashboard.json --dry-run
`

const helpDashboardsUpdate = `Usage:
  ddctl dashboards update <id> --from-file <path> --replace-all [flags]

Replace a dashboard via PUT /api/v1/dashboard/{id}.

This mutates Datadog. PUT is full replacement, not a patch.

Positional arguments:
  id    Dashboard ID to update

Flags:
  --from-file <path>                 Dashboard JSON payload (required)
  --replace-all                      Confirm full replacement (required)
  --dry-run                          Validate and show semantic diff; do not PUT
  --diff                             Show semantic diff against the current dashboard
  --if-unmodified-since <rfc3339>    Abort if remote modified_at does not match
  --expected-modified-at <rfc3339>   Alias of --if-unmodified-since
  --skip-validate                    Skip query preflight (structure still checked)
  --from <time>                      Query window start (default: now-30d)
  --to <time>                        Query window end (default: now)
  --template-variable <name=value>   Substitute $name.value during validate (repeatable)
  --allow-empty-series               Accepted for compatibility; no-data is always a warning

Input JSON shape:
  Same as create / get. Required: title, widgets, layout_type.

Exit codes:
  0  updated, or dry-run success
  2  missing --replace-all, concurrency mismatch, or invalid payload
  3  authentication failure
  5  API failure

Example:
  ddctl dashboards update abc-def-ghi \
    --from-file dashboard.json \
    --replace-all
  ddctl dashboards update abc-def-ghi \
    --from-file dashboard.json \
    --replace-all \
    --dry-run
  ddctl dashboards update abc-def-ghi \
    --from-file dashboard.json \
    --replace-all \
    --if-unmodified-since "2026-09-09T16:40:00Z"
`
