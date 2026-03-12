# Playbook: DataGrip Datasource Update

## Goal

Update one or more DataGrip datasource definitions while preserving rollback safety.

## Preconditions

- You have the exact requested change list.
- You know the target environment and datasource names.
- You can access the current datasource config files.

## Procedure

1. Export or copy current datasource configuration to a timestamped backup.
2. Record baseline connection fields: host, port, database, username, SSL/SSH options.
3. Prepare a minimal patch containing only explicit user-requested changes.
4. Apply the patch.
5. Re-open datasource config and verify all relevant connection fields.
6. Summarize the applied diff and provide rollback path using the backup.

## Verification Checklist

- Backup exists before modification.
- No extra parameters changed.
- Final settings equal approved values.

## Rollback

Restore the backup export/copy and re-verify connection settings.
