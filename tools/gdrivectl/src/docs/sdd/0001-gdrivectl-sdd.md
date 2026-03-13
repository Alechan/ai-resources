# SDD-0001: gdrivectl Core Design (Iteration 1)

Status: Draft  
Version: 0.2  
Last updated: 2026-03-05

## 1. Purpose

Define the first executable design for `gdrivectl`: a deterministic CLI for Google Drive/Docs operations that behaves consistently across Codex sessions and local shells.

## 2. SDD methodology used

This document combines proven architecture documentation practices:

- **ISO/IEC/IEEE 42010** mindset: document stakeholders, concerns, constraints, and architecture views.
- **arc42** structure influence: goals, constraints, context/scope, solution strategy, runtime/deployment concerns, decisions, risks.
- **C4 model** for visual viewpoints: context/container/component levels.
- **ADR practice (MADR style)** for decision capture and traceability.
- **RFC workflow** before implementation of significant changes.

## 3. Scope

### In scope (iteration 1)

- Authentication through existing `gcloud` login state.
- Drive search (`name` and `fullText`).
- File metadata retrieval.
- Google Doc export (text/markdown/docx/html/pdf where supported by API).
- Google Docs tab listing (tab metadata hierarchy).
- Machine-readable output (`json`) and human output (`table`/`text`).
- Stable exit codes for automation.

### Out of scope (iteration 1)

- Writing/updating Drive files.
- Full sync behavior replacement for Google Drive Desktop.
- UI/TUI.
- Multi-provider support (only Google Workspace APIs).

## 4. Stakeholders and concerns

- Primary user: engineer using Codex + terminal.
- Secondary user: scripts/CI jobs.

Top concerns:

- Predictability across shell environments.
- Fast troubleshooting when auth/scopes fail.
- Low-friction usage from Codex prompts.
- Clear contract for output parsing.

## 5. Context and constraints

- Runtime: macOS + shell + Codex.
- `gcloud` may not be in PATH in non-login shells.
- Drive and Docs APIs may behave differently per endpoint/doc.
- Shared drives must be supported.

Hard constraints:

- Must support absolute `gcloud` path override.
- Must support `supportsAllDrives` + `includeItemsFromAllDrives` flows.
- Must not require Google Drive Desktop app.

## 6. Functional requirements

- FR-1: Resolve access token from `gcloud auth print-access-token`.
- FR-2: `search` command with query, corpora, mime filters, pagination.
- FR-3: `file-meta` command by file ID.
- FR-4: `doc-export` command by doc ID + mime.
- FR-5: `doc-tabs` command by doc ID, returning flattened tab tree and hierarchy fields.
- FR-6: `doctor` command validating binary path, auth, and endpoint reachability.

## 7. Non-functional requirements

- NFR-1: Command startup < 300ms excluding network.
- NFR-2: Deterministic exit codes:
  - `0` success
  - `2` CLI misuse/validation
  - `3` auth/scope failure
  - `4` API timeout/network
  - `5` API semantic error (4xx/5xx with parsed payload)
- NFR-3: JSON output must be backward compatible per minor version.
- NFR-4: No secrets written to logs.

## 8. Architecture views

### 8.1 Context view (C4 level 1)

`gdrivectl` sits between user/automation and Google APIs, relying on local `gcloud` for token issuance.

### 8.2 Container view (C4 level 2)

- CLI process
- `gcloud` subprocess token provider
- Drive API
- Docs API

### 8.3 Component view (C4 level 3)

- CLI layer (`cmd/gdrivectl`): argument parsing and command routing.
- Auth adapter (`internal/auth`): token provider via `gcloud` command.
- Google API client (`internal/googleapi`): Drive/Docs HTTP wrappers.
- Output formatter (`internal/output`): json/text/table.
- Error model (`internal/errors`): typed, actionable failures.

### 8.4 Runtime flow

1. Parse args and config.
2. Resolve token via auth adapter.
3. Execute API operation.
4. Normalize response.
5. Render output and return stable exit code.

## 9. CLI contract (proposed)

- `gdrivectl doctor [--json]`
- `gdrivectl search --query <q> [--mime <m>] [--page-size 100] [--json]`
- `gdrivectl file-meta --id <file_id> [--json]`
- `gdrivectl doc-export --id <doc_id> --mime <mime> [--out <path>]`
- `gdrivectl doc-tabs --id <doc_id> [--json]`

Global flags:

- `--gcloud-bin <path>` default: `gcloud`
- `--timeout <duration>` default: `20s`
- `--json`

## 10. Error handling design

Each error must provide:

- Category (`auth`, `scope`, `network`, `api`, `validation`, `config`)
- Human message
- Suggested next action
- Optional raw API snippet

Example:

- `scope`: `ACCESS_TOKEN_SCOPE_INSUFFICIENT` -> suggest `gcloud auth login --enable-gdrive-access --update-adc`.

## 11. Security and privacy

- Do not print access tokens.
- Redact Authorization headers in debug logs.
- Minimize logged payload fields for Docs/Drive responses.
- Keep credentials managed by `gcloud`; no local credential vault in v1.

## 12. Testing strategy

- Unit tests: argument parsing, error mapping, output formatting.
- Contract tests: canned Drive/Docs JSON fixtures.
- Integration smoke tests (optional locally):
  - `doctor`
  - `file-meta` known ID
  - `doc-tabs` known multi-tab document

## 13. Milestones

- M1: CLI skeleton + `doctor` + token provider.
- M2: `search` and `file-meta`.
- M3: `doc-export` and `doc-tabs`.
- M4: tests + docs + release `v0.1.0`.

## 14. Open questions

- Should we support ADC fallback if `gcloud` is unavailable?
- Do we need structured retries by endpoint type?
- Should tab content extraction be included in v1 or left to v2?

## 15. Traceability

- Significant design changes require RFC under `docs/rfc/`.
- Accepted RFCs must update this SDD.
- Irreversible architecture decisions must be captured in `docs/adr/`.

## 16. External references

- ISO/IEC/IEEE 42010 standard page.
- IEEE 1016 standard page.
- arc42 overview and section model.
- C4 model official documentation.
- MADR guidance for ADRs.
