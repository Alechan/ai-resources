# ddctl Tool Resource

`ddctl` is an unofficial DataDog CLI maintained in this repository under `tools/ddctl/src`.
It authenticates using DataDog session cookies stored in the macOS Keychain. All operations are **read-only**.

## Quick Start

1. Install: `go install ./cmd/ddctl` (from `tools/ddctl/src`)
2. Get cookies from Chrome DevTools (see [Workflow](#workflow) below)
3. `ddctl init --cookie '<cookie string>' --csrf-token '<x-csrf-token value>'`
4. `ddctl doctor`
5. `ddctl logs-query --query "service:my-svc"`

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

1. Open Chrome and log in to `https://app.datadoghq.com/logs` (the **Logs Explorer**)
2. Open DevTools (Cmd+Option+I) → Network tab
3. Filter requests by `logs-analytics` — find a **POST** request to `/api/v1/logs-analytics/list`
4. Right-click it → Copy → **Copy as cURL**
5. Extract from the cURL:
   - The cookie string: value after `-b '...'` or `Cookie:` header
   - The CSRF token: value of `-H 'x-csrf-token: ...'`
6. Run:
   ```bash
   ddctl init --cookie '<cookie string>' --csrf-token '<x-csrf-token value>'
   ```
   Or pass the full cURL (auto-extracts both):
   ```bash
   ddctl init --curl '<full cURL command>'
   ```
7. Verify: `ddctl doctor`

> **Important**: copy from the Logs Explorer page, not settings pages. Only Logs Explorer requests carry `dd_csrf_token` which is required for `logs-query`.

## Design decisions

### Why the CLI parses the cURL, not the AI

When used through an AI skill, the user pastes a cURL command into the chat and the skill calls `ddctl init --curl '...'`. The Cookie header extraction happens inside the CLI, not in the LLM.

This is intentional:

- **Deterministic by nature.** Extracting a `Cookie:` header from a cURL string is a mechanical regex match — no ambiguity, no judgment required. That's the wrong job for an LLM.
- **Consistent and tested.** The parser has unit tests and behaves identically regardless of which model or prompt runs the skill.
- **Safe at the boundary.** Cookie values contain `=`, `;`, and quotes. An LLM parsing and re-serialising them risks silent corruption that only surfaces as a confusing 401 later.
- **Self-contained CLI.** The tool works without an AI in the loop. A human can run `ddctl init --curl '...'` directly.

The AI skill's job is to know *when* and *why* to call `ddctl init` — guiding the user through DevTools and deciding which command to run. String processing belongs in code.

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

When your DataDog session expires (HTTP 401/403 errors), repeat the workflow above:
```bash
ddctl init --curl '<new cURL from Chrome DevTools>'
```

## Usage

```
Usage: ddctl [global flags] <command> [flags]

Commands:
  init            Store DataDog session cookies from a cURL command or raw cookie string
  doctor          Check credentials, DataDog auth, and reachability
  logs-query      Query DataDog logs
  monitors-list   List DataDog monitors
  monitors-get    Get a specific DataDog monitor by ID
  events-list     List DataDog events

Global flags:
  --site <domain>        DataDog site domain (default: datadoghq.com)
                           Env override: DDCTL_SITE
  --timeout <duration>   Timeout per command (default: 30s)
  --json                 JSON output
  --debug                Debug logging
```

### init

Store DataDog session cookies in the macOS Keychain.

```bash
# From a cURL command (auto-extracts cookies + CSRF token)
ddctl init --curl 'curl "https://app.datadoghq.com/..." -b "dogweb=...; _dd_s_v2=..." -H "x-csrf-token: abc"'

# From individual values (preferred — avoids shell escaping issues)
ddctl init --cookie 'dogweb=...; _dd_s_v2=...' --csrf-token 'abc123'

# Clear stored credentials
ddctl init --clear
```

### doctor

Check that credentials exist in Keychain and DataDog is reachable.

```bash
ddctl doctor
ddctl doctor --json
```

### logs-query

Query DataDog logs with a search filter and time range.

```bash
ddctl logs-query --query "service:my-service status:error" --from now-1h --to now
ddctl logs-query -q "env:prod" --from now-4h --limit 100 --json

# Manual pagination: next_cursor is printed at the end of single-page results
ddctl logs-query --cursor '<next_cursor value>'

# Auto-paginate up to --limit total events
ddctl logs-query --all --limit 200
```

Accepted time formats: `now`, `now-1h`, `now-30m`, `now-2d`, `now-1w`, Unix milliseconds, RFC3339.

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

- **Credentials not found**: run `ddctl init --cookie '<cookie str>' --csrf-token '<csrf token>'`.
- **Auth failures (HTTP 401/403)**: your session has expired; re-run `ddctl init` with a fresh cURL from the Logs Explorer.
- **logs-query returns 401 but doctor passes**: missing CSRF token. Re-run `ddctl init` with `--csrf-token`.
- **events-list returns 401**: the `/api/v1/events` endpoint may not accept session-cookie auth on your DataDog instance; report the issue.
- **Keychain access denied**: macOS may prompt for keychain access; accept the prompt.
- **`command not found`**: ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `PATH`.
- **Network errors**: verify connectivity to `app.datadoghq.com`; retry with `--timeout 60s`.
