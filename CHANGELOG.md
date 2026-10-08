# Changelog

## Unreleased

### Added

- `gchatctl search messages` writes hit permalinks when thread and message
  web IDs are present in the search envelope
- `gchatctl members list --space` replays a learned `list_members` POST and
  writes JSON/Markdown member exports; names stay in files
- `gchatctl spaces get` replays a learned `get_group` POST for one space ID
- `gchatctl search messages` replays a learned Chat `batchexecute` search
  and writes JSON/Markdown hit exports; `learn search.messages` stores that
  form body without replacing `list_topics`
- `gchatctl spaces list` replays a learned `paginated_world` POST and writes
  JSON/Markdown space exports; `learn paginated_world` stores that body
  without replacing `list_topics`
- `gchatctl conversation export` sends a `heartbeat` RPC between pages
  using the stored list_topics client blob
- `gchatctl` refreshes `x-framework-xsrf-token` from Chat HTML once after
  HTTP 401/403, persists it, and retries the RPC
- `gchatctl doctor --live` probes the stored session and reports
  `ok`, `http_401`, `http_403`, or `http_other` without printing secrets
- `gchatctl doctor` prints a redacted `list_topics` body slot map
- `gchatctl conversation export` takes a `chat.google.com/room` permalink as
  an inclusive starting message and pages `list_topics` until that thread and
  later topics are written locally
- `gchatctl topics list` replays a stored `list_topics` POST, optionally
  filters message text locally, and writes JSON/Markdown exports

### Fixed

- `gchatctl conversation export` keeps permalink Referer and space headers
  after applying the captured session, converts a topic-selector
  `list_topics` body into a time-cursor history request, and preserves the
  captured page size
- `gchatctl` streams large HAR captures, skips static assets, and writes
  deduplicated redacted fixtures with request counts

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

- Generic skill installers, worktree/diagram guidance, and Datadog examples/test fixtures are organization-agnostic; external skill roots are configurable.
- `gchatctl topics list --space` fetches that space via rewritten
  `list_topics` pages; `--query` is now `--contains` (local text filter)
- `gchatctl-conversation-ops` skill routes Chat read asks to exact commands
  and documents what stdout may contain
- `ddctl monitors create --muted` applies global mute in the initial POST
  (`options.silenced["*"]`), verifies mute state before success, and reports
  `muted` / `mute_scope` in JSON output; post-create mute is defensive only
- `ddctl monitors update` preserves remote global mute when the file omits
  `options.silenced`
  `ddctl logs query`, `ddctl metrics query`, and `ddctl events list`
- HTTP client uses request context for deadlines (removed duplicate `http.Client.Timeout`)
- CSRF body token injection limited to mutation and logs-analytics paths
- GET requests retry on HTTP 429/503 (up to 2 retries; logged with `--debug`)
- `ddctl events list --all` auto-paginates like `logs query --all`
- Shared mutation helpers for dashboard/monitor/notebook update dry-run and
  `--if-unmodified-since` checks (`internal/service/mutation.go`, `jsonutil.go`)
- **Breaking:** removed `--allow-empty-series` from dashboards and notebooks
  validate/create/update (no-data remains a warning, exit 0)
- `ddctl notebooks create`/`update`: validate-before-write, `--dry-run`,
  `--skip-validate`, `--if-unmodified-since`, and `--diff` (parity with dashboards)
- `--json` errors always use structured envelopes; `doctor --json` uses the
  same exit codes as text mode; API `details` redacted unless `--debug`
- Keychain-centric auth messaging; macOS guard on `init`/`doctor`; unified
  cookie parsing in `internal/auth`
- **Breaking:** removed `ddctl monitors-list` and `ddctl monitors-get`; use
  `ddctl monitors list` and `ddctl monitors get`
- **Breaking:** removed dashboard `--expected-modified-at`; use
  `--if-unmodified-since` only
- `ddctl dashboards validate` accepts unmodified `dashboards get` payloads
  (including logs `search.query`), treats no-data as a warning (exit 0), and
  prints widget-attributed query results
- `ddctl notebooks validate`: empty metric series is a warning (exit 0), aligned
  with dashboards
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

- `logs-query`, `metrics-query`, and `events-list` (replaced by nested
  `ddctl logs query`, `ddctl metrics query`, and `ddctl events list`)
- `ddctl monitors-list` and `ddctl monitors-get` (replaced by nested
  `ddctl monitors list` and `ddctl monitors get`)
