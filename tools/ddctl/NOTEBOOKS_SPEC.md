# ddctl Notebook Commands Spec (v1)

## Goal

Provide safe CLI commands to read and write DataDog notebooks through browser-session auth, while preventing common destructive mistakes (especially partial `PUT` replacement).

## Commands

`ddctl notebooks <subcommand> [flags]`

### 1) `ddctl notebooks get <id>`

Fetch notebook JSON from:

`GET /api/v1/notebooks/{id}?include_metadata=true`

Flags:
- `--include-metadata` (default: true)
- `--json` (global output flag)

Behavior:
- Text output: notebook id, name, cell count.
- JSON output: raw API response.

### 2) `ddctl notebooks create --from-file <path> [--name <name>] [--time <live_span>]`

Create notebook via:

`POST /api/v1/notebooks`

Accepted input file shapes:
1. `{"data":{"type":"notebooks","attributes":{...}}}` (API-like envelope)
2. `{"attributes":{...}}`

Normalization:
- Ensure `data.type = "notebooks"`.
- Drop `data.id` if present.
- Apply optional `--name` / `--time` overrides.

### 3) `ddctl notebooks update <id> --from-file <path> --replace-all`

Update notebook via:

`PUT /api/v1/notebooks/{id}`

Safety:
- Requires `--replace-all` explicitly.
- Reject payloads missing required full-replacement fields:
  - `attributes.name`
  - `attributes.time`
  - `attributes.cells` (non-empty array)

Normalization:
- Force `data.type = "notebooks"`.
- Force `data.id = <id>`.

### 4) `ddctl notebooks validate --from-file <path> [--from <time>] [--to <time>] [--allow-empty-series]`

Local schema checks:
- Supported cell wrapper shape (`type: notebook_cells`, `attributes.definition`).
- Timeseries query shape:
  - `definition.type == "timeseries"`
  - `requests[].queries[]` entries include `data_source`, `name`, `query`.

Online query preflight (best effort):
- Extract metric queries from timeseries cells.
- Execute each with DataDog metrics API.
- If query returns no series:
  - warning by default
  - validation failure unless `--allow-empty-series` is set.

## Known caveats captured by the CLI docs

1. Notebook `PUT` behaves as full replacement.
2. `GET /api/v1/notebooks/template/{id}` can return 404; clone template in UI, then use notebook ID.
3. `aws.sqs.*` queries should be scoped by queue tags (e.g. `queuename`) rather than `kube_namespace`.
4. Pod-name filters are brittle; prefer stable service/namespace filters when possible.

## Exit behavior

- `0`: success
- validation error: malformed file/flags/payload
- API/auth/network errors use existing ddctl error mapping
