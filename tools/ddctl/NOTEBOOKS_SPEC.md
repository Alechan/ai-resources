# ddctl Notebook Commands Spec (v2)

## Goal

Provide safe CLI commands to read and write DataDog notebooks through browser-session auth, while preventing common destructive mistakes (especially partial `PUT` replacement) and false validation failures on real notebook exports.

## Commands

`ddctl notebooks <subcommand> [flags]`

### 1) `ddctl notebooks get <id>`

Fetch notebook JSON from:

`GET /api/v1/notebooks/{id}?include_metadata=true`

Flags:
- `--include-metadata` (default: true)
- `--raw` — with `--json`, emit the full Datadog response (default: concise summary)
- `--json` (global output flag)

Behavior:
- Text output: notebook id, name, cell count.
- JSON output: concise summary by default; `--raw` returns the full API response.

### 2) `ddctl notebooks create --from-file <path> [--name <name>] [--time <live_span>]`

Create notebook via:

`POST /api/v1/notebooks`

Preflight: metric queries are validated before POST unless `--skip-validate`.
Use `--dry-run` to validate without creating.

Accepted input file shapes:
1. `{"data":{"type":"notebooks","attributes":{...}}}` (API-like envelope)
2. `{"attributes":{...}}`

Normalization:
- Ensure `data.type = "notebooks"`.
- Drop `data.id` if present.
- Apply optional `--name` / `--time` overrides.
- Convert deprecated `template_variables[].default` to `defaults` (warning).

### 3) `ddctl notebooks update <id> --from-file <path> --replace-all`

Update notebook via:

`PUT /api/v1/notebooks/{id}`

Safety:
- Requires `--replace-all` explicitly.
- Preflight: metric queries are validated before PUT unless `--skip-validate`.
- `--dry-run --replace-all` prints diff without PUT.
- `--if-unmodified-since` aborts when remote `modified_at` differs.
- Reject payloads missing required full-replacement fields:
  - `attributes.name`
  - `attributes.time`
  - `attributes.cells` (non-empty array)

Normalization:
- Force `data.type = "notebooks"`.
- Force `data.id = <id>`.

### 4) `ddctl notebooks validate --from-file <path> [--from <time>] [--to <time>]`

Local schema checks (shared with create/update via `PrepareNotebookSchema`):
- Envelope: `data.attributes` with `name`, `time`, non-empty `cells`.
- Cell wrapper: `type: notebook_cells`, `attributes.definition`.
- Text cells (`markdown`, `rich_text`, `note`): non-empty `definition.text`.
- Chart cells (`timeseries`, `heatmap`, `distribution`, `toplist`, `query_value`, `change`, `scatterplot`, `geomap`, `servicemap`, `trace`, `log_stream`):
  - Non-empty `definition.requests`.
  - Each request must include non-empty legacy `q` or at least one `queries[]` entry with non-empty `query`.
  - `timeseries` additionally requires `attributes.graph_size`, `attributes.split_by`, and `attributes.time` (null allowed).

Template variables:
- `$name` and `$name.value` in metric queries resolve to `template_variables[].defaults[0]` for preflight.
- Conflicting `default` and `defaults` on the same variable is a validation error.
- Deprecated `default` alone is converted to `defaults` with a warning.

Online query preflight (best effort):
- Extract metric queries from chart cells (both `q` and `queries[]`).
- Execute each resolved query with the DataDog metrics API.
- Validate output includes `cell_index`, `request_index`, `original`, and `resolved` per query.
- Empty metric series in the selected window is a **warning** (exit 0), not invalidity.

## JSON output

- `--json` on get/create/update returns a concise mutation summary by default (`id`, `name`, `url`, `cell_count`, etc.).
- `--raw` with `--json` returns the full Datadog API envelope.
- Dry-run and `--diff` fields are preserved in summary output when present.

## API errors

HTTP error responses parse Datadog `errors[]` (or top-level `error`) into the ddctl error envelope.
Text mode prints parsed `Server error:` lines and always includes a redacted `Details:` response body when Datadog returned one (including non-JSON and empty-body cases).

## Alias preservation

Visible timeseries legend aliases are stored in `formulas[].alias` with `queries[]` and `response_format`.
Legacy `metadata[].alias_name` on `q` requests is rejected during validation.

## Concurrency guard

`notebooks get` exposes `modified_at` (from `attributes.modified`).
`notebooks update` compares the remote revision against `--if-unmodified-since` or `attributes.modified` from the file unless `--force` is set.
Stale updates fail with guidance to re-get and merge; `--force` is separate from `--replace-all`.

## Known caveats captured by the CLI docs

1. Notebook `PUT` behaves as full replacement.
2. `GET /api/v1/notebooks/template/{id}` can return 404; clone template in UI, then use notebook ID.
3. `aws.sqs.*` queries should be scoped by queue tags (e.g. `queuename`) rather than `kube_namespace`.
4. Pod-name filters are brittle; prefer stable service/namespace filters when possible.

## Exit behavior

- `0`: success (including validate with no-data warnings)
- validation error: malformed file/flags/payload
- API/auth/network errors use existing ddctl error mapping
