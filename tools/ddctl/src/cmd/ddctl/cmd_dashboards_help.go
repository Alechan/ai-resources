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
	case "list":
		fmt.Fprint(w, helpDashboardsList)
	case "search":
		fmt.Fprint(w, helpDashboardsSearch)
	case "clone":
		fmt.Fprint(w, helpDashboardsClone)
	case "delete":
		fmt.Fprint(w, helpDashboardsDelete)
	default:
		fmt.Fprintf(w, "Unknown dashboards subcommand %q.\nValid choices: get, list, search, validate, create, update, clone, delete.\n\n", sub)
		fmt.Fprint(w, helpDashboards)
	}
}

const helpDashboards = `Usage:
  ddctl dashboards <subcommand> [flags]

Subcommands:
  get       Fetch a dashboard by ID
  list      List dashboards
  search    Search dashboards by title or tag
  validate  Check payload structure and preflight queries (no writes)
  create    Create a dashboard from JSON
  update    Replace a dashboard (requires --replace-all)
  clone     Clone a dashboard to a new title
  delete    Delete a dashboard (requires --confirm)

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
  --skip-validate                    Skip query preflight (structure still checked)
  --from <time>                      Query window start (default: now-30d)
  --to <time>                        Query window end (default: now)
  --template-variable <name=value>   Substitute $name.value during validate (repeatable)

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

const helpDashboardsList = `Usage:
  ddctl dashboards list [--limit <n>]

List dashboards from GET /api/v1/dashboard.

Note: loads the full dashboard list from Datadog; use --limit to cap output.

Flags:
  --limit <n>   Return at most n dashboards (default: all)

Output:
  Text: ID, title, URL per dashboard
  JSON (--json): {"dashboards":[...]}

Exit codes:
  0  success
  3  authentication failure
  5  API failure

Example:
  ddctl dashboards list --limit 20
`

const helpDashboardsSearch = `Usage:
  ddctl dashboards search [--title <substr>] [--tag <tag>] [--limit <n>]

Search dashboards client-side after listing all dashboards.

At least one of --title or --tag is required.

Flags:
  --title <substr>   Case-insensitive title substring
  --tag <tag>        Exact tag match
  --limit <n>        Return at most n matches (warns when truncated)

Exit codes:
  0  success
  2  missing filters or invalid usage
  5  API failure

Example:
  ddctl dashboards search --title "DELETE ME ddctl-dev"
  ddctl dashboards search --tag team:platform --limit 10
`

const helpDashboardsClone = `Usage:
  ddctl dashboards clone <id> --title <title> [flags]

Clone a dashboard by fetching the source, stripping identity fields, and creating a copy.

This mutates Datadog. Do not run unless clone was requested.

Positional arguments:
  id    Source dashboard ID

Flags:
  --title <title>                    Title for the clone (required)
  --dry-run                          Validate and print summary; do not POST
  --skip-validate                    Skip query preflight
  --from <time>                      Query window start (default: now-30d)
  --to <time>                        Query window end (default: now)
  --template-variable <name=value>   Substitute $name.value during validate (repeatable)

Exit codes:
  0  cloned, or dry-run success
  2  missing --title or invalid usage
  5  API failure

Example:
  ddctl dashboards clone cec-7ix-73w --title "DELETE ME ddctl-dev copy"
`

const helpDashboardsDelete = `Usage:
  ddctl dashboards delete <id> --confirm <id>

Delete a dashboard via DELETE /api/v1/dashboard/{id}.

This mutates Datadog. Do not run unless delete was requested.

Positional arguments:
  id    Dashboard ID to delete

Flags:
  --confirm <id>   Must exactly equal the dashboard ID

Exit codes:
  0  deleted
  2  missing or mismatched --confirm
  5  API failure

Example:
  ddctl dashboards delete abc-def-ghi --confirm abc-def-ghi
`
