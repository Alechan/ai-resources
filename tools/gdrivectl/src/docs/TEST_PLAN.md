# Test plan (v0.1)

## Preconditions

- `gcloud` installed and reachable.
- Auth with Drive scope:

```bash
gcloud auth login --enable-gdrive-access --update-adc
```

- If PATH is unstable, use absolute gcloud path via `--gcloud-bin`.

## Build check

```bash
go build ./...
```

Expected: exits `0`.

## Install verification (recommended user mode)

```bash
go install github.com/Alechan/ai-resources/tools/gdrivectl/src/cmd/gdrivectl@latest
command -v gdrivectl
gdrivectl --help
```

Expected: binary resolves and help exits `0`.

## Smoke checks

### 1) Help

```bash
gdrivectl --help
```

Expected: usage and command list printed, exit `0`.

### 2) Doctor

```bash
gdrivectl doctor --gcloud-bin gcloud --timeout 10s
```

Expected:

- `gcloud exists: true`
- `token ok: true`
- `drive endpoint ok: true`
- `docs endpoint ok: true` (or clear network/scope failure with action hint)

### 2b) Doctor JSON mode

```bash
gdrivectl doctor --json
```

Expected: JSON object with `gcloud_bin`, `token_ok`, `drive_ok`, and `docs_ok`.

### 3) File metadata

```bash
gdrivectl file-meta --id <FILE_ID> --json
```

Expected: JSON with `id`, `name`, `mimeType`.

### 4) Drive search

```bash
gdrivectl search --query "name contains 'RFC'" --page-size 5 --json
```

Expected: JSON containing `files` array.

### 5) Docs tabs

```bash
gdrivectl doc-tabs --id <DOC_ID> --json
```

Expected: JSON containing `tabs` with `tabProperties`.

### 6) Docs export

```bash
gdrivectl doc-export --id <DOC_ID> --mime text/plain --out /tmp/doc.txt
```

Expected: `/tmp/doc.txt` created and non-empty.

### 7) Upload local file

```bash
echo "hello from gdrivectl" > /tmp/gdrivectl-upload.txt
gdrivectl upload --path /tmp/gdrivectl-upload.txt --parent-id <FOLDER_ID> --json
```

Expected: JSON object with uploaded file `id` and `name`.

### 8) Output contract consistency

- `search`, `file-meta`, `doc-tabs`, `upload` return JSON success payloads.
- `doctor` returns text by default and JSON with `--json`.
- `doc-export` writes raw bytes and does not print status text on success.

## Negative-path checks

### Missing required flag

```bash
gdrivectl file-meta
```

Expected: validation error, exit `2`.

### Upload invalid path

```bash
gdrivectl upload --path /tmp/does-not-exist.txt --json
```

Expected: validation error, exit `2`.

### Invalid gcloud path

```bash
gdrivectl doctor --gcloud-bin /invalid/gcloud
```

Expected: config/auth error with action hint, exit `2` or `3`.

### Timeout behavior

```bash
gdrivectl doc-tabs --id <DOC_ID> --timeout 1ms
```

Expected: network timeout category, exit `4`.

### Scope remediation

If response category is `scope`, run:

```bash
gcloud auth login --enable-gdrive-access --update-adc
```

## Exit code contract

- `0`: success
- `2`: validation/config
- `3`: auth/scope
- `4`: network timeout/reachability
- `5`: API semantic error

## Optional integration harness

Run local end-to-end smoke with known IDs:

```bash
GDRIVECTL_FILE_ID=<FILE_ID> \
GDRIVECTL_DOC_ID=<DOC_ID> \
scripts/smoke_integration.sh
```

Optional env overrides:

- `GDRIVECTL_GCLOUD_BIN`
- `GDRIVECTL_TIMEOUT`
- `GDRIVECTL_SEARCH_QUERY`
- `GDRIVECTL_EXPORT_MIME`
- `GDRIVECTL_EXPORT_OUT`

## Debug playbook checks

### Canonical sequence

```bash
command -v gdrivectl
gdrivectl --help
command -v gcloud
gcloud auth list
gdrivectl doctor --json --gcloud-bin "$(command -v gcloud || echo gcloud)"
```

Expected:

- Command list prints correctly.
- `gcloud` resolves or a clear config remediation path is provided.
- `doctor --json` output includes `gcloud_bin`, `gcloud_exists`, `token_ok`, `drive_ok`, `docs_ok`.

### Sandbox fallback

If shell permissions block `~/.config/gcloud` writes, verify:

```bash
CLOUDSDK_CONFIG=/tmp/gcloud-config gcloud auth list
```

Then retry `doctor --json`.

### Sandbox config-store escalation rule

If a command returns exit `2` with config-store/auth-store signatures (for example `gcloud auth config store unavailable`), apply:

1. keep least-privilege attempt as recorded first run
2. preserve same command and arguments
3. rerun unsandboxed/escalated
4. record before/after outputs and exit codes

## Source fallback (contributors)

If binary install is unavailable, run all checks from repo root using:

```bash
go run ./cmd/gdrivectl <command> ...
```
