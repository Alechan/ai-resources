# SDD-0004: v0.1 Documentation Completion

Status: Draft  
Version: 0.1  
Last updated: 2026-03-05

## 1. Purpose

Specify the minimum documentation set required to mark v0.1 as release-ready.

## 2. Scope

### In scope

- README quickstart and usage examples.
- Troubleshooting guidance for auth/scope/path/timeouts.
- Status and test-plan synchronization with implemented behavior.

### Out of scope

- Long-form tutorials.
- Public website/docs portal.

## 3. Required artifacts

- `README.md` must include:
  - what `gdrivectl` does
  - install/build command
  - quickstart examples for all five commands
  - global flags summary
- `docs/TEST_PLAN.md` must match real command flags and expected exits.
- `docs/STATUS.md` must reflect actual implementation and pending gaps.
- `docs/RELEASE.md` checklist items must be actionable and current.

## 4. Troubleshooting content requirements

At minimum, include sections for:

- missing/invalid `gcloud` binary path
- missing auth login
- insufficient Drive scope
- timeout/network failures
- common invalid-flag usage

Each section should include a concrete remediation command where possible.

## 5. Acceptance criteria

- A new contributor can run `doctor` and one data command using documented steps only.
- Docs mention exit-code contract and error categories.
- No documented command contradicts current CLI flags or defaults.

## 6. Traceability

- Implements open item #4 from `docs/STATUS.md`.
- Satisfies documentation readiness checklist in `docs/RELEASE.md`.
