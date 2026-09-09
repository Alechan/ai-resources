package main

import (
	"fmt"
	"io"
)

func printMonitorsHelp(w io.Writer, args []string) {
	sub := firstNonHelpArg(args)
	switch sub {
	case "":
		fmt.Fprint(w, helpMonitors)
	case "list":
		fmt.Fprint(w, helpMonitorsList)
	case "get":
		fmt.Fprint(w, helpMonitorsGet)
	case "validate":
		fmt.Fprint(w, helpMonitorsValidate)
	case "create":
		fmt.Fprint(w, helpMonitorsCreate)
	case "update":
		fmt.Fprint(w, helpMonitorsUpdate)
	case "mute":
		fmt.Fprint(w, helpMonitorsMute)
	case "unmute":
		fmt.Fprint(w, helpMonitorsUnmute)
	case "delete":
		fmt.Fprint(w, helpMonitorsDelete)
	default:
		fmt.Fprintf(w, "Unknown monitors subcommand %q.\nValid choices: list, get, validate, create, update, mute, unmute, delete.\n\n", sub)
		fmt.Fprint(w, helpMonitors)
	}
}

const helpMonitors = `Usage:
  ddctl monitors <subcommand> [flags]

Subcommands:
  list      List monitors
  get       Fetch a monitor by numeric ID
  validate  Check payload structure and preflight query (no writes)
  create    Create a monitor from JSON
  update    Replace a monitor (requires --replace-all)
  mute      Mute a monitor
  unmute    Unmute a monitor
  delete    Delete a monitor (requires --confirm)

Use ddctl monitors <subcommand> --help for flags, defaults, exit codes, and examples.

Global flags (all subcommands):
  --json                 Machine-readable JSON output
  --site <domain>        DataDog site (default: datadoghq.com)
  --timeout <duration>   Timeout (default: 30s)
  --debug                Debug logging (never prints cookies or CSRF tokens)
`

const helpMonitorsList = `Usage:
  ddctl monitors list [--tag <tag>]

List DataDog monitors with pagination.

Note: --tag filters client-side after loading monitors from Datadog.
Large accounts may be slow.

Flags:
  --tag <tag>   Filter monitors that include this exact tag

Exit codes:
  0  success
  2  validation / usage error
  5  API failure
`

const helpMonitorsGet = `Usage:
  ddctl monitors get <id>

Fetch a monitor by numeric ID.

Positional arguments:
  id    Monitor ID

Output:
  Text: ID, name, type, state, URL, query
  JSON (--json): raw Datadog object with canonical url injected.

Exit codes:
  0  success
  2  missing or invalid ID
  5  API failure
`

const helpMonitorsValidate = `Usage:
  ddctl monitors validate --from-file <path> [--from <time>] [--to <time>]

Validate monitor structure and preflight the metric query.

Flags:
  --from-file <path>   Monitor JSON payload (required)
  --from <time>        Query window start (default: now-24h)
  --to <time>          Query window end (default: now)

No-data in the selected window is a warning (exit 0), not invalidity.

Exit codes:
  0  structure valid (warnings allowed)
  2  malformed payload or invalid query
  5  API failure
`

const helpMonitorsCreate = `Usage:
  ddctl monitors create --from-file <path> [flags]

Create a monitor via POST /api/v1/monitor.

This mutates Datadog. Do not run unless create was requested.

Flags:
  --from-file <path>     Monitor JSON payload (required)
  --dry-run              Validate and print summary; do not POST
  --skip-validate        Skip validation before create
  --muted                Mute monitor after create
  --from <time>          Query window start (default: now-24h)
  --to <time>            Query window end (default: now)

Exit codes:
  0  created, or dry-run success
  2  invalid payload or flags
  5  API failure
`

const helpMonitorsUpdate = `Usage:
  ddctl monitors update <id> --from-file <path> --replace-all [flags]

Replace a monitor via PUT /api/v1/monitor/{id}.

This mutates Datadog. PUT is full replacement, not a patch.

Flags:
  --from-file <path>                 Monitor JSON payload (required)
  --replace-all                      Confirm full replacement (required)
  --dry-run                          Show semantic diff; do not PUT
  --diff                             Show semantic diff against current monitor
  --if-unmodified-since <rfc3339>    Abort if remote modified timestamp does not match
  --skip-validate                    Skip validation before update
  --from <time>                      Query window start (default: now-24h)
  --to <time>                        Query window end (default: now)

Exit codes:
  0  updated, or dry-run success
  2  missing --replace-all or invalid payload
  5  API failure
`

const helpMonitorsMute = `Usage:
  ddctl monitors mute <id> [--until <rfc3339>]

Mute a monitor. Omit --until for an indefinite mute.

Flags:
  --until <rfc3339>   Mute until timestamp (RFC3339)

Exit codes:
  0  muted
  2  invalid usage
  5  API failure
`

const helpMonitorsUnmute = `Usage:
  ddctl monitors unmute <id> [--confirm <id>]

Unmute a monitor.

Production monitors (tags env:prod or env:production) require --confirm equal to the monitor ID.

Flags:
  --confirm <id>   Required for production monitors; must equal monitor ID

Exit codes:
  0  unmuted
  2  missing --confirm on production monitor
  5  API failure
`

const helpMonitorsDelete = `Usage:
  ddctl monitors delete <id> --confirm <id>

Delete a monitor via DELETE /api/v1/monitor/{id}.

This mutates Datadog. Do not run unless delete was requested.

Positional arguments:
  id    Monitor ID to delete

Flags:
  --confirm <id>   Must exactly equal the monitor ID

Exit codes:
  0  deleted
  2  missing or mismatched --confirm
  5  API failure

Example:
  ddctl monitors delete 12345678 --confirm 12345678
`
