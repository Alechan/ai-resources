---
name: gdrivectl-drive-ops
description: Procedures for safe Google Drive file operations using the gdrivectl CLI tool. Use when asked to upload, download, list, or manage files in Google Drive.
---

# gdrivectl-drive-ops

## Purpose

Provide a repeatable procedure for safe Google Drive operations through `gdrivectl`.

## When To Use

- Listing files and folders in shared drives.
- Exporting Google Docs resources.
- Inspecting file metadata.
- Applying non-destructive organization changes requested by the user.

## Inputs

- User goal and scope (drive, folder, file IDs, or search terms).
- Expected output format (table, JSON, markdown summary).
- Credentials context (which Google identity should be used).

## Workflow

1. Confirm the exact target scope and output the user expects.
2. Run read-only commands first (`search`, `file-meta`, `doc-tabs`, `doc-export`) to gather state.
3. For write operations, restate the requested change and apply only approved edits.
4. Re-run relevant read-only checks to verify the resulting state.
5. Summarize changes, including command evidence and any skipped actions.

## Validation

- `gdrivectl --help` executes successfully.
- Returned IDs and paths match the user-provided scope.
- Post-change checks show expected state.

## Safety

- No destructive action without explicit user intent.
- Prefer read-only operations unless a write is clearly requested.
- Stop and report if auth scope or ownership is ambiguous.

## References

- `tools/gdrivectl/README.md`
- `docs/CONVENTIONS.md`
