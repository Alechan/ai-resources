# GitHub Copilot credits tool (`gh-copilot-credits`)

`gh-copilot-credits` is a local-only CLI for observing the current GitHub
Copilot AI-credit quota through the internal account endpoint used by Copilot
IDE integrations. It delegates authentication to the local `gh` CLI.

It does not run Copilot requests, read browser cookies, save raw API responses,
or collect project/repository metadata.

## Install

```bash
go install github.com/Alechan/ai-resources/tools/gh-copilot-credits/src/cmd/gh-copilot-credits@latest
gh-copilot-credits --help
```

For local development:

```bash
cd tools/gh-copilot-credits/src
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/gh-copilot-credits
```

Install the binary from the current checkout:

```bash
cd tools/gh-copilot-credits/src
go install ./cmd/gh-copilot-credits
```

The tool requires an authenticated local `gh` session.

## Query the current quota

```bash
gh-copilot-credits current
gh-copilot-credits current --json
```

The output focuses on the `premium_interactions` quota and includes the
observation timestamp, reset date, entitlement, remaining balance, and usage.

It also prints two passive burn-rate estimates:

- calendar-day rate and projected period total across every calendar day;
- weekday rate and projected period total across the actual local Monday-Friday
  dates in the billing period.

The weekday projection assumes zero weekend usage. Both estimates use the
endpoint's reported `credits_used` value and the quota reset date. The output
also shows the required calendar-day and workday pace for staying within the
entitlement.

The endpoint is internal and undocumented by GitHub. It may change. The tool
validates required fields and fails rather than writing an untrusted snapshot.

## Append a local snapshot

```bash
gh-copilot-credits current append \
  --csv "$HOME/Library/Application Support/gh-copilot-credits/usage.csv"
```

The command creates the parent directory and CSV header as needed. It appends
only filtered quota fields. The default CSV does not contain the GitHub login,
organization list, analytics identifiers, raw response, cookies, tokens,
project names, branches, repositories, prompts, or source code.

The CSV is low direct PII but remains work-account operational data. Keep it
local and outside the repository by default.

## Diagnose access

```bash
gh-copilot-credits doctor
```

This checks the `gh` executable, the endpoint, authentication indirectly via a
successful read, the quota shape, numeric fields, and the reset date.

## Manage an optional launchd job

The scheduled action is:

```bash
gh-copilot-credits current append --csv PATH
```

Print a per-user LaunchAgent plist without changing the system:

```bash
gh-copilot-credits schedule print \
  --csv "$HOME/Library/Application Support/gh-copilot-credits/usage.csv" \
  --interval 6h
```

Install it explicitly:

```bash
gh-copilot-credits schedule install \
  --csv "$HOME/Library/Application Support/gh-copilot-credits/usage.csv" \
  --interval 6h
```

Inspect status without changing it:

```bash
gh-copilot-credits schedule status
```

Remove only this tool's LaunchAgent:

```bash
gh-copilot-credits schedule uninstall --yes
```

Installation is never automatic. The default schedule is every six hours.

## Interpretation

This tool provides the account-level meter. It does not attribute credits to a
specific harness or model.

For a complete analysis, combine its CSV with native Copilot event-file deltas,
OpenCode `session_v2` aggregates, and known external usage such as
administrator-reported Code Review credits. Keep the residual visible.

## Safety

- Authentication is delegated to `gh`; the tool never asks for a token.
- The tool invokes `gh` without a shell.
- Raw endpoint responses are never stored.
- Failed or schema-invalid queries do not append CSV rows.
- No cloud upload or Google Sheets integration is included.
- `schedule install` and `schedule uninstall` are explicit operations.
- `schedule print` and `schedule status` are read-only.
