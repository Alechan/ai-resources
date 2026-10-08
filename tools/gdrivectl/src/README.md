# gdrivectl

`gdrivectl` is a spec-driven CLI for reliable Google Drive and Google Docs operations from Codex and terminal workflows.

## Build

```bash
go build ./...
```

## Install (recommended)

```bash
go install github.com/Alechan/ai-resources/tools/gdrivectl/src/cmd/gdrivectl@latest
```

Contributor local install from module root:

```bash
go install ./cmd/gdrivectl
```

Verify:

```bash
command -v gdrivectl
gdrivectl --help
```

If `gdrivectl` is not found, ensure `$(go env GOPATH)/bin` is in your PATH.

Standalone repository note: the previous standalone `gdrivectl` repository is deprecated; this monorepo path is canonical.

## Quickstart (binary-first)

Authenticate with Drive scope first:

```bash
gcloud auth login --enable-gdrive-access --update-adc
```

If needed, use absolute gcloud path:

```bash
GCLOUD=gcloud
```

Doctor check:

```bash
gdrivectl doctor --gcloud-bin "$GCLOUD"
```

Search:

```bash
gdrivectl search --query "name contains 'RFC'" --page-size 5 --json
```

File metadata:

```bash
gdrivectl file-meta --id <FILE_ID> --json
```

Doc tabs:

```bash
gdrivectl doc-tabs --id <DOC_ID> --json
```

Doc export:

```bash
gdrivectl doc-export --id <DOC_ID> --mime text/plain --out /tmp/doc.txt
```

Upload local file:

```bash
gdrivectl upload --path ./notes.txt --parent-id <FOLDER_ID> --json
```

## Quickstart (source fallback for contributors)

If binary install is unavailable, run from repository root:

```bash
go run ./cmd/gdrivectl --help
```

Optional end-to-end smoke harness:

```bash
GDRIVECTL_FILE_ID=<FILE_ID> GDRIVECTL_DOC_ID=<DOC_ID> scripts/smoke_integration.sh
```

## Commands

- `doctor`: Validate gcloud binary, token retrieval, and Drive/Docs reachability.
- `search`: Query Drive files.
- `file-meta`: Read metadata by file id.
- `doc-tabs`: Read Google Docs tabs metadata.
- `doc-export`: Export a Google Doc to a MIME type.
- `upload`: Upload a local file to Google Drive.

## Global flags

- `--gcloud-bin <path>`: gcloud binary path.
  - env override: `GDRIVECTL_GCLOUD_BIN`
- `--timeout <duration>`: per-command timeout (default `20s`).
- `--json`: JSON output for structured commands; `doctor` switches from text to JSON.
- `--debug`: enable debug logging.

## Output contract (v0.1)

- `search`, `file-meta`, `doc-tabs`, `upload`: JSON payload on success.
- `doctor`: text by default, JSON with `--json`.
- `doc-export`: raw bytes to stdout or `--out`; no JSON envelope on success.
- Errors are emitted to `stderr` with category and action hint when available.

## Exit codes

- `0`: success
- `2`: validation/config
- `3`: auth/scope
- `4`: network timeout/reachability
- `5`: API semantic error

## Troubleshooting

### gcloud binary not found

- Check:
  - `command -v gcloud`
- Use `--gcloud-bin <absolute_path>`.

### auth/scope issues

- Check:
  - `gcloud auth list`
- Run `gcloud auth login --enable-gdrive-access --update-adc`.

### timeout/network issues

- Retry with `--timeout 60s`.
- Verify DNS/connectivity.

### Sandbox/config permission issues

- If gcloud cannot write under `~/.config/gcloud`, retry in a shell with home-config access.
- Fallback:
  - `CLOUDSDK_CONFIG=/tmp/gcloud-config gcloud auth list`

### Canonical debug sequence

```bash
command -v gdrivectl
gdrivectl --help
command -v gcloud
gcloud auth list
gdrivectl doctor --json --gcloud-bin "$(command -v gcloud || echo gcloud)"
```

If `gdrivectl` is not installed, use source fallback from repo root:

```bash
go run ./cmd/gdrivectl doctor --json --gcloud-bin "$(command -v gcloud || echo gcloud)"
```

See full guide: `docs/DEBUG.md`.

## SDD-first workflow

1. Start with or update an SDD in `docs/sdd/`.
2. Propose significant design changes in `docs/rfc/`.
3. Record irreversible choices as ADRs in `docs/adr/`.
4. Implement code only after spec acceptance.

## Repository structure

- `docs/sdd/` Software Design Documents (versioned)
- `docs/rfc/` Feature/design proposals before implementation
- `docs/adr/` Architecture Decision Records
- `docs/templates/` Reusable templates for SDD/RFC/ADR
- `cmd/gdrivectl/` CLI entrypoint (Go)
- `internal/` Internal implementation packages
- `scripts/` Local automation scripts (dev/release/docs)

See first SDD iteration: `docs/sdd/0001-gdrivectl-sdd.md`.
