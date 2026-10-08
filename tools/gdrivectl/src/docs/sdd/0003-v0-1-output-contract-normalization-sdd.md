# SDD-0003: v0.1 Output Contract Normalization

Status: Draft  
Version: 0.1  
Last updated: 2026-03-05

## 1. Purpose

Define a single output contract strategy for all commands before v0.1.0 release.

## 2. Problem

Current behavior is mixed:

- data commands return JSON payloads directly
- `doctor` supports text unless `--json`
- errors are emitted in text to `stderr`

This creates ambiguity for automation consumers.

## 3. Decision to specify

For v0.1, keep command payload compatibility and normalize at the command policy level:

- `--json` guarantees machine-readable success payloads for all commands that return structured data.
- Errors always map to stable categories and exit codes.
- `doc-export` remains byte-stream oriented and does not wrap output in JSON when successful.

Full envelope (`ok/data/error`) is deferred to a later version to avoid breaking current consumers.

## 4. Requirements

- OR-1: All structured commands support `--json` and output valid JSON on success.
- OR-2: `doc-export` success output is raw bytes (stdout or `--out` path), never mixed with text status lines.
- OR-3: Error messages include category and action hint when available.
- OR-4: Exit code mapping remains:
  - `2`: validation/config
  - `3`: auth/scope
  - `4`: network
  - `5`: api/unknown

## 5. Command contract matrix

- `doctor`:
  - default: human-readable checks
  - `--json`: structured doctor report
- `search`, `file-meta`, `doc-tabs`:
  - always JSON success payloads
- `doc-export`:
  - success: raw exported bytes
  - failure: categorized stderr error + exit code

## 6. Non-goals

- Retrofitting a new breaking envelope into v0.1.
- Introducing new output modes beyond current text/json/bytes behavior.

## 7. Acceptance criteria

- Documentation clearly states per-command output behavior.
- Tests assert JSON validity for structured success paths.
- No command emits mixed binary + status text on success path.

## 8. Traceability

- Implements open item #3 from `docs/STATUS.md`.
- Aligns with release checklist output/contract consistency goals.
