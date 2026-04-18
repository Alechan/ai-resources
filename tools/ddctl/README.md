# ddctl Tool Resource

`ddctl` is an unofficial DataDog CLI maintained in this repository under `tools/ddctl/src`.
It authenticates using Chrome browser cookies (macOS). All operations are **read-only**.

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

## Usage

```
Usage: ddctl [global flags] <command> [flags]

Commands:
  doctor       Check Chrome cookies, DataDog auth, and reachability
  logs-query   Query DataDog logs

Global flags:
  --cookies-path <path>  Path to Chrome Cookies SQLite file
                           Env override: DDCTL_COOKIES_PATH
                           Default: ~/Library/Application Support/Google/Chrome/Default/Cookies
  --site <domain>        DataDog site domain (default: datadoghq.com)
                           Env override: DDCTL_SITE
  --timeout <duration>   Timeout per command (default: 30s)
  --json                 JSON output
  --debug                Debug logging
```

### doctor

Check that the Chrome cookies file exists, count DataDog cookies, and verify DataDog reachability.

```bash
ddctl doctor
ddctl doctor --json
```

### logs-query

Query DataDog logs with a search filter and time range.

```bash
ddctl logs-query --query "service:my-service status:error" --from now-1h --to now
ddctl logs-query -q "env:prod" --from now-4h --limit 100 --json
```

## Troubleshooting

- **Cookies file not found**: ensure `~/.../Chrome/Default/Cookies` exists; pass a custom path with `--cookies-path`.
- **Keychain access denied**: macOS may prompt for keychain access; accept the prompt or run `security find-generic-password -s "Chrome Safe Storage" -w` manually to verify.
- **Auth failures (HTTP 401/403)**: visit `app.datadoghq.com` in Chrome to refresh session cookies, then retry.
- **`command not found`**: ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `PATH`.
- **Network errors**: verify connectivity to `app.datadoghq.com`; retry with `--timeout 60s`.
