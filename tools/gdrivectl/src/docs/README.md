# Documentation model

This repo follows a lightweight spec-driven model.

## Structure

- `sdd/`: canonical design specification(s)
- `rfc/`: proposals for significant changes
- `adr/`: architecture decisions and rationale
- `templates/`: reusable templates
- `research/`: external references and distilled best practices
- `STATUS.md`: implementation state vs spec
- `TEST_PLAN.md`: smoke and negative-path test procedures
- `RELEASE.md`: release readiness checklist

## Workflow

1. Create or update an RFC for non-trivial changes.
2. Review and accept/reject RFC.
3. Update SDD with accepted behavior.
4. Add ADRs for irreversible decisions.
5. Implement code.

## Quality gates

- Every command behavior implemented in code must be specified in SDD.
- Every major design change must be traceable to an RFC/ADR.
- SDD should be implementation-ready, not aspirational.
