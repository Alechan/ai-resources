# SDD-0005: v0.1 Fixture Contract Tests for Drive/Docs Clients

Status: Draft  
Version: 0.1  
Last updated: 2026-03-05

## 1. Purpose

Specify fixture-based contract tests needed to close v0.1 open item #2.

## 2. Scope

### In scope

- Fixture tests for Drive and Docs HTTP client behavior.
- Parsing success-path payloads into expected map structures.
- Error mapping for 401/403/network/non-2xx/malformed JSON scenarios.

### Out of scope

- Live API integration tests.
- Golden tests for full CLI stdout formatting.

## 3. Requirements

- FR-1: Drive client fixture tests cover:
  - `Search`
  - `FileMeta`
  - `ExportDoc`
- FR-2: Docs client fixture tests cover:
  - `DocTabs` nested tab payload handling
- FR-3: Error classification must be asserted for:
  - unauthorized (`auth`)
  - forbidden (`scope`)
  - timeout/network (`network`)
  - other non-2xx (`api`)
  - malformed JSON (`api`)

## 4. Fixture organization

- `testdata/drive/`:
  - `search_success.json`
  - `file_meta_success.json`
  - `error_401.json`
  - `error_403.json`
  - `error_500.json`
  - `malformed.json`
- `testdata/docs/`:
  - `doc_tabs_success_nested.json`
  - `error_401.json`
  - `error_403.json`
  - `error_500.json`
  - `malformed.json`

File names are normative to keep tests and fixtures predictable.

## 5. Test design constraints

- Tests must use local HTTP test servers (`httptest`) only.
- No token values are asserted or logged.
- Assertions focus on behavior contract (category/message/action presence), not exact remote error text.

## 6. Acceptance criteria

- All fixture tests run offline and deterministically.
- Each client method has both success and failure-path fixture coverage.
- Open item #2 from `docs/STATUS.md` is fully covered by tests mapped to this SDD.

## 7. Traceability

- Implements open item #2 from `docs/STATUS.md`.
- Supports code readiness and exit-code validation objectives in `docs/RELEASE.md`.
