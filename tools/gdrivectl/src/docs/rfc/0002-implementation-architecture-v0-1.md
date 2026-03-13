# RFC-0002: Implementation architecture for gdrivectl v0.1

Status: Draft  
Owner: Alejandro Danos  
Date: 2026-03-05

## Problem

RFC-0001 defines command behavior, but implementation boundaries are still implicit. Without explicit module contracts, the codebase can drift into tight coupling and brittle tests.

## Goals

- Define package boundaries and interfaces for v0.1.
- Specify error taxonomy and mapping rules.
- Define testing layers and fixtures.
- Enable incremental delivery (M1..M4) with stable contracts.

## Non-goals

- Final API for v1.0+.
- Plugin architecture.
- Cross-language SDKs.

## Proposed architecture

## 1. Directory/package layout

```text
cmd/gdrivectl/
  main.go
  root.go
  cmd_doctor.go
  cmd_search.go
  cmd_file_meta.go
  cmd_doc_tabs.go
  cmd_doc_export.go

internal/app/
  app.go                  # dependency wiring
  config.go               # global config + defaults

internal/auth/
  token_provider.go       # interface + implementation
  gcloud_provider.go

internal/googleapi/
  drive_client.go
  docs_client.go
  models.go

internal/service/
  doctor_service.go
  search_service.go
  file_meta_service.go
  doc_tabs_service.go
  doc_export_service.go

internal/output/
  writer.go               # json/text/table rendering

internal/fail/
  codes.go                # exit codes
  errors.go               # typed errors + constructors
  map.go                  # map upstream errors -> typed errors

testdata/
  drive/
  docs/
```

## 2. Core interfaces

### 2.1 Token provider

```go
type TokenProvider interface {
    AccessToken(ctx context.Context) (string, error)
}
```

Implementation:

- `GcloudTokenProvider` executes `${gcloudBin} auth print-access-token`.

### 2.2 Drive client

```go
type DriveClient interface {
    Search(ctx context.Context, req SearchRequest) (SearchResponse, error)
    FileMeta(ctx context.Context, req FileMetaRequest) (FileMetaResponse, error)
    ExportDoc(ctx context.Context, req ExportRequest) (io.ReadCloser, string, error)
}
```

### 2.3 Docs client

```go
type DocsClient interface {
    DocTabs(ctx context.Context, req DocTabsRequest) (DocTabsResponse, error)
}
```

### 2.4 Services

- Services are command-usecase orchestrators.
- Services only depend on interfaces, never concrete API/CLI packages.

## 3. Error taxonomy and mapping

Canonical categories:

- `validation`
- `config`
- `auth`
- `scope`
- `network`
- `api`

Rules:

- Missing/invalid flags -> `validation` (exit `2`).
- `gcloud` missing/not executable -> `config` (exit `2`).
- token retrieval failure / unauthenticated -> `auth` (exit `3`).
- `ACCESS_TOKEN_SCOPE_INSUFFICIENT` -> `scope` (exit `3`).
- timeout/DNS/refused -> `network` (exit `4`).
- other non-2xx with parsed body -> `api` (exit `5`).

All returned errors should include:

- category
- summary
- action hint
- optional details (redacted)

## 4. Output contract

### 4.1 JSON mode

- Envelope shape for all commands:

```json
{
  "ok": true,
  "command": "search",
  "data": {...},
  "error": null
}
```

Failure:

```json
{
  "ok": false,
  "command": "doc-tabs",
  "data": null,
  "error": {
    "category": "network",
    "message": "Docs API request timed out",
    "action": "Retry with --timeout 60s or run gdrivectl doctor",
    "details": null
  }
}
```

### 4.2 Text/table mode

- Human-readable summary first.
- Tabular rows for list responses.
- Hints printed for recoverable failures.

## 5. Config and precedence

Resolution order (highest to lowest):

1. CLI flags
2. Environment vars
3. Built-in defaults

Environment keys:

- `GDRIVECTL_GCLOUD_BIN`
- `GDRIVECTL_TIMEOUT`

## 6. Observability/logging

- Debug logs enabled only with `--debug`.
- Never log tokens or auth headers.
- Include request id/timestamp per command in debug mode.

## 7. Testing strategy

### 7.1 Unit tests

- token provider command execution and stderr parsing.
- error mapper (all category paths).
- flag/config precedence.
- output writer JSON/text snapshots.

### 7.2 Contract tests (HTTP fixtures)

- Drive search/file-meta/export responses.
- Docs tab tree flattening (nested tabs).
- Scope error payload mapping.

### 7.3 Optional integration smoke

- Controlled by env var (disabled by default in CI).
- Uses known IDs in local config.

## 8. Rollout plan

- Step 1: skeleton + `doctor` + typed errors.
- Step 2: Drive search + file-meta.
- Step 3: Docs tabs + export.
- Step 4: harden JSON contracts + docs/examples.

## 9. Risks

- Docs API intermittency by document.
- API schema drift for tab fields.
- Overcoupling CLI parsing with services.

## 10. Mitigations

- Keep docs and drive clients isolated.
- Keep response models minimal + tolerant.
- Use fixture-based regression tests for representative payloads.

## 11. Acceptance criteria

- Packages created according to layout.
- All core interfaces implemented and unit-tested.
- Error taxonomy enforced across commands.
- JSON output schema documented and stable in v0.1.
- RFC-0001 behavior fully supported.
