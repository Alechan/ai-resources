# ddctl Dashboard Commands Spec (v1)

## Goal

Provide safe CLI commands to read and write DataDog dashboards through
browser-session auth, while preventing common destructive mistakes
(especially partial `PUT` replacement and unvalidated queries).

## Commands

`ddctl dashboards <subcommand> [flags]`

### 1) `ddctl dashboards get <id>`

Fetch dashboard JSON from:

`GET /api/v1/dashboard/{id}`

Behavior:
- Text output: id, title, layout_type, widget count, URL.
- JSON output: raw API object. `url` is always
  `https://app.{site}/dashboard/{id}`.

Dashboard IDs are strings (for example `cec-7ix-73w`).

### 2) `ddctl dashboards validate --from-file <path> [--from <time>] [--to <time>] [--allow-empty-series]`

Local schema checks:
- `title` non-empty
- `layout_type` is `ordered` or `free`
- `widgets` is a non-empty array
- each widget has `definition.type`
- group widgets recursively require non-empty `definition.widgets`

Online query preflight (best effort):
- Walk widgets including group children.
- Metrics queries (`requests[].q` and `requests[].queries[]` with
  metrics data_source) run through `metrics-query`.
- Log queries (`log_stream.query`, logs data_source, `query_string`)
  run through `logs-query --count-only`.
- Empty metric series or `hit_count=0` fails unless
  `--allow-empty-series` (then warning).
- APM, RUM, SLO, formulas-as-data-source, cloud cost: warn and skip.

### 3) `ddctl dashboards create --from-file <path> [--title <title>] [--skip-validate] [--allow-empty-series]`

Create via:

`POST /api/v1/dashboard`

Accepted input file shapes:
1. Raw dashboard object (`title`, `widgets`, `layout_type`, ...)
2. `{"dashboard":{...}}`

Normalization:
- Apply optional `--title`.
- Strip identity fields from a previous get: `id`, `url`,
  `author_handle`, `author_name`, `created_at`, `modified_at`.
- Run validate unless `--skip-validate`.

### 4) `ddctl dashboards update <id> --from-file <path> --replace-all [--dry-run] [--expected-modified-at <rfc3339>] [--skip-validate]`

Update via:

`PUT /api/v1/dashboard/{id}`

Safety:
- Requires `--replace-all` explicitly (PUT is full replacement).
- `--dry-run` GETs current, prints a JSON diff, and does not PUT.
- `--expected-modified-at` GETs current and aborts if `modified_at`
  does not match (Datadog has no If-Match).
- Run validate unless `--skip-validate`.

## Exit behavior

- `0`: success
- validation error: malformed file/flags/payload or empty series
- API/auth/network errors use existing ddctl error mapping
