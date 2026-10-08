# ADR-0001: Authentication token provider strategy (gcloud-first)

Status: Accepted  
Date: 2026-03-05  
Related: SDD-0001, RFC-0001

## Context

`gdrivectl` needs OAuth access tokens for Drive and Docs APIs. In practice, the most reliable and user-friendly setup for this environment is an already authenticated Google Cloud CLI session.

Observed constraints:

- Shell/path inconsistency can break plain `gcloud` invocation.
- Scope errors are common unless login is done with Drive access.
- We need consistent behavior in Codex + terminal workflows.

## Decision

Use a **gcloud-first** token provider in v0.1:

1. Primary token source: `gcloud auth print-access-token`.
2. CLI must support explicit `--gcloud-bin <path>` to avoid PATH drift.
3. Default path is `gcloud`.
4. Scope remediation guidance is part of error handling:
   - `gcloud auth login --enable-gdrive-access --update-adc`
5. ADC fallback is deferred to a later RFC/version.

## Consequences

### Positive

- Minimal setup for engineers already using `gcloud`.
- No custom credential persistence in `gdrivectl` v0.1.
- Better operational consistency in Codex sessions.

### Negative

- Depends on local `gcloud` installation quality.
- Non-gcloud environments require extra setup (future work).

### Neutral

- The design remains extensible: we can add provider chaining later (`gcloud -> ADC -> explicit token`) without breaking command contracts.

## Alternatives considered

1. **ADC-first**
- Rejected for v0.1: less transparent troubleshooting for this workflow.

2. **Direct OAuth flow in gdrivectl**
- Rejected for v0.1: higher complexity and secret-handling burden.

3. **Require environment token (`GOOGLE_OAUTH_ACCESS_TOKEN`)**
- Rejected for v0.1: brittle and easy to misuse in day-to-day local usage.

## Follow-ups

- Add RFC for token provider fallback chain in v0.2.
- Add `doctor` checks specifically validating the chosen `gcloud` binary and token scope hints.
