# Release checklist (v0.1.0)

## Preconditions

- RFC-0001 and RFC-0002 accepted.
- ADR-0001 accepted.
- SDD updated to latest accepted behavior.

## Code readiness

- [x] `go build ./...` passes.
- [x] Unit tests added and passing.
- [x] Core smoke commands script provided (`scripts/smoke_integration.sh`).
- [x] Exit code contract validated (unit + command tests).
- [ ] No secrets in logs or docs.

## Documentation readiness

- [x] `docs/STATUS.md` reflects final state.
- [x] `docs/TEST_PLAN.md` up to date.
- [x] `README.md` includes install and quickstart.
- [x] Troubleshooting section includes scope/path/timeouts.

## Versioning and tagging

- [ ] Choose version `v0.1.0`.
- [ ] Create changelog entry (summary + breaking notes if any).
- [ ] Tag release commit.

## Post-release follow-ups

- [ ] Open RFC for ADC fallback token provider strategy (v0.2).
- [ ] Open RFC for output envelope stabilization (if changed).
- [ ] Open issue for retry policy tuning by endpoint.
