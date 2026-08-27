---
name: datagrip-datasources
description: Procedures for updating project-local DataGrip datasource definitions under ~/DataGripProjects. Use when asked to add or edit a project datasource.
---

# datagrip-datasources

## Purpose

Safely update project-local DataGrip datasource definitions under `~/DataGripProjects`.

## When To Use

- Updating host, port, database, username, SSL, or driver options for a project datasource.
- Creating or editing a datasource stored in a project `.idea` folder.
- Checking what DataGrip has stored for a project-only connection.

## Inputs

- Explicit requested change set.
- Project path under `~/DataGripProjects`.
- Existing datasource file path or datasource name.
- Environment constraints (dev/test/acceptance/prod).

## Workflow

1. Find the project `.idea` directory under `~/DataGripProjects/<project>/`.
2. Back up the datasource XML before editing.
3. Read `dataSources.xml`, `dataSources.local.xml`, and `dataSources/` if present.
4. If SSH is enabled for a datasource, resolve the SSH profile through JetBrains global SSH config.
5. Apply only explicitly requested non-password changes.
6. Verify the final host, port, database, and user values.
7. Leave the password for manual entry in DataGrip.

## Validation

- Backup artifact exists and is readable.
- Edited datasource contains only approved deltas.
- Connection details match the request.
- For SSH-enabled datasources, `ssh-config-id` resolves to an existing SSH config.
- Password is intentionally left unset in files.

## Safety

- No destructive action without explicit user intent.
- Never guess missing file paths; verify by reading current config files.
- Stop when configuration ownership, environment target, or rollback path is unclear.

## DataGrip storage model

DataGrip stores datasource data across both project-local and global files:

- Project-local (editable per project):
  - `~/DataGripProjects/<project>/.idea/dataSources.xml`
  - `~/DataGripProjects/<project>/.idea/dataSources.local.xml`
  - `~/DataGripProjects/<project>/.idea/dataSources/*.xml`
- Global SSH configs (shared across projects):
  - macOS: `~/Library/Application Support/JetBrains/DataGrip*/options/sshConfigs.xml`

When a datasource uses SSH, project-local files usually store `<ssh-config-id>...`, and that ID must match `<sshConfig id="...">` in `sshConfigs.xml`.

## References

- `docs/CONVENTIONS.md`
- `README.md`
