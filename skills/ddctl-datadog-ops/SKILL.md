---
name: ddctl-datadog-ops
description: Procedures for querying DataDog logs, metrics, monitors, and notebooks using the ddctl CLI tool. Use when investigating DataDog alerts, checking service health, querying logs/metrics, or reading/updating notebooks.
---

# ddctl-datadog-ops

## Purpose

Provide a repeatable procedure for querying DataDog logs, metrics, monitors, and notebooks via `ddctl`.

## When To Use

- Querying logs with a search filter (service, status, environment, custom attributes).
- Checking DataDog reachability and verifying session cookie authentication.
- Iterative log investigation: narrowing down a time window or refining a query based on results.
- Confirming DataDog connectivity before beginning a deeper investigation.
- Reading or updating DataDog notebooks through CLI automation.

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
The `x-csrf-token` header is extracted automatically from the cURL when using `--curl`.

However, **shell-escaping a complex multi-line cURL** is error-prone. The reliable path:
extract the cookie string and CSRF token from the cURL yourself and pass them as:
```
ddctl init --cookie '<cookie_string>' --csrf-token '<x-csrf-token value>'
```
To get the cookie string: find the `-b '...'` or `Cookie:` value in the cURL.
To get the CSRF token: find the `-H 'x-csrf-token: ...'` line in the cURL.

**Why is the CSRF token needed?**
DataDog's browser UI endpoint (`/api/v1/logs-analytics/list`) validates a CSRF token sent
both as the `x-csrf-token` request header and as `_authentication_token` in the POST body.
This token is NOT a cookie — it lives in a `x-csrf-token` header in the request.
`ddctl init` stores it as a synthetic `dd_csrf_token` cookie so the API client can inject it.

### Step 2 — Initialize credentials

Run in the terminal (not the chat, to avoid shell escaping issues):
```
ddctl init --cookie '<cookie_string>'
```

Or, if the cURL shell-escaping works:
```
ddctl init --curl '<full cURL command>'
```

The skill can tell the user to run this command; it cannot execute interactive terminal commands itself.

### Step 3 — Verify

```
ddctl doctor
```

Expected output: `credentials found: true`, `datadog reachable: true`, `auth query valid: true`.
If `datadog reachable: false` or you get HTTP 401, the cookies are expired — go back to Step 1.

### Step 4 — Query logs

```
ddctl logs-query --query "service:<name> status:error" --from now-1h
ddctl logs-query --query "*" --from now-4h --limit 50 --json

# Count first for large windows
ddctl logs-query --query "service:<name>" --from now-24h --count-only --json

# Single-page result shows cursor hint if more pages exist:
# next_cursor: Aw...
# Use it:
ddctl logs-query --cursor '<next_cursor value>'

# Auto-paginate (collects up to --limit total events across pages):
ddctl logs-query --all --limit 200
```

Supported `--from`/`--to` formats: `now`, `now-1h`, `now-30m`, `now-2d`, `now-1w`, Unix milliseconds, RFC3339.

### Step 4.1 — Field discovery workflow (mandatory)

1. Start with `--count-only` to verify there is data before sampling rows.
2. Run a narrow query and inspect returned fields (`timestamp`, `status`, `service`, `host`, `message`).
3. If needed fields are missing, assume they are not queryable from this endpoint and move to Dashboard/raw logs.
4. Treat `hit_count` as the source of truth for matching volume.
5. If `hit_count=0` but rows are returned, treat rows as housekeeping/noise unless proven otherwise.

### Step 4.2 — Logged fields vs queryable fields

- Application code may log structured fields (e.g. `log.WithField("panic_stacktrace", ...)`).
- DataDog can store them, but `ddctl logs-query` may not expose them as queryable/returned fields.
- Practical rule:
  1. query with standard fields first,
  2. then use Dashboard/raw logs for deep structured payload inspection.

### Step 5 — List monitors

```
ddctl monitors-list
ddctl monitors-list --tag env:prod
ddctl monitors-get <monitor-id>
```

Output format (text): `[id] state    type     name    tags:…`

### Step 6 — List events

```
ddctl events-list --from now-2h
ddctl events-list --from now-4h --tags env:prod
```

> **Note**: `events-list` uses `/api/v1/events`. If this returns HTTP 401, the DataDog
> instance may require a different endpoint for events. Report the error and we will investigate.

### Step 7 — Query metrics

```
ddctl metrics-query --query "avg:system.cpu.user{service:<name>}" --from now-1h
ddctl metrics-query --query "sum:aws.sqs.number_of_messages_received{service:<name>} by {queuename}.as_rate()" --from now-1h
ddctl metrics-query --query "<query>" --from now-4h --json
ddctl metrics-query --query "<query>" --from now-4h --json --raw   # includes full pointlist
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

# Create notebook
ddctl notebooks create --from-file notebook-create.json --name "Incident notebook" --time 1w

# Update notebook (full replacement)
ddctl notebooks update <id> --from-file notebook-update.json --replace-all
```

Notebook update caveats:
- `PUT` is full replacement, not patch.
- `--replace-all` is required.
- `attributes.name`, `attributes.time`, and non-empty `attributes.cells` must be present.
- `GET /api/v1/notebooks/template/{id}` may return 404; clone template in UI first, then operate on the cloned notebook ID.

Timeseries query caveats:
- Validate queries before write to avoid blank charts.
- `aws.sqs.*` metrics are typically scoped by queue tags (`queuename`), not `kube_namespace`.
- Strict `pod_name` prefixes can go stale; prefer stable service/namespace metrics where possible.

## Validation

- `ddctl doctor` shows `credentials found: true`, `datadog reachable: true`, and `auth query valid: true`.
- `ddctl logs-query --query "*" --limit 1` returns at least one log event or empty result without error.
- `ddctl logs-query --count-only --query "*" --from now-1h --json` returns metadata with `hit_count`.
- `ddctl monitors-list` returns a list of monitors (even if empty).
- `ddctl events-list --from now-2h` returns events or empty; HTTP 401 = endpoint needs investigation.
- `ddctl metrics-query --query "avg:system.cpu.user{*}" --from now-1h` returns series or "no data".
- `ddctl notebooks get <id>` returns notebook details without error.
- `ddctl notebooks validate --from-file <file>` reports timeseries queries and catches empty-series risks.

## Known obstacles and workarounds

### HTTP 401 from logs-query even after a successful doctor

`ddctl` relies on browser session cookies and CSRF token. If auth is stale server-side, queries fail.

**Fix:** Re-run `ddctl init` using a cURL from the Logs Explorer page, which always has `x-csrf-token`.
Make sure to pass `--csrf-token` (or use `--curl` which extracts it automatically).

### `hit_count` vs returned rows mismatch

When `hit_count=0` but rows are returned, those rows are often housekeeping/retention records and not true matches.

**Fix:** treat `hit_count` as authoritative for query matching, then refine query/time window and sample again.

## Incident templates

### Endpoint-focused investigation

```bash
ddctl logs-query --query 'service:<svc> kube_namespace:<env> @request.endpoint:"<endpoint>"' --from now-2h --count-only --json
ddctl logs-query --query 'service:<svc> kube_namespace:<env> @request.endpoint:"<endpoint>"' --from now-2h --all --limit 200 --json
```

### Identity/email-focused investigation

```bash
ddctl logs-query --query 'service:<svc> kube_namespace:<env> *<email-or-id>*' --from now-24h --count-only --json
ddctl logs-query --query 'service:<svc> kube_namespace:<env> *<email-or-id>*' --from now-24h --all --limit 200 --json
```

### Error-vs-throughput investigation

```bash
ddctl logs-query --query 'service:<svc> kube_namespace:<env> status:error' --from now-2h --count-only --json
ddctl logs-query --query 'service:<svc> kube_namespace:<env> <throughput-signal-query>' --from now-2h --count-only --json
```

### Chrome HAR exports strip cookies (do not use HAR files for init)

Chrome's "Save all as HAR" redacts all `Cookie:` headers from the export for privacy.
Even if you export a HAR file with dozens of requests, all cookie fields will be empty.

**Only "Copy as cURL" from a single request preserves the cookie string.**

### ddctl init --curl hangs in the terminal

If the pasted cURL contains single quotes inside a single-quoted shell argument, the shell
treats the command as incomplete and hangs.

**Workaround:** Extract the cookie string from the cURL and pass it directly:
```
ddctl init --cookie '<extracted_cookie_string>'
```

### Ideal request to copy is /api/v1/logs-analytics/list

This is the exact endpoint `ddctl logs-query` calls. Copying a cURL from this request guarantees:
- All required cookies are present (`dd_csrf_token`, `DD_S`, `dogweb`, `_dd_s_v2`, etc.)
- The CSRF token is visible in the request body as `_authentication_token` (for debugging)
- The cookie string is confirmed to be fresh and working

## Safety

- Most `ddctl` commands are read-only. Notebook `create`/`update` commands mutate DataDog notebooks.
- Do not run notebook mutation commands unless the user explicitly asked for notebook creation/update.
- For updates, prefer: get → edit file → validate → update with `--replace-all`.
- Stop and report to the user if DataDog returns authentication errors (HTTP 401/403).
- Do not store or log raw cookie values.
- Do not use `ddctl` to access DataDog data outside the scope authorized for the current session.
- Cookie strings contain full session credentials — do not paste them into public channels.

## References

- `tools/ddctl/README.md`
- `docs/CONVENTIONS.md`
