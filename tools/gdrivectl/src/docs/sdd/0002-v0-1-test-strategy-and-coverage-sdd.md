# SDD-0002: v0.1 Test Strategy and Coverage

Status: Draft  
Version: 0.1  
Last updated: 2026-03-05

## 1. Purpose

Specify the unit-test contract needed to close v0.1 open item #1.

## 2. Scope

### In scope

- Unit tests for:
  - `internal/fail`
  - `internal/auth`
  - command flag parsing in `cmd/gdrivectl`

### Out of scope

- End-to-end integration against live Google APIs in CI.
- Performance/load testing.

## 3. Requirements

- TR-1: `go test ./...` must execute deterministic tests without network dependency.
- TR-2: Each exit-code class (`0,2,3,4,5`) must be covered by automated tests.
- TR-3: Flag-validation tests must cover all required-flag failures.
- TR-4: Token provider tests must validate success, missing binary, and scope-insufficient paths.

## 4. Test layers

### 4.1 Unit tests

- `internal/fail`:
  - category-to-exit-code mapping
  - network message classification in `MapNetworkOrAPI`
- `internal/auth`:
  - `AccessToken` returns trimmed token on success
  - empty token error
  - missing executable error mapping
  - scope-related stderr mapping
- `cmd/gdrivectl`:
  - root args parse (`--gcloud-bin`, `--timeout`, `--json`, `--debug`)
  - command-level required flags (`search --query`, `file-meta --id`, `doc-tabs --id`, `doc-export --id/--mime`)

## 5. Acceptance criteria

- `go test ./...` passes locally with no external credentials.
- New tests cover open item #1 requirements from `docs/STATUS.md`.
- Test names clearly map to command/error contracts, not implementation internals.

## 6. Traceability

- Implements open item #1 from `docs/STATUS.md`.
- Constrains release checklist items in `docs/RELEASE.md` under code readiness.
