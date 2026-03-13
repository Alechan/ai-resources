# Implementation status

Last updated: 2026-03-05

## Scope baseline

- SDD: `docs/sdd/0001-gdrivectl-sdd.md`
- RFC command surface: `docs/rfc/0001-cli-v0-1-command-surface.md`
- RFC implementation architecture: `docs/rfc/0002-implementation-architecture-v0-1.md`
- ADR auth strategy: `docs/adr/0001-auth-token-provider-gcloud-first.md`

## Command status (v0.1)

- `doctor`: Implemented and smoke-tested
- `search`: Implemented and smoke-tested
- `file-meta`: Implemented and smoke-tested
- `doc-tabs`: Implemented and smoke-tested
- `doc-export`: Implemented and smoke-tested

## Architecture status (RFC-0002)

Implemented:

- `cmd/gdrivectl/*` command handlers
- `internal/app` wiring
- `internal/auth` gcloud token provider
- `internal/googleapi` Drive/Docs clients
- `internal/service` use-case services
- `internal/output` JSON writer
- `internal/fail` error categories + exit-code mapping

Pending/partial:

- None

## Open items before v0.1.0

1. None.

## Completed on 2026-03-05

1. Added unit tests for `fail`, `auth`, and command flag parsing.
2. Added fixture tests for Drive/Docs client parsing and error mapping.
3. Normalized v0.1 output contract without introducing a breaking JSON envelope.
4. Finalized docs with command examples and troubleshooting snippets.
5. Added optional integration smoke harness under `scripts/smoke_integration.sh`.

## Output contract decision (v0.1)

- No new envelope (`ok/data/error`) in v0.1 to avoid breaking consumers.
- `search`, `file-meta`, `doc-tabs` return JSON on success.
- `doctor` supports text (default) and JSON (`--json`).
- `doc-export` returns raw bytes on success.
