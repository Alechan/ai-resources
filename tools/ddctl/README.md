# ddctl Tool Resource

`ddctl` is an unofficial DataDog CLI maintained in this repository under `tools/ddctl/src`.
It authenticates using DataDog session cookies stored in the macOS Keychain.
Most commands are read-only; notebook and dashboard create/update commands perform explicit user-requested writes.

## Quick Start

1. Install: `go install ./cmd/ddctl` (from `tools/ddctl/src`)
2. Get cURL from Chrome DevTools:
   - Log in to https://app.datadoghq.com/logs (Logs Explorer)
   - Open DevTools (Cmd+Option+I) → Network tab
   - Find a POST request to `/api/v1/logs-analytics/list`
   - Right-click → Copy → Copy as cURL
3. Initialize (recommended): `pbpaste | ddctl init`
4. Try a query: `ddctl logs-query --query "service:my-svc"`

## Build and Install

```bash
go install github.com/Alechan/ai-resources/tools/ddctl/src/cmd/ddctl@latest
```

Local contributor install from this repository:

```bash
cd tools/ddctl/src
go install ./cmd/ddctl
```

## Validate Source

```bash
cd tools/ddctl/src
go test ./...
```

## Verify

```bash
ddctl --help
```

## Workflow

### Initialization: Clipboard Pipe (Recommended)

The simplest approach: copy the cURL from DevTools directly to Keychain.

```bash
# 1. Open Chrome and log in to https://app.datadoghq.com/logs (Logs Explorer)
# 2. Open DevTools (Cmd+Option+I) → Network tab
# 3. Find a POST request to /api/v1/logs-analytics/list
# 4. Right-click → Copy → Copy as cURL
# 5. Run:
pbpaste | ddctl init
```

### Initialization: File-based (Alternative)

If you prefer to save the cURL to a file first:

```bash
ddctl init --curl-file ~/curl.txt
```

### Clear credentials

```bash
ddctl init --clear
```

## Design decisions

### Why clipboard pipe

Terminal pasting has a fundamental limitation: most terminals buffer a single pasted line to ~4096 bytes. DataDog cURL commands (especially with large cookie jars) often exceed this. By reading from macOS clipboard directly (`pbpaste`), we bypass the terminal buffer entirely—no truncation, instant parsing, and simpler UX.

If the file approach is preferred, `--curl-file` is available as an alternative.

### Why macOS Keychain, not a config file

An earlier design read cookies directly from Chrome's SQLite database. That required deriving Chrome's master AES key from macOS Keychain, which could decrypt *any* Chrome cookie (Google, banks, everything) — a much broader blast radius than needed.

The current approach stores only the DataDog session string, scoped under service `"ddctl"`. A config file (`~/.config/ddctl/session.json`) would work too, but Keychain gives OS-level access control and keeps credentials out of the filesystem where they might be swept up by backups, dotfile sync, or accidental `cat`.

## Credential Storage

Cookies are stored in the macOS Keychain under:
- **Service**: `ddctl`
- **Account**: the DataDog site domain (e.g. `datadoghq.com`)

To inspect manually: `security find-generic-password -s "ddctl" -a "datadoghq.com" -w`

To clear stored credentials: `ddctl init --clear`

## Refresh Session

When your DataDog session expires (HTTP 401/403 errors), run `ddctl init` again with fresh values from DevTools.

## Usage

```
Usage: ddctl [global flags] <command> [flags]

Commands:
  init            Store DataDog session cookies from a cURL file or stdin
  doctor          Check credentials, DataDog auth, and reachability
  logs-query      Query DataDog logs
  monitors        Manage DataDog monitors (list/get/validate/create/update/mute/unmute)
  events-list     List DataDog events
  metrics-query   Query DataDog timeseries metrics
  notebooks       Manage DataDog notebooks (get/create/update/validate)
  dashboards      Manage DataDog dashboards (get/list/search/create/update/validate/clone/delete)

Global flags:
  --site <domain>        DataDog site domain (default: datadoghq.com)
                           Env override: DDCTL_SITE
  --timeout <duration>   Timeout per command (default: 30s)
  --json                 JSON output
  --debug                Debug logging
```

### init

Store DataDog session cookies in the macOS Keychain by parsing a cURL command.

```bash
# From clipboard (recommended)
pbpaste | ddctl init

# From file
ddctl init --curl-file ~/curl.txt

# Clear stored credentials
ddctl init --clear
```

**Workflow**:
1. Open Chrome and log in to `https://app.datadoghq.com/logs` (Logs Explorer)
2. Open DevTools (Cmd+Option+I) → Network tab
3. Find a **POST** request to `/api/v1/logs-analytics/list`
4. Right-click → **Copy** → **Copy as cURL**
5. Run: `pbpaste | ddctl init` (or: `ddctl init --curl-file ~/curl.txt`)

`init` will:
1. Parse the cURL command to extract cookies and CSRF token
2. Validate required cookies are present (session + CSRF)
3. Store credentials in macOS Keychain
4. Run `ddctl doctor` to verify connectivity

### doctor

Check credentials exist, DataDog is reachable, and authentication works via a lightweight query.

```bash
ddctl doctor
ddctl doctor --json
```

`doctor` exits non-zero if auth validation fails.

### logs-query

Query DataDog logs with a search filter and time range.

```bash
ddctl logs-query --query "service:my-service status:error" --from now-1h --to now
ddctl logs-query -q "env:prod" --from now-4h --limit 100 --json

# Manual pagination: next_cursor is printed at the end of single-page results
ddctl logs-query --cursor '<next_cursor value>'

# Auto-paginate up to --limit total events
ddctl logs-query --all --limit 200

# Count-only mode (total matches, metadata-only output)
ddctl logs-query --query "service:my-service" --from now-1h --count-only --json
```

Accepted time formats: `now`, `now-1h`, `now-30m`, `now-2d`, `now-1w`, Unix milliseconds, RFC3339.

Notes:
- Output includes `hit_count` in text and JSON.
- When Datadog returns rows with `hitCount=0`, `warnings` are emitted.
- When `--all --limit` truncates results, JSON includes:
  - `truncated`
  - `returned_count`
  - `limit`
  - `hit_count` (when available)

### monitors

Manage DataDog monitors through browser-authenticated API endpoints.

```bash
# List monitors
ddctl monitors list
ddctl monitors list --tag env:prod
ddctl --json monitors list

# Get monitor (raw JSON includes options for round-trip create/update)
ddctl monitors get 12345678
ddctl --json monitors get 12345678 > monitor.json

# Validate, create, update
ddctl monitors validate --from-file monitor.json
ddctl monitors create --from-file monitor.json --dry-run
ddctl monitors update 12345678 --from-file monitor.json --replace-all

# Mute / unmute / delete
ddctl monitors mute 12345678
ddctl monitors mute 12345678 --until "2026-09-10T12:00:00Z"
ddctl monitors unmute 12345678 --confirm 12345678   # required for env:prod monitors
ddctl monitors delete 12345678 --confirm 12345678     # throwaway monitors only
```

Text list output: `[id] state type name tags:…`

Production monitors (tags `env:prod` or `env:production`) require
`--confirm <id>` on unmute.

### events-list

List DataDog events in a time range. Uses the same browser endpoint as logs-query
(`/api/v1/logs-analytics/list?type=feed`), so it works with session-cookie auth.

```bash
ddctl events-list --from now-2h
ddctl events-list --from now-4h --tags env:prod --json
ddctl events-list --from now-1h --sources containerd,kubernetes --limit 20
ddctl events-list --from now-1h --count-only --json
ddctl events-list --cursor '<next_cursor value>'
```

### metrics-query

Query DataDog timeseries metrics. Returns summary stats (min/avg/max/last) per series.

```bash
# Summary stats (default)
ddctl metrics-query --query "avg:system.cpu.user{service:my-svc}" --from now-1h

# Multiple series with grouping
ddctl metrics-query --query "sum:aws.sqs.number_of_messages_received{service:tapir} by {queuename}.as_rate()" --from now-1h

# JSON output (stats only, no pointlist)
ddctl metrics-query --query "avg:system.cpu.user{*}" --from now-4h --json

# JSON with full pointlist
ddctl metrics-query --query "avg:system.cpu.user{*}" --from now-1h --json --raw
```

### notebooks

Manage DataDog notebooks through browser-authenticated API endpoints.

```bash
# Get notebook summary (text)
ddctl notebooks get 14515133

# Get raw notebook JSON
ddctl --json notebooks get 14515133 > notebook.json

# Create notebook from file
ddctl notebooks create --from-file notebook-create.json --name "My notebook" --time 1w

# Update notebook (full replacement; explicit confirmation required)
ddctl notebooks update 14515133 --from-file notebook-update.json --replace-all

# Validate notebook payload and preflight timeseries queries
ddctl notebooks validate --from-file notebook.json --from now-30d
```

`notebooks create` and `notebooks update` accept these file shapes:

1. Full API envelope:
```json
{"data":{"type":"notebooks","attributes":{...}}}
```

2. Attributes-only envelope:
```json
{"attributes":{...}}
```

Notes:
- `update` is full replacement (`PUT`), not patch.
- `--replace-all` is mandatory for update.
- `attributes.name`, `attributes.time`, and non-empty `attributes.cells` are required.

### dashboards

Manage DataDog dashboards through browser-authenticated API endpoints.

```bash
# Get dashboard summary (text)
ddctl dashboards get bx7-nsy-pm5

# Get raw dashboard JSON
ddctl --json dashboards get bx7-nsy-pm5 > dashboard.json

# Command-specific help
ddctl dashboards --help
ddctl dashboards validate --help

# Validate payload and preflight metric/log queries
ddctl dashboards validate --from-file dashboard.json --from now-4h
ddctl dashboards validate --from-file dashboard.json --from now-4h --template-variable environment=acceptance

# Preview create/update without writing
ddctl dashboards create --from-file dashboard.json --title "DELETE ME ddctl-dev copy" --dry-run
ddctl dashboards update cec-7ix-73w --from-file dashboard.json --replace-all --dry-run

# List and search dashboards
ddctl dashboards list
ddctl dashboards list --limit 20
ddctl dashboards search --title "DELETE ME ddctl-dev"
ddctl dashboards search --tag team:platform --limit 10

# Clone and delete (mutating)
ddctl dashboards clone cec-7ix-73w --title "DELETE ME ddctl-dev copy" --dry-run
ddctl dashboards delete abc-def-ghi --confirm abc-def-ghi

# Create from a previous get (identity fields are stripped)
ddctl dashboards create --from-file dashboard.json --title "DELETE ME ddctl-dev copy"

# Update (full replacement; explicit confirmation required)
ddctl dashboards update cec-7ix-73w --from-file dashboard.json --replace-all --if-unmodified-since "2026-09-09T16:40:00Z"
```

`dashboards create` and `dashboards update` accept a raw dashboard object or
`{"dashboard":{...}}`.

Required fields: `title`, non-empty `widgets`, `layout_type` of `ordered` or `free`.

Notes:
- `update` is full replacement (`PUT`), not patch.
- `--replace-all` is mandatory for update.
- Create/update run validate unless `--skip-validate`.
- `--dry-run` on create/update/clone does not write. `--diff` prints a semantic diff.
- `--if-unmodified-since` aborts if the remote `modified_at` does not match.
- `--json` prints structured error envelopes on failure (validation includes widget/query fields when known).
- `--debug` logs HTTP method/path/status only; never prints cookies, CSRF tokens, or log bodies.
- Query preflight covers metrics (including formulas), logs (`query` / `query_string` / `search.query`), and monitor IDs.
- No-data in the selected window is a warning (exit 0), not a validation failure.
- Other data sources warn and skip.

## Troubleshooting

- **Credentials not found**: run `pbpaste | ddctl init` with a fresh cURL from the Logs Explorer (copy to clipboard first).
- **Auth failures (HTTP 401/403)**: your session has expired; re-run `pbpaste | ddctl init` with a fresh cURL from the Logs Explorer.
- **Parse error**: ensure the cURL command includes a `-b` or `Cookie:` header with session cookies, or use `--curl-file` if pasting fails.
- **Missing CSRF token**: the cURL must include an `-H 'x-csrf-token: ...'` header; use the Logs Explorer (not Settings) to capture it.
- **Template endpoint 404**: `GET /api/v1/notebooks/template/{id}` may return 404. Clone the template in UI first, then use the cloned notebook ID.
- **Blank notebook charts**: preflight timeseries with `ddctl notebooks validate` or `ddctl metrics-query` before writing.
- **SQS metric with no data**: avoid `kube_namespace` filters on `aws.sqs.*`; scope by `queuename` tags.
- **Keychain access denied**: macOS may prompt for keychain access; accept the prompt.
- **`command not found`**: ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `PATH`.
- **Network errors**: verify connectivity to `app.datadoghq.com`; retry with `--timeout 60s`.
