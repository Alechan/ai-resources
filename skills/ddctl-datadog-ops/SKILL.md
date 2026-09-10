---
name: ddctl-datadog-ops
description: Procedures for querying DataDog logs, metrics, monitors, notebooks, and dashboards using the ddctl CLI tool. Use when investigating DataDog alerts, checking service health, querying logs/metrics, or reading/updating notebooks and dashboards.
---

# ddctl-datadog-ops

## Purpose

Provide a repeatable procedure for querying DataDog logs, metrics, monitors, notebooks, and dashboards via `ddctl`.

## When To Use

- Querying logs with a search filter (service, status, environment, custom attributes).
- Checking DataDog reachability and verifying session cookie authentication.
- Iterative log investigation: narrowing down a time window or refining a query based on results.
- Confirming DataDog connectivity before beginning a deeper investigation.
- Reading or updating DataDog notebooks or dashboards through CLI automation.

## Inputs

- **cURL command from Chrome DevTools** (for init), or confirmation that `ddctl init` was already run.
- **Query string**: DataDog log search syntax (e.g. `service:my-service status:error env:prod`).
- **Time range**: relative (e.g. `now-1h`, `now-4h`) or ISO-8601 timestamps.
- **Output format**: text (default) or `--json` for structured output.
- **Site**: DataDog site domain (default: `datadoghq.com`; override with `--site` or `DDCTL_SITE`).

## Workflow

### Step 1 — One-time setup: extract cookies from Chrome

**Critical: you must copy a cURL from a POST request on the Logs Explorer page.**  
Not all requests carry the full auth cookies. GET requests (e.g. feature flags, settings pages) are
missing `dd_csrf_token` and possibly `DD_S`, which will cause HTTP 401 on logs queries.

1. Open Chrome and navigate to https://app.datadoghq.com/logs (the **Logs Explorer**).
2. Open DevTools → **Network** tab.
3. Wait for the page to load fully so that log queries fire.
4. In the Network filter box, type `logs-analytics` to find the right request.
   - Look for a **POST** to `/api/v1/logs-analytics/list?type=logs`.
   - If you see it, right-click → **Copy** → **Copy as cURL (bash)** — this is the ideal request.
   - If you can't find `logs-analytics`, any POST to `app.datadoghq.com` from the Logs Explorer page works.
5. Paste the cURL into the chat (as a code block is fine).

**Why this specific request?**
The browser's Logs Explorer sends `dd_csrf_token` in its cookie jar, which the init step needs.
This cookie is only present on pages that show interactive DataDog content — it won't appear
in requests made from settings pages or on initial page load.

**cURL format note:**
Chrome may produce either `-H 'Cookie: ...'` or `-b '...'` form — `ddctl init` handles both.
The CSRF token from the cURL body (`_authentication_token`) or cookies is merged automatically.

**Shell-escaping a complex multi-line cURL is error-prone.** Prefer piping or a file:
```
pbpaste | ddctl init
ddctl init --curl-file ~/curl.txt
```

**Why is the CSRF token needed?**
DataDog's browser UI endpoint (`/api/v1/logs-analytics/list`) validates a CSRF token sent
both as the `x-csrf-token` request header and as `_authentication_token` in the POST body.
`ddctl init` extracts it from the cURL and stores it as a synthetic `dd_csrf_token` cookie
so the API client can inject it on each request.

### Step 2 — Initialize credentials

Run in the terminal (not the chat, to avoid shell escaping issues):
```
pbpaste | ddctl init
```

Or save the cURL to a file first:
```
ddctl init --curl-file ~/curl.txt
```

The skill can tell the user to run this command; it cannot execute interactive terminal commands itself.

### Step 3 — Verify

```
ddctl doctor
```

Expected output: `credentials found: true`, `datadog reachable: true`, `auth query valid: true`.
If `datadog reachable: false` or you get HTTP 401, the cookies are expired — go back to Step 1.

### Step 4 — Query logs

```bash
ddctl logs query --query "service:<name> status:error" --from now-1h
ddctl logs query --query "service:<name>" --from now-24h --count-only --json
ddctl logs query --all --limit 200 --json

# Forensics projection (JSON)
ddctl logs query -q 'service:beaver status:error' \
  --fields timestamp,msg,body,status_code,url,email,payload --json

# Verbose text mode
ddctl logs query -q 'service:beaver status:error' --verbose

# Export incident bundle
ddctl logs export -q 'service:beaver *ORDER*' \
  --from '<start>' --to '<end>' -o datadog-logs.ndjson

# Fetch one event by ID
ddctl logs get <event_id> --from '<start>' --to '<end>' --json
```

JSON events are flat objects with a full `custom` map. Use `--fields` for targeted
forensics (`body`, `status_code`, `url`, `email`, `payload`, `error`, `msg`).

Supported `--from`/`--to` formats: `now`, `now-1h`, `now-30m`, `now-2d`, `now-1w`, Unix milliseconds, RFC3339.

### Step 4.1 — Field discovery workflow (mandatory)

1. Start with `--count-only` to verify there is data before sampling rows.
2. Run a narrow query with `--json` and inspect `custom` keys (`msg`, `body`, `payload`, `error`).
3. Use `--fields` to project only the keys needed for root-cause analysis.
4. Export with `logs export` when you need the full incident timeline offline.
5. Treat `hit_count` as the source of truth for matching volume.
6. If `hit_count=0` but rows are returned, treat rows as housekeeping/noise unless proven otherwise.

### Step 4.2 — Mytheresa log field conventions

Common Beaver/custom fields inside `event.custom`:

| Field | Use |
|-------|-----|
| `msg` | Human-readable line (preferred for `message`) |
| `body` | HTTP response body (Dixa client) |
| `method`, `url`, `status_code` | HTTP client debug |
| `payload` | Business object (NPS message, survey request) |
| `email`, `error`, `reason` | Domain identifiers and failures |

Kubernetes tags (`kube_namespace`, `pod_name`) are queryable in search syntax but may not appear in `custom`.

### Step 5 — Monitor operations

```bash
ddctl monitors list
ddctl monitors list --tag env:prod
ddctl monitors get <monitor-id>
ddctl --json monitors get <monitor-id> > monitor.json
ddctl monitors validate --from-file monitor.json
ddctl monitors create --from-file monitor.json --dry-run
ddctl monitors create --from-file monitor.json --muted   # atomic global mute on create
ddctl monitors update <id> --from-file monitor.json --replace-all
ddctl monitors mute <id> [--until <rfc3339>]
ddctl monitors unmute <id> [--confirm <id>]   # confirm required for env:prod
ddctl monitors delete <id> --confirm <id>     # throwaway monitors only
```

Output format (text list): `[id] state type name tags:…`

For production monitor rollouts, prefer `monitors create --muted` so Datadog receives
`options.silenced["*"]` in the initial create request. Do not rely on a separate
mute step after create. When updating a muted monitor, export with `monitors get --json`;
ddctl preserves mute on update unless you change `options.silenced` in the file.

### Step 6 — List events

```
ddctl events list --from now-2h
ddctl events list --from now-4h --tags env:prod
ddctl events list --from now-1h --sources containerd,kubernetes --limit 20
ddctl events list --from now-1h --count-only --json
ddctl events list --from now-1h --all --limit 200
ddctl events list --cursor '<next_cursor value>'
```

Uses the same browser endpoint as `ddctl logs query` (`/api/v1/logs-analytics/list?type=feed`),
so it works with session-cookie auth. Supports `--limit`, `--cursor`, and `--count-only`.

### Step 7 — Query metrics

```
ddctl metrics query --query "avg:system.cpu.user{service:<name>}" --from now-1h
ddctl metrics query --query "sum:aws.sqs.number_of_messages_received{service:<name>} by {queuename}.as_rate()" --from now-1h
ddctl metrics query --query "<query>" --from now-4h --json
ddctl metrics query --query "<query>" --from now-4h --json --raw   # includes full pointlist
```

The query syntax is standard DataDog metrics query syntax:
- `avg:`, `sum:`, `max:`, `min:`, `count:` aggregators
- `{<tag>:<value>}` filter, `by {<tag>}` grouping
- `.as_count()`, `.as_rate()`, `.fill(last)` rollup functions

Text output shows per-series summary stats: `min`, `avg`, `max`, `last`, point count, interval.

### Step 8 — Iterate and summarize

Refine the query based on results; summarize findings with timestamps, services, and log lines.

### Step 9 — Notebook operations (optional)

Use notebook commands when the task requires shareable incident writeups or reproducible dashboard notes.

```bash
# Read notebook
ddctl notebooks get <id>
ddctl --json notebooks get <id> > notebook.json

# Validate notebook payload (timeseries preflight)
ddctl notebooks validate --from-file notebook.json --from now-30d

# JSON output: concise summary (default) or full API envelope
ddctl --json notebooks get <id>
ddctl --json notebooks get <id> --raw > notebook-full.json
ddctl --json notebooks create --from-file notebook-create.json --dry-run

# Create notebook (preflight runs by default; use --dry-run first)
ddctl notebooks create --from-file notebook-create.json --name "Incident notebook" --time 1w --dry-run
ddctl notebooks create --from-file notebook-create.json --name "Incident notebook" --time 1w

# Update notebook (full replacement; preflight + optional safety flags)
ddctl notebooks update <id> --from-file notebook-update.json --replace-all --dry-run
ddctl notebooks update <id> --from-file notebook-update.json --replace-all --if-unmodified-since <rfc3339>
```

Notebook update caveats:
- `PUT` is full replacement, not patch.
- `--replace-all` is required.
- Create/update run metric preflight unless `--skip-validate`.
- `--dry-run` validates and prints diff without POST/PUT.
- `attributes.name`, `attributes.time`, and non-empty `attributes.cells` must be present.
- Structure validation covers text and chart cell types; timeseries accepts `q` or `queries[]`/`formulas[]`.
- Template variables (`$name` / `$name.value`) resolve to `defaults[0]` for preflight.
- Visible timeseries aliases use `formulas[].alias`; `metadata[].alias_name` on legacy `q` fails validation.
- `notebooks get` exposes `modified_at`; update requires a revision guard unless `--force`.
- `--json` returns a concise summary by default; use `--raw` for the full Datadog response.
- `GET /api/v1/notebooks/template/{id}` may return 404; clone template in UI first, then operate on the cloned notebook ID.

Timeseries query caveats:
- Validate queries before write to avoid blank charts.
- `aws.sqs.*` metrics are typically scoped by queue tags (`queuename`), not `kube_namespace`.
- Strict `pod_name` prefixes can go stale; prefer stable service/namespace metrics where possible.

### Step 10 — Dashboard operations (optional)

Use dashboard commands to export, validate, and write Datadog dashboards.

```bash
ddctl dashboards get <id>
ddctl --json dashboards get <id> > dashboard.json
ddctl dashboards list
ddctl dashboards search --title "<substr>"
ddctl dashboards validate --from-file dashboard.json --from now-4h
ddctl dashboards validate --from-file dashboard.json --template-variable environment=acceptance --from now-4h
ddctl dashboards create --from-file dashboard.json --title "Copy" --dry-run
ddctl dashboards create --from-file dashboard.json --title "Copy"
ddctl dashboards clone <id> --title "Copy" --dry-run
ddctl dashboards update <id> --from-file dashboard.json --replace-all --dry-run
ddctl dashboards update <id> --from-file dashboard.json --replace-all --if-unmodified-since "<modified_at>"
ddctl dashboards delete <id> --confirm <id>
```

Dashboard caveats:
- `PUT` is full replacement; `--replace-all` is required.
- Create/update run validate unless `--skip-validate`.
- Query preflight covers metrics, logs (`search.query` included), formulas, and monitor IDs; other widget data sources warn and skip.
- No-data in the selected window is a warning (exit 0).
- `--if-unmodified-since` is a client-side concurrency check; Datadog has no If-Match.
- `ddctl dashboards --help` and `ddctl dashboards <subcommand> --help` print command-specific usage.

## Validation

- `ddctl doctor` shows `credentials found: true`, `datadog reachable: true`, and `auth query valid: true`.
- `ddctl logs query --query "*" --limit 1` returns at least one log event or empty result without error.
- `ddctl logs query --count-only --query "*" --from now-1h --json` returns metadata with `hit_count`.
- `ddctl logs query --json` returns flat events with `custom.body`, `custom.payload`, and related forensics fields.
- `ddctl logs export -o <path>` writes NDJSON/CSV and prints an stderr summary.
- `ddctl monitors list` returns a list of monitors (even if empty).
- `ddctl monitors get <id>` returns raw monitor JSON with `options` when present.
- `ddctl events list --from now-2h` returns events or empty without error.
- `ddctl metrics query --query "avg:system.cpu.user{*}" --from now-1h` returns series or "no data".
- `ddctl notebooks get <id>` returns notebook details without error.
- `ddctl notebooks validate --from-file <file>` reports timeseries queries; no-data is a warning.
- `ddctl dashboards get <id>` returns dashboard details without error.
- `ddctl dashboards validate --from-file <file>` reports metric/log/monitor results; no-data is a warning.
- `ddctl dashboards validate --help` prints command-specific usage and exits 0.

## Known obstacles and workarounds

### HTTP 401 from logs query even after a successful doctor

`ddctl` relies on browser session cookies and CSRF token. If auth is stale server-side, queries fail.

**Fix:** Re-run `ddctl init` using a cURL from the Logs Explorer page, which always has `x-csrf-token`.
Make sure to pass `--csrf-token` (or use `--curl` which extracts it automatically).

### `hit_count` vs returned rows mismatch

When `hit_count=0` but rows are returned, those rows are often housekeeping/retention records and not true matches.

**Fix:** treat `hit_count` as authoritative for query matching, then refine query/time window and sample again.

## Incident templates

### Endpoint-focused investigation

```bash
ddctl logs query --query 'service:<svc> kube_namespace:<env> @request.endpoint:"<endpoint>"' --from now-2h --count-only --json
ddctl logs query --query 'service:<svc> kube_namespace:<env> @request.endpoint:"<endpoint>"' --from now-2h --all --limit 200 --json
```

### Identity/email-focused investigation

```bash
ddctl logs query --query 'service:<svc> kube_namespace:<env> *<email-or-id>*' --from now-24h --count-only --json
ddctl logs query --query 'service:<svc> kube_namespace:<env> *<email-or-id>*' --from now-24h --all --limit 200 --json
```

### Error-vs-throughput investigation

```bash
ddctl logs query --query 'service:<svc> kube_namespace:<env> status:error' --from now-2h --count-only --json
ddctl logs query --query 'service:<svc> kube_namespace:<env> <throughput-signal-query>' --from now-2h --count-only --json
```

### Chrome HAR exports strip cookies (do not use HAR files for init)

Chrome's "Save all as HAR" redacts all `Cookie:` headers from the export for privacy.
Even if you export a HAR file with dozens of requests, all cookie fields will be empty.

**Only "Copy as cURL" from a single request preserves the cookie string.**

### ddctl init hangs in the terminal

If you run `ddctl init` with no stdin and no `--curl-file`, it prints setup instructions and exits.
If you paste a cURL directly on the command line and the shell sees unbalanced quotes, the shell
may hang waiting for a closing quote.

**Workaround:** pipe from the clipboard or use a file:
```
pbpaste | ddctl init
ddctl init --curl-file ~/curl.txt
```

### Ideal request to copy is /api/v1/logs-analytics/list

This is the exact endpoint `ddctl logs query` calls. Copying a cURL from this request guarantees:
- All required cookies are present (`dd_csrf_token`, `DD_S`, `dogweb`, `_dd_s_v2`, etc.)
- The CSRF token is visible in the request body as `_authentication_token` (for debugging)
- The cookie string is confirmed to be fresh and working

## Safety

- Most `ddctl` commands are read-only. Notebook and dashboard `create`/`update` commands mutate DataDog resources.
- Do not run notebook or dashboard mutation commands unless the user explicitly asked for create/update.
- For updates, prefer: get → edit file → validate → update with `--replace-all` (dashboards: `--dry-run` first).
- Stop and report to the user if DataDog returns authentication errors (HTTP 401/403).
- Do not store or log raw cookie values.
- Do not use `ddctl` to access DataDog data outside the scope authorized for the current session.
- Cookie strings contain full session credentials — do not paste them into public channels.

## References

- `tools/ddctl/README.md`
- `tools/ddctl/DASHBOARDS_SPEC.md`
- `docs/CONVENTIONS.md`
