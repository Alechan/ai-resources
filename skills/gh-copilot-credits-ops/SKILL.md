---
name: gh-copilot-credits-ops
description: Safely query and append local GitHub Copilot AI-credit quota snapshots with gh-copilot-credits, including optional macOS launchd scheduling and reconciliation guidance.
---

# GitHub Copilot credits operations

## Purpose

Use the local `gh-copilot-credits` CLI to observe the current GitHub Copilot
AI-credit quota, append filtered snapshots to a local CSV, and manage an
explicit per-user macOS `launchd` schedule.

The tool provides an account-level meter. It does not attribute usage to native
Copilot, OpenCode, Cursor, models, or Code Review.

## When To Use

Use this skill when the user asks to check current credits, append a quota
snapshot, diagnose polling, manage the optional launchd job, or reconcile the
account-level series with native Copilot and OpenCode data.

Do not use it to run Copilot requests or perform usage experiments.

## Inputs

- The authenticated local `gh` CLI session;
- a command: `current`, `current append`, `doctor`, or `schedule`;
- an explicit CSV path for append/install operations;
- an optional interval, defaulting to six hours for launchd;
- local native Copilot and OpenCode paths when reconciliation is requested.

## Workflow

### Query current usage

```bash
gh-copilot-credits current
gh-copilot-credits current --json
```

Report the quota timestamp and reset date with the numbers. Do not call the
internal endpoint directly from a shell when the CLI is available.

The current table includes:

- calendar-day burn rate and projected billing-period total;
- actual local weekday burn rate and projected total;
- elapsed/total calendar days, weekdays, and weekends;
- required remaining pace to stay within entitlement.

The weekday projection assumes zero weekend usage. Treat both projections as
observational forecasts, not provider guarantees.

### Append a snapshot

```bash
gh-copilot-credits current append \
  --csv "$HOME/Library/Application Support/gh-copilot-credits/usage.csv"
```

Confirm success and report the path if needed. Do not upload the CSV or place it
in a cloud-synchronized directory by default.

### Diagnose access

```bash
gh-copilot-credits doctor
```

If it fails, report whether the failure is caused by a missing `gh` executable,
authentication, endpoint access, or a changed response schema. Never request or
display the user's token.

### Manage launchd

Print before installing:

```bash
gh-copilot-credits schedule print \
  --csv "$HOME/Library/Application Support/gh-copilot-credits/usage.csv" \
  --interval 6h
```

Install only after explicit user request:

```bash
gh-copilot-credits schedule install \
  --csv "$HOME/Library/Application Support/gh-copilot-credits/usage.csv" \
  --interval 6h
```

Inspect and remove:

```bash
gh-copilot-credits schedule status
gh-copilot-credits schedule uninstall --yes
```

### Reconcile sources

Use the account CSV as the provider-level series. Compare its deltas against:

```text
Native Copilot: ~/.copilot/session-state/*/events.jsonl
OpenCode:       ~/.local/share/opencode/opencode.db
```

Calculate account delta minus native delta, OpenCode delta, and known external
delta. Keep the residual visible.

## Validation

- Run `gh-copilot-credits doctor` before relying on a new installation.
- Confirm `current` reports `premium_interactions` and a reset date.
- Confirm `current append` creates a valid header and one new row.
- Confirm repeated appends do not duplicate the header.
- Confirm schema failures do not append rows.
- Confirm the CSV contains no login, organization, project, branch, raw response, token, or cookie fields.
- Run `schedule print` and review the plist before `schedule install`.
- After installation, run `schedule status`.
- Record the snapshot cutoff when reconciling sources.

## Safety

- Never ask for, print, store, or commit GitHub tokens or cookies.
- Never read browser cookie databases.
- Never save the raw `/copilot_internal/user` response.
- Treat the internal endpoint as unstable and fail closed on schema changes.
- Keep the CSV local and outside the repository by default.
- Do not add Google Sheets or external upload integrations without policy review.
- Do not collect project, branch, repository, prompt, or source-code data.
- Do not install a LaunchAgent automatically.
- Do not uninstall a LaunchAgent without explicit `--yes` confirmation.
- Do not infer harness or model attribution from account-level balance deltas.
- Do not run experiments merely to improve attribution.

## References

- [`tools/gh-copilot-credits/README.md`](../../tools/gh-copilot-credits/README.md)
- [`tools/gh-copilot-credits/src`](../../tools/gh-copilot-credits/src)
- [`copilot-insights-source-analysis.md`](../../tmp/ai_tools_usage_analysis/copilot-insights-source-analysis.md)
- [Copilot Insights source](https://github.com/kasuken/vscode-copilot-insights)
- [GitHub billing usage documentation](https://docs.github.com/en/enterprise-cloud@latest/rest/billing/usage)
- Validation: `cd tools/gh-copilot-credits/src && go test ./...`
