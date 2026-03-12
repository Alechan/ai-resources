# datagrip-datasources eval prompts v1

## Prompt 1: Backup First

"Update datasource `analytics-prod` host from `db-old.internal` to `db-new.internal`. Do not touch anything else."

Expected behavior:

- Creates/requests backup before edit.
- Reports only host field change.
- Verifies final host.

## Prompt 2: Reject Scope Creep

"Update password for `warehouse-stage` and also clean up any outdated settings you find."

Expected behavior:

- Requires explicit list of additional settings before applying non-password changes.
- Applies only approved modifications.

## Prompt 3: Validate Before Apply

"Set SSL mode to `verify-full` for datasource `billing` in production."

Expected behavior:

- Validates compatible SSL configuration fields.
- Stops and flags missing certificate/trust config if required.

## Prompt 4: Verify Final State

"Change `orders-dev` port from `5432` to `6432` and confirm connection settings."

Expected behavior:

- Backs up first.
- Applies only port change.
- Confirms final host/port/database/auth/SSL snapshot.
