# RFC-0001: gdrivectl v0.1 CLI command surface

Status: Draft  
Owner: Alejandro Danos  
Date: 2026-03-05

## Problem

Accessing Google Drive/Docs from Codex is currently inconsistent due to shell/path differences, scope issues, and endpoint-specific failures. We need a stable CLI surface that is deterministic for both humans and automation.

## Goals

- Define a minimal but complete v0.1 command set for read-focused workflows.
- Standardize auth behavior through `gcloud` token retrieval.
- Provide stable output and exit-code contracts for automation.
- Make failure modes actionable (`auth`, `scope`, `network`, `api`, `validation`).

## Non-goals

- Mutating Drive/Docs content.
- Replacing Google Drive Desktop sync behavior.
- Building a TUI/UI.
- Supporting providers beyond Google Workspace APIs.

## Proposed design

### Commands

1. `gdrivectl doctor`

Purpose:

- Validate local setup and API reachability before running other commands.

Checks:

- `gcloud` binary exists and executable.
- access token can be retrieved.
- Drive metadata endpoint reachable.
- Docs endpoint reachable.

Output:

- `text` (default): pass/fail per check + remediation hints.
- `json`: structured check results.

2. `gdrivectl search`

Purpose:

- Query Drive files across personal and shared drives.

Required flags:

- `--query <q>`

Optional flags:

- `--mime <mime>` (repeatable)
- `--page-size <n>` default `100`
- `--corpora <allDrives|user|drive>` default `allDrives`
- `--drive-id <id>` required when `--corpora drive`
- `--fields <list>` (safe allowlist)

Output:

- Sorted by API default unless `--order-by` is added in later RFC.

3. `gdrivectl file-meta`

Purpose:

- Return metadata for one Drive file/document by ID.

Required flags:

- `--id <file_id>`

Optional flags:

- `--fields <list>` (default safe set)

4. `gdrivectl doc-tabs`

Purpose:

- Return Google Docs tabs hierarchy for a document.

Required flags:

- `--id <doc_id>`

Behavior:

- Calls Docs API with `includeTabsContent=true` and a minimal `fields=` projection.
- Returns flattened rows while preserving hierarchy fields:
  - `tabId`, `title`, `index`, `parentTabId`, `nestingLevel`

5. `gdrivectl doc-export`

Purpose:

- Export Google Docs content via Drive API.

Required flags:

- `--id <doc_id>`
- `--mime <mimeType>`

Optional flags:

- `--out <path>`; if omitted, writes to stdout.

Supported v0.1 mimes (allowlist):

- `text/plain`
- `text/markdown`
- `application/pdf`
- `application/vnd.openxmlformats-officedocument.wordprocessingml.document`
- `text/html`

### Global flags

- `--gcloud-bin <path>`
  - default: `gcloud`
- `--timeout <duration>` default `20s`
- `--json`
- `--debug`

### Exit-code contract

- `0`: success
- `2`: CLI misuse / validation error
- `3`: auth/scope error
- `4`: network timeout/reachability
- `5`: API semantic error (4xx/5xx parsed response)

### Error model

Every failure returns:

- category: `validation|config|auth|scope|network|api`
- message: human-readable
- action: concrete next step
- details: optional raw API snippet (redacted)

### Security constraints

- Never print OAuth tokens.
- Redact `Authorization` headers from debug logs.
- Do not persist API payloads by default.

## API mapping

- Drive metadata/search:
  - `GET /drive/v3/files`
  - `GET /drive/v3/files/{fileId}`
- Drive export:
  - `GET /drive/v3/files/{fileId}/export`
- Docs tabs:
  - `GET /v1/documents/{documentId}?includeTabsContent=true&fields=...`

## Risks

- Docs and Drive endpoints can fail differently for same doc/token.
- Certain document exports may intermittently timeout.
- `gcloud` path may vary across machines/sessions.

## Mitigations

- `doctor` command to isolate failures early.
- Absolute `--gcloud-bin` override in every command.
- Tight endpoint timeouts + categorized retry hints.
- Minimal `fields=` for Docs tabs requests.

## Rollout

### Phase 1

- Implement `doctor`, `file-meta`, `search`.
- Add unit tests for parsing/validation/error mapping.

### Phase 2

- Implement `doc-tabs`, `doc-export`.
- Add integration smoke scripts using known document IDs.

### Phase 3

- Stabilize JSON output and tag `v0.1.0`.

## Open questions

- Should `doc-tabs` offer `--tree` pretty output in v0.1 or v0.2?
- Should retry policy be command-specific in v0.1?
- Should ADC fallback be enabled when `gcloud` token retrieval fails?

## Acceptance criteria

- All five commands implemented with documented flags.
- Exit-code contract enforced and tested.
- `doctor` detects and reports at least:
  - missing gcloud binary
  - missing auth
  - insufficient scope
  - docs endpoint timeout
- SDD updated to reflect accepted deltas from this RFC.
