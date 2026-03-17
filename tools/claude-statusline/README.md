# claude-statusline Tool Resource

`claude-statusline` is maintained in this repository under `tools/claude-statusline/src`.

## Purpose

`claude-statusline` renders a compact Claude Code status line by combining `ccusage`
output with the Claude stdin JSON payload. It shows the active model, true
calendar-day cost, month cost, burn rate with block time, and context window
usage.

The script prefers a locally installed `ccusage` binary only when its version
matches the pinned expectation. Otherwise it falls back to `npx -y ccusage@16.2.3`
unless you explicitly opt into unpinned behavior.

Example output:

```text
🤖 Sonnet 4.6 | 💰 $0.82 today / $31.23 mo | 🔥 $1.20/hr 🟢 (1h 57m left) | 🧠 16.5k / 200k (82% left)
```

## Dependencies

- `jq` for context window parsing and daily/monthly total extraction
- either `ccusage` `16.2.3` on `PATH`, or `npx` with access to `ccusage@16.2.3`

## Install

Automatic install from this repository:

```bash
bash scripts/install_claude_statusline.sh
```

Manual install:

1. From the repository root, symlink the checked-in script into `~/.claude/`:

```bash
mkdir -p "$HOME/.claude"
ln -sf \
  "$(pwd)/tools/claude-statusline/src/statusline-command.sh" \
  "$HOME/.claude/statusline-command.sh"
```

2. Add this block to `~/.claude/settings.json`, replacing the example path with
   your actual home directory:

```json
"statusLine": {
  "type": "command",
  "command": "bash /Users/your-user/.claude/statusline-command.sh",
  "padding": 0
}
```

If `settings.json` already has a `statusLine` entry, resolve that conflict before
running the installer. The installer fails instead of overwriting an existing
status line configuration.

## Runtime behavior

- Monthly cost is cached for 15 minutes in
  `${XDG_CACHE_HOME:-$HOME/.cache}/claude-statusline`.
- Today cost is computed from an explicit `ccusage daily --since YYYYMMDD --until YYYYMMDD --json`
  query and cached for 60 seconds.
- The script omits unavailable segments instead of printing malformed placeholders.
- If both `ccusage` and `jq` are unavailable, it falls back to a single warning
  line instead of returning an empty status.

Environment variables:

- `CLAUDE_STATUSLINE_CCUSAGE_PACKAGE`: override the fallback `npx` package
  version.
- `CLAUDE_STATUSLINE_CCUSAGE_VERSION`: override the expected version for a PATH
  `ccusage` binary.
- `CLAUDE_STATUSLINE_ALLOW_UNPINNED_CCUSAGE=1`: allow any PATH `ccusage` version
  instead of requiring the pinned version match.
- `CLAUDE_STATUSLINE_DAILY_CACHE_TTL_SECONDS`: override the today-cost cache TTL.
- `CLAUDE_STATUSLINE_MONTHLY_CACHE_TTL_SECONDS`: override the monthly cost cache
  TTL.
- `CLAUDE_STATUSLINE_CACHE_DIR`: override the cache directory.
- `CLAUDE_STATUSLINE_DEBUG=1`: emit debug logs to stderr.

## Validate Source

```bash
bash scripts/verify_repo.sh
```

## Verify

```bash
bash -n tools/claude-statusline/src/statusline-command.sh
bash -n scripts/install_claude_statusline.sh
```

## Troubleshooting

- `jq: command not found`: install `jq` to restore context and monthly cost
  segments.
- The `today` amount looks wrong: compare it with
  `npx -y ccusage@16.2.3 daily --since YYYYMMDD --until YYYYMMDD --json`.
- `ccusage` is slow on every refresh: install `ccusage` `16.2.3` globally so the
  script can avoid `npx` startup overhead while staying deterministic.
- A global `ccusage` is installed but ignored: either align it with the pinned
  version or set `CLAUDE_STATUSLINE_ALLOW_UNPINNED_CCUSAGE=1`.
- `npx` or `ccusage` failures: verify Node.js and npm are installed and retry.
- Installer reports existing `statusLine`: edit `~/.claude/settings.json` manually
  or remove the existing entry before reinstalling.
