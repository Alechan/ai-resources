# ddctl Tool Resource

`ddctl` is an unofficial DataDog CLI maintained in this repository under `tools/ddctl/src`.
It authenticates using DataDog session cookies stored in the macOS Keychain.
Most commands are read-only; notebook create/update commands perform explicit user-requested writes.

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
  monitors-list   List DataDog monitors
  monitors-get    Get a specific DataDog monitor by ID
  events-list     List DataDog events
  metrics-query   Query DataDog timeseries metrics
  notebooks       Manage DataDog notebooks (get/create/update/validate)

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

### monitors-list

List all DataDog monitors.

```bash
ddctl monitors-list
ddctl monitors-list --tag env:prod
ddctl monitors-list --json
```

### monitors-get

Fetch a specific monitor by ID.

```bash
ddctl monitors-get 12345678
ddctl monitors-get 12345678 --json
```

### events-list

List DataDog events in a time range.

```bash
ddctl events-list --from now-2h
ddctl events-list --from now-4h --tags env:prod --json
```

> **Note**: `events-list` uses `/api/v1/events` which may return HTTP 401 depending on your DataDog configuration. If this happens, report it — the browser may use a different internal endpoint.

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
ddctl notebooks validate --from-file notebook.json --from now-30d --allow-empty-series
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

## Troubleshooting

- **Credentials not found**: run `pbpaste | ddctl init` with a fresh cURL from the Logs Explorer (copy to clipboard first).
- **Auth failures (HTTP 401/403)**: your session has expired; re-run `pbpaste | ddctl init` with a fresh cURL from the Logs Explorer.
- **Parse error**: ensure the cURL command includes a `-b` or `Cookie:` header with session cookies, or use `--curl-file` if pasting fails.
- **Missing CSRF token**: the cURL must include an `-H 'x-csrf-token: ...'` header; use the Logs Explorer (not Settings) to capture it.
- **events-list returns 401**: the `/api/v1/events` endpoint may not accept session-cookie auth on your DataDog instance; report the issue.
- **Template endpoint 404**: `GET /api/v1/notebooks/template/{id}` may return 404. Clone the template in UI first, then use the cloned notebook ID.
- **Blank notebook charts**: preflight timeseries with `ddctl notebooks validate` or `ddctl metrics-query` before writing.
- **SQS metric with no data**: avoid `kube_namespace` filters on `aws.sqs.*`; scope by `queuename` tags.
- **Keychain access denied**: macOS may prompt for keychain access; accept the prompt.
- **`command not found`**: ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `PATH`.
- **Network errors**: verify connectivity to `app.datadoghq.com`; retry with `--timeout 60s`.
