# ddctl Dashboard Commands Spec (v1)

## Goal

Provide safe CLI commands to read and write DataDog dashboards through
browser-session auth, while preventing common destructive mistakes
(especially partial `PUT` replacement and unvalidated queries).

## Commands

`ddctl dashboards <subcommand> [flags]`

`ddctl dashboards --help` and `ddctl dashboards <subcommand> --help`
print command-specific usage (not the global help page) and exit 0.

### 1) `ddctl dashboards get <id>`

Fetch dashboard JSON from:

`GET /api/v1/dashboard/{id}`

Behavior:
- Text output: id, title, layout_type, widget count, URL.
- JSON output: raw API object. `url` is always
  `https://app.{site}/dashboard/{id}`.
- The JSON is reusable by `validate`, `create`, and `update`.

Dashboard IDs are strings (for example `bx7-nsy-pm5`).

### 2) `ddctl dashboards validate --from-file <path> [--from <time>] [--to <time>] [--template-variable name=value]`

Local schema checks:
- `title` non-empty
- `layout_type` is `ordered` or `free`
- `widgets` is a non-empty array
- each widget has `definition.type`
- group widgets recursively require `definition.widgets`

Online query preflight (best effort):
- Walk widgets including group children.
- Metrics: `requests[].q` and `requests[].queries[]` with metrics
  data_source.
- Logs: `query`, `query_string`, and `search.query` (the Datadog
  formula/query widget shape). Empty log search is treated as `*`.
- Formulas are kept with their queries; they are not a separate
  data source.
- Monitor IDs: `alert_id` / `monitor_id` on alert widgets via
  `GET /api/v1/monitor/{id}`.
- `--template-variable name=value` (repeatable) substitutes
  `$name.value` then `$name` before preflight.
- APM, RUM, SLO, issue_stream, cloud cost: warn and skip.
- Empty metric series or log `hit_count=0` is a **warning**
  (exit 0), not invalidity. `--allow-empty-series` is accepted
  for compatibility.

Text summary:

```text
dashboard structure: valid
metric queries: 14 valid, 4 no-data
log queries: 0 valid, 1 no-data
monitor references: 0 valid
warnings:
  - widget "API 5xx Error %": no data in selected window
```

### 3) `ddctl dashboards create --from-file <path> [--title <title>] [--dry-run] [--skip-validate]`

Create via:

`POST /api/v1/dashboard`

Accepted input file shapes:
1. Raw dashboard object (`title`, `widgets`, `layout_type`, ...)
2. `{"dashboard":{...}}`

Normalization:
- Apply optional `--title`.
- Strip identity fields from a previous get: `id`, `url`,
  `author_handle`, `author_name`, `created_at`, `modified_at`.
- Other fields are preserved (widgets, layout, formulas,
  template_variables, notify_list).
- Run validate unless `--skip-validate`.
- `--dry-run` validates and prints a summary; it does not POST.

### 4) `ddctl dashboards update <id> --from-file <path> --replace-all [--dry-run] [--diff] [--if-unmodified-since <rfc3339>]`

Update via:

`PUT /api/v1/dashboard/{id}`

Safety:
- Requires `--replace-all` explicitly (PUT is full replacement).
- `--dry-run` GETs current, prints a semantic diff, and does not PUT.
- `--diff` prints the same semantic diff (and still PUTs unless
  `--dry-run` is also set).
- `--if-unmodified-since` GET current and
  abort if `modified_at` does not match (Datadog has no If-Match).
- Run validate unless `--skip-validate`.

### 5) `ddctl dashboards list [--limit <n>]`

List dashboards via `GET /api/v1/dashboard`.

Text output: ID, title, URL per row. JSON: `{"dashboards":[...]}`.

### 6) `ddctl dashboards search [--title <substr>] [--tag <tag>] [--limit <n>]`

Client-side filter after listing all dashboards. At least one of
`--title` or `--tag` is required. Warns when `--limit` truncates results.

### 7) `ddctl dashboards clone <id> --title <title> [--dry-run]`

Compose get → strip identity → create with the required `--title`.

### 8) `ddctl dashboards delete <id> --confirm <id>`

Hard delete via `DELETE /api/v1/dashboard/{id}`. `--confirm` must equal
the dashboard ID exactly.

Semantic diff example:

```text
+ group: AI enhancement
+ 1 widgets
~ template variable kube_namespace: acceptance -> production
- widget note: legacy OTEL widget
```

## Exit behavior

- `0`: success, including `--help` and no-data warnings
- `2`: malformed file/flags/payload or invalid query / missing monitor
- API/auth/network errors use existing ddctl error mapping
