# Changelog

## Unreleased

### Added

- `ddctl` tool: unofficial DataDog CLI using Chrome cookies for auth (doctor, logs-query, monitors-list, monitors-get, events-list commands)
- `ddctl logs-query`: pagination via `--cursor` and `--all` flags; `next_cursor` printed in text output
- `ddctl monitors-list`: list all monitors with optional `--tag` filter
- `ddctl monitors-get`: fetch a single monitor by ID
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

- Hardened `claude-statusline` with a pinned `ccusage` fallback, cached monthly totals, and graceful segment fallback behavior.
- Tightened `claude-statusline` to version-gate PATH `ccusage` binaries and clarified manual install documentation.
- Fixed `claude-statusline` daily cost reporting to use an explicit calendar-day `ccusage daily` query instead of parsing `statusline` output.
- Added conditional `shellcheck`-based linting for repository shell scripts in local verification and CI.
- Updated `gdrivectl` references to treat this repository as canonical and deprecate external standalone source references.
- Migrated `gdrivectl` module/import path to monorepo ownership:
  - `github.com/Alechan/ai-resources/tools/gdrivectl/src`
- Updated `tools/gdrivectl/src/go.mod` language version from `go 1.22` to `go 1.24`.
- Restored backup-first policy requirements in DataGrip skill/agent/playbook/evals.
