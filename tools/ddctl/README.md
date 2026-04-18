# ddctl Tool Resource

`ddctl` is an unofficial DataDog CLI maintained in this repository under `tools/ddctl/src`.
It authenticates using DataDog session cookies stored in the macOS Keychain. All operations are **read-only**.

## Quick Start

1. Install: `go install ./cmd/ddctl` (from `tools/ddctl/src`)
2. Get cookies from Chrome DevTools (see [Workflow](#workflow) below)
3. `ddctl init --curl '<paste cURL here>'`
4. `ddctl doctor`
5. `ddctl logs-query --query "service:my-svc"`

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

## Workflow

1. Open Chrome and log in to `https://app.datadoghq.com`
2. Open DevTools (Cmd+Option+I) → Network tab
3. Filter by "Fetch/XHR", then reload the page or click any DataDog UI element
4. Right-click any request to `app.datadoghq.com` → Copy → **Copy as cURL**
5. Run: `ddctl init --curl '<pasted cURL command>'`
6. Verify: `ddctl doctor`

## Design decisions

### Why the CLI parses the cURL, not the AI

When used through an AI skill, the user pastes a cURL command into the chat and the skill calls `ddctl init --curl '...'`. The Cookie header extraction happens inside the CLI, not in the LLM.

This is intentional:

- **Deterministic by nature.** Extracting a `Cookie:` header from a cURL string is a mechanical regex match — no ambiguity, no judgment required. That's the wrong job for an LLM.
- **Consistent and tested.** The parser has unit tests and behaves identically regardless of which model or prompt runs the skill.
- **Safe at the boundary.** Cookie values contain `=`, `;`, and quotes. An LLM parsing and re-serialising them risks silent corruption that only surfaces as a confusing 401 later.
- **Self-contained CLI.** The tool works without an AI in the loop. A human can run `ddctl init --curl '...'` directly.

The AI skill's job is to know *when* and *why* to call `ddctl init` — guiding the user through DevTools and deciding which command to run. String processing belongs in code.

### Why macOS Keychain, not a config file

An earlier design read cookies directly from Chrome's SQLite database. That required deriving Chrome's master AES key from macOS Keychain, which could decrypt *any* Chrome cookie (Google, banks, everything) — a much broader blast radius than needed.

The current approach stores only the DataDog session string, scoped under service `"ddctl"`. A config file (`~/.config/ddctl/session.json`) would work too, but Keychain gives OS-level access control and keeps credentials out of the filesystem where they might be swept up by backups, dotfile sync, or accidental `cat`.

## Credential Storage

Cookies are stored in the macOS Keychain under:
- **Service**: `ddctl`
- **Account**: the DataDog site domain (e.g. `datadoghq.com`)

To inspect manually: `security find-generic-password -s "ddctl" -a "datadoghq.com" -w`

To clear stored credentials: `ddctl init --clear`

## Refresh Session

When your DataDog session expires (HTTP 401/403 errors), repeat the workflow above:
```bash
ddctl init --curl '<new cURL from Chrome DevTools>'
```

## Usage

```
Usage: ddctl [global flags] <command> [flags]

Commands:
  init         Store DataDog session cookies from a cURL command or raw cookie string
  doctor       Check credentials, DataDog auth, and reachability
  logs-query   Query DataDog logs

Global flags:
  --site <domain>        DataDog site domain (default: datadoghq.com)
                           Env override: DDCTL_SITE
  --timeout <duration>   Timeout per command (default: 30s)
  --json                 JSON output
  --debug                Debug logging
```

### init

Store DataDog session cookies in the macOS Keychain.

```bash
# From a cURL command (recommended)
ddctl init --curl 'curl "https://app.datadoghq.com/api/v1/validate" -H "Cookie: DD_S=abc; ..."'

# From a raw cookie string
ddctl init --cookie 'DD_S=abc123; dd_csrf_token=xyz'

# Clear stored credentials
ddctl init --clear
```

### doctor

Check that credentials exist in Keychain and DataDog is reachable.

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

- **Credentials not found**: run `ddctl init --curl '<cURL from Chrome DevTools>'`.
- **Auth failures (HTTP 401/403)**: your session has expired; re-run `ddctl init` with a fresh cURL.
- **Keychain access denied**: macOS may prompt for keychain access; accept the prompt.
- **`command not found`**: ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `PATH`.
- **Network errors**: verify connectivity to `app.datadoghq.com`; retry with `--timeout 60s`.
