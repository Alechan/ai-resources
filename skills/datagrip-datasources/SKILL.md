---
name: datagrip-datasources
description: Procedures for safely updating DataGrip datasource definitions with explicit, reversible, and validated changes. Use when asked to add, update, or remove DataGrip database connections.
---

# datagrip-datasources

## Purpose

Safely update DataGrip datasource definitions with explicit, reversible, and validated changes.

## When To Use

- Rotating credentials or connection parameters.
- Updating host/port/database values.
- Adjusting SSL, SSH tunnel, or driver options.
- Auditing datasource settings before and after a requested change.

## Inputs

- Explicit requested change set.
- Path to DataGrip configuration directory or exported datasource file.
- Environment constraints (local/dev/stage/prod).

## Workflow

1. Backup/export current datasource configuration before any edit.
2. Parse and validate intended changes against the current settings.
3. Reject implicit or out-of-scope modifications.
4. Apply only explicit user-requested changes.
5. Verify resulting connection settings and summarize diffs.

## Validation

- Backup artifact exists and is readable.
- Edited datasource contains only approved deltas.
- Connection details (host, port, database, auth mode, SSL mode) match the request.

## Safety

- No destructive action without explicit user intent.
- Never rotate secrets unless specifically requested.
- Stop when configuration ownership, environment target, or rollback path is unclear.

## References

- `playbooks/datagrip-datasource-update.md`
- `skills/datagrip-datasources/evals/v1-prompts.md`
- `docs/CONVENTIONS.md`
