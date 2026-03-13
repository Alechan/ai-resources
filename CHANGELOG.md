# Changelog

## Unreleased

### Added

- Initial repository bootstrap for AI resource management.
- Imported `gdrivectl` source into `tools/gdrivectl/src` as first-party code in this repository.
- CI workflow at `.github/workflows/ci.yml` with Go `1.24.2`, repository verification, and `gdrivectl` test execution.
- DataGrip datasource eval expansion with positive + negative prompts and results artifact:
  - `skills/datagrip-datasources/evals/v1-prompts.md`
  - `skills/datagrip-datasources/evals/v1-results-2026-03-13.md`

### Changed

- Updated `gdrivectl` references to treat this repository as canonical and deprecate external standalone source references.
- Migrated `gdrivectl` module/import path to monorepo ownership:
  - `github.com/Alechan/ai-resources/tools/gdrivectl/src`
- Updated `tools/gdrivectl/src/go.mod` language version from `go 1.22` to `go 1.24`.
- Restored backup-first policy requirements in DataGrip skill/agent/playbook/evals.
