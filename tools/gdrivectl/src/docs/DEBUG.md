# Debug Playbook

Use this playbook for common `gdrivectl` and `gcloud` failures.

## 1) `gdrivectl: command not found`

Install binary:

```bash
go install github.com/Alechan/ai-resources/tools/gdrivectl/src/cmd/gdrivectl@latest
command -v gdrivectl
```

If still unavailable, run from source as contributor fallback:

```bash
cd <repo-root>
go run ./cmd/gdrivectl --help
```

Then run commands as:

```bash
go run ./cmd/gdrivectl doctor --json
```

## 2) `gcloud binary not found` (exit code `2`)

Check if `gcloud` resolves:

```bash
command -v gcloud
```

If not found, either:

1. Use explicit path:

```bash
gdrivectl doctor --json --gcloud-bin "$HOME/.local/google-cloud-sdk/bin/gcloud"
```

2. Or add a PATH-visible symlink:

```bash
ln -sf "$HOME/.local/google-cloud-sdk/bin/gcloud" "$HOME/.local/bin/gcloud"
command -v gcloud
```

## 3) `unable to get access token` (exit code `3`)

Re-authenticate with Drive scope:

```bash
gcloud auth login --enable-gdrive-access --update-adc
gcloud auth list
```

Retry:

```bash
gdrivectl doctor --json
```

## 4) Sandbox/permission issue with gcloud config

Typical error includes inability to create files under `~/.config/gcloud`.

This is environment permission restriction. Retry in a shell with home-config access, or use:

```bash
CLOUDSDK_CONFIG=/tmp/gcloud-config gcloud auth list
```

## 5) Healthy baseline (`doctor --json`)

A healthy output should include:

```json
{
  "gcloud_bin": "gcloud",
  "gcloud_exists": true,
  "token_ok": true,
  "drive_ok": true,
  "docs_ok": true,
  "note": "all checks passed"
}
```

## 6) Canonical debug sequence

```bash
command -v gdrivectl
gdrivectl --help
command -v gcloud
gcloud auth list
gdrivectl doctor --json --gcloud-bin "$(command -v gcloud || echo gcloud)"
```

Source fallback if binary is missing:

```bash
go run ./cmd/gdrivectl doctor --json --gcloud-bin "$(command -v gcloud || echo gcloud)"
```

If exit code is:

- `2`: if config/auth-store failures are reported (for example `gcloud auth config store unavailable`), rerun the same command unsandboxed/escalated after basic flag/path checks.
- `3`: run scope re-auth command and retry.
- `4`: retry with `--timeout 60s` and verify network.

## 7) One-shot validation

```bash
command -v gcloud \
&& gdrivectl doctor --json --gcloud-bin "$(command -v gcloud)"
```

## 8) Exit code map

- `0`: success
- `2`: validation/config
- `3`: auth/scope
- `4`: network
- `5`: api
