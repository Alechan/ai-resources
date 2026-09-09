# Changelog

## Unreleased

### Added

- Nested `ddctl monitors` group: list, get, validate, create, update, mute,
  unmute, and delete (raw JSON get with `options`; production unmute requires
  `--confirm`; delete requires `--confirm <id>`)
- `ddctl dashboards` list, search, clone, and delete (in addition to get,
  validate, create, update)
- Structured `--json` error envelopes with resource/widget/query fields when
  known; `--debug` HTTP logging with cookie and log-body redaction
- Nested `ddctl` help for `dashboards` and `monitors` subcommands (command-specific
  usage, exit 0)
- `ddctl dashboards`: query preflight for metrics/logs/monitor IDs,
  `--replace-all`, `--dry-run`, `--diff`, `--if-unmodified-since`,
  `--template-variable`
- `slackctl conversation export` now preserves root and thread-reply reactions
  in normalized JSON schema version 2, resolves returned reactor IDs as
  participants, and renders clearly labeled reaction metadata in Markdown;
  raw responses and manifest schema version 1 remain unchanged
- `slackctl conversation export` now accepts generic Slack archive permalinks as
  exact inclusive root-message lower boundaries, selects the Keychain workspace
  from the permalink host, and preserves complete selected threads beyond root
  time bounds
- `slackctl` tool and `slackctl-conversation-ops` skill for safe, resumable Slack conversation exports with raw, JSON, and Markdown output
- `scripts/install_cursor_skill.sh`: install skills into `~/.cursor/skills/` via symlinks
- `scripts/install_git_hooks.sh`: install git hooks that strip Co-authored-by trailers
- `ddctl` tool: unofficial DataDog CLI — added `metrics-query` (Phase 5)
- `ddctl logs-query`: pagination via `--cursor` and `--all` flags; `next_cursor` printed in text output
- `ddctl events-list`: list events in a time range with optional source/tag filters
- `internal/timeutil` package: shared relative-time parser (`now`, `now-1h`, `now-30m`, `now-2d`, `now-1w`, Unix ms, RFC3339)
- `ddctl-datadog-ops` skill: procedure for querying DataDog logs with ddctl
- Initial repository bootstrap for AI resource management.
- Imported `gdrivectl` source into `tools/gdrivectl/src` as first-party code in this repository.
- CI workflow at `.github/workflows/ci.yml` with Go `1.24.2`, repository verification, and `gdrivectl` test execution.
- `claude-statusline` tool resource with checked-in status line script, installer, and documentation.
- DataGrip datasource eval expansion with positive + negative prompts and results artifact:
  - `skills/datagrip-datasources/evals/v1-prompts.md`
  - `skills/datagrip-datasources/evals/v1-results-2026-03-13.md`

### Changed

- **Breaking:** removed `ddctl monitors-list` and `ddctl monitors-get`; use
  `ddctl monitors list` and `ddctl monitors get`
- **Breaking:** removed dashboard `--expected-modified-at`; use
  `--if-unmodified-since` only
- `ddctl dashboards validate` accepts unmodified `dashboards get` payloads
  (including logs `search.query`), treats no-data as a warning (exit 0), and
  prints widget-attributed query results
- `ddctl notebooks validate`: empty metric series is a warning (exit 0), aligned
  with dashboards; `--allow-empty-series` is accepted for compatibility only
- Hardened `claude-statusline` with a pinned `ccusage` fallback, cached monthly totals, and graceful segment fallback behavior.
- Tightened `claude-statusline` to version-gate PATH `ccusage` binaries and clarified manual install documentation.
- Fixed `claude-statusline` daily cost reporting to use an explicit calendar-day `ccusage daily` query instead of parsing `statusline` output.
- Added conditional `shellcheck`-based linting for repository shell scripts in local verification and CI.
- Updated `gdrivectl` references to treat this repository as canonical and deprecate external standalone source references.
- Migrated `gdrivectl` module/import path to monorepo ownership:
  - `github.com/Alechan/ai-resources/tools/gdrivectl/src`
- Updated `tools/gdrivectl/src/go.mod` language version from `go 1.22` to `go 1.24`.
- Restored backup-first policy requirements in DataGrip skill/agent/playbook/evals.

### Removed

- `ddctl monitors-list` and `ddctl monitors-get` (replaced by nested
  `ddctl monitors list` and `ddctl monitors get`)
