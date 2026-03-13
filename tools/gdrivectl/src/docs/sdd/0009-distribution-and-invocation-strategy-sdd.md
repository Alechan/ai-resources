# SDD-0009: Distribution and Invocation Strategy

Status: Draft  
Version: 0.1  
Last updated: 2026-03-05

## 1. Purpose

Define a standard invocation strategy that avoids repository-root coupling for normal users while preserving contributor workflows.

## 2. Problem

Using `go run ./cmd/gdrivectl ...` as the primary pattern couples execution to a local source checkout and repository root. This is correct for development but brittle for user workflows and cross-repo automation.

## 3. Scope

### In scope

- user-facing invocation policy
- documentation and skill guidance
- install verification and fallback behavior

### Out of scope

- package-manager distribution (brew/apt/choco)
- release pipeline automation

## 4. Decision

Adopt **binary-first invocation** with **source fallback**:

1. Primary mode (users/automation): `gdrivectl ...` from PATH.
2. Fallback mode (contributors/dev): `go run ./cmd/gdrivectl ...` from repo root.

## 5. Installation standard

Recommended install:

```bash
go install github.com/Alechan/ai-resources/tools/gdrivectl/src/cmd/gdrivectl@latest
```

Verification:

```bash
command -v gdrivectl
gdrivectl --help
```

If not available, document PATH setup for Go bin.

## 6. Skill and runbook policy

- Skills/subagents must prefer `gdrivectl` binary invocation.
- If binary is missing, they must explicitly state source fallback requirements:
  - run from gdrivectl repository root
  - use `go run ./cmd/gdrivectl ...`
- Responses should indicate which invocation mode was used.

## 7. Documentation requirements

Required updates across docs:

- README: install section + binary-first quickstart + source fallback
- DEBUG: binary-first checks before source commands
- TEST_PLAN: include install verification and retain source-run contributor checks

No operational guide should imply source-run is the only supported path.

## 8. Acceptance criteria

- A user can execute `gdrivectl --help` from any directory after installation.
- Skills are location-agnostic by default.
- Debug/test docs clearly differentiate user mode and contributor mode.

## 9. Traceability

- Extends SDD-0004 documentation completion requirements.
- Complements SDD-0006 skill behavior requirements.
- Aligns with SDD-0007 debug playbook portability goals.
