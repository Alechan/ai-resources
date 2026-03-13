# SDD-0008: Auth Resolution and Diagnostics Hardening

Status: Draft  
Version: 0.1  
Last updated: 2026-03-05

## 1. Purpose

Define concrete improvements for `gcloud` binary resolution, doctor reporting accuracy, and auth/config diagnostics.

## 2. Problem statement

Current behavior has three practical gaps:

1. `doctor` can report `gcloud_exists=false` while token/API checks still pass.
2. Errors reference `GDRIVECTL_GCLOUD_BIN` but root parsing does not currently consume that env var.
3. Some permission/config-store failures from `gcloud auth print-access-token` are categorized as generic auth errors instead of config/environment issues.

## 3. Findings distilled

- PATH execution generally works today (`exec.CommandContext("gcloud", ...)`), so `--gcloud-bin` is not universally required.
- The core issue is resolution/reporting consistency and diagnostics quality, not a complete inability to run from PATH.
- Sandbox or restricted environments can surface config-store errors that deserve explicit classification and remediation guidance.
- Token acquisition is shared across command paths, so repeated token failures should be treated as auth/runtime-environment issues, not as an intrinsic property of any single command.
- Partial-success sessions are possible in practice (some commands succeed, later commands fail) due to execution-context differences; diagnostics must guide users without implying endpoint-specific auth design differences.

## 4. Design decisions

### 4.1 Binary resolution precedence (accepted)

Resolution order:

1. `--gcloud-bin` flag
2. `GDRIVECTL_GCLOUD_BIN` env var
3. `exec.LookPath("gcloud")`

Do not add machine-specific install-path scanning as a default strategy in v0.1.

Rationale:

- deterministic and portable
- avoids brittle host-specific assumptions
- aligns with Go execution conventions

### 4.2 Doctor reporting model (accepted)

`doctor` should report executable availability based on resolved executable discovery, not `os.Stat("gcloud")`.

Add reporting clarity:

- `gcloud_bin`: configured or resolved value used at runtime
- `gcloud_exists`: true when the final executable path resolves and is executable

### 4.3 Env/config messaging alignment (accepted)

If `GDRIVECTL_GCLOUD_BIN` is supported, keep it in help and error actions.
If not supported, remove mention. This SDD mandates supporting it.

### 4.4 Auth error diagnostics (accepted)

Enhance token-provider classification:

- scope-related failures -> `scope`
- executable missing/path issues -> `config`
- config/permission-store failures (for example credential DB/config dir write failures) -> `config`
- remaining token failures -> `auth`

Add explicit runtime guidance:

- if auth-class failures persist after re-auth in constrained environments, recommend unsandboxed/escalated retry and report this as environment limitation.
- if config-class failures indicate gcloud auth/config-store access problems in constrained environments, recommend unsandboxed/escalated retry and report this as environment limitation.

### 4.5 Known-path fallback scan (rejected)

Do not implement implicit known-path probing (for example `$HOME/.local/google-cloud-sdk/bin/gcloud`) as part of default resolution.

Reason:

- brittle across OS/package managers
- hides configuration issues
- creates unpredictable behavior

## 5. Required code changes

- `cmd/gdrivectl/root.go`
  - parse `GDRIVECTL_GCLOUD_BIN`
  - resolve executable path once using precedence
- `internal/app/config.go`
  - include resolved/configured gcloud path fields as needed
- `internal/app/app.go`
  - inject resolved path into token provider and doctor service
- `internal/service/doctor_service.go`
  - report based on resolved executable validity, not raw `os.Stat` on command token
- `internal/auth/gcloud_provider.go`
  - classify permission/config-store failures as `config`

## 6. Required test changes

- `cmd/gdrivectl/root_test.go`
  - precedence: flag > env > PATH
- `internal/auth/gcloud_provider_test.go`
  - config-store/permission error classification
- doctor reporting tests
  - `gcloud_exists=true` when resolution succeeds via PATH
  - consistency between reported binary and runtime invocation path

## 7. Acceptance criteria

- Users can run without `--gcloud-bin` when `gcloud` is in PATH.
- `doctor` no longer shows false-negative `gcloud_exists` in normal PATH-resolved setups.
- `GDRIVECTL_GCLOUD_BIN` behavior is implemented and documented.
- Permission/config-store failures are reported as config/environment remediation, not generic auth.
- Skill and runbook guidance explicitly include config-store-driven escalation criteria for exit `2`.
- Tests cover precedence and new diagnostic classifications.
- Documentation and skill guidance avoid claiming command-specific intrinsic auth sensitivity when root cause is shared token/runtime constraints.

## 8. Traceability

- Extends SDD-0001 auth/doctor requirements.
- Supports SDD-0007 debug-playbook reliability and remediation accuracy.
