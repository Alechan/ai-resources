---
name: jenkinsctl
description: Interact with Jenkins instances using the jenkinsctl CLI. Use when checking build status, listing jobs, or verifying Jenkins connectivity.
---

# jenkinsctl

## Purpose

Read-only CLI tool for interacting with Jenkins instances. Provides a lightweight interface for listing jobs, checking connectivity, and retrieving build statuses.

---

## Installation

The tool source is at `tools/jenkinsctl/src`. Build with:

```bash
cd tools/jenkinsctl/src
go build -o jenkinsctl ./cmd/jenkinsctl/
```

Or install to `$GOPATH/bin`:

```bash
cd tools/jenkinsctl/src
go install ./cmd/jenkinsctl/
```

---

## Required Flags

All credentials and connection details are passed via command-line flags:

| Flag | Short | Description |
|------|-------|-------------|
| `--url` | `-u` | Jenkins instance URL |
| `--user` | | Jenkins username |
| `--token` | | Jenkins API token |

All three flags are required for every command.

### Recommended usage

Pass credentials via environment variables at the call site to avoid them appearing in shell history:

```bash
jenkinsctl --url "$JENKINS_URL" --user "$JENKINS_USERNAME" --token "$JENKINS_TOKEN" doctor
```

### Recommended environment variables

Set these in a sourced file (e.g. `~/.keys/tokens.zsh`):

| Variable | Description |
|----------|-------------|
| `JENKINS_USERNAME` | Your Jenkins username (shared across instances) |
| `JENKINS_CI_URL` | CI instance URL (builds, PR checks) |
| `JENKINS_CI_TOKEN` | API token for CI instance |
| `JENKINS_ACCEPTANCE_URL` | Deploy instance URL (dev/test/acceptance) |
| `JENKINS_ACCEPTANCE_TOKEN` | API token for deploy instance |
| `JENKINS_PROD_URL` | Production deploy instance URL |
| `JENKINS_PROD_TOKEN` | API token for production instance |

The CLI does **not** read environment variables directly. This keeps the tool generic and stateless.

### Token rotation

Jenkins API tokens may expire frequently. If you get a `401` error, regenerate the token:

1. Go to your Jenkins user configuration page: `<JENKINS_URL>/user/<YOUR_USERNAME>/configure`
2. Scroll to the **API Token** section
3. Click **Add new Token**, give it a name, click **Generate**
4. Copy the token and update the corresponding environment variable

Note: different Jenkins instances require separate tokens. A token generated on one instance will not work on another.

---

## Commands

### Connectivity Check

```bash
jenkinsctl --url "$JENKINS_URL" --user "$JENKINS_USERNAME" --token "$JENKINS_TOKEN" doctor
```

### List Jobs

```bash
jenkinsctl --url "$JENKINS_URL" --user "$JENKINS_USERNAME" --token "$JENKINS_TOKEN" job list
```

### Build Status

```bash
jenkinsctl --url "$JENKINS_URL" --user "$JENKINS_USERNAME" --token "$JENKINS_TOKEN" build status <JOB_NAME>
```

**Job name format for nested jobs:** Use `/job/` as the path separator.

```bash
# Nested job (folder/pipeline)
jenkinsctl --url "$JENKINS_URL" --user "$JENKINS_USERNAME" --token "$JENKINS_TOKEN" build status "myservice/job/pre-prod-pipeline"
```

**Output format:**

```
state=succeeded build=42
state=failed build=43 url=https://jenkins.example.com/job/myservice/job/pre-prod-pipeline/43/
state=running build=44 url=https://jenkins.example.com/job/myservice/job/pre-prod-pipeline/44/
```

- `state` values: `succeeded`, `failed`, `aborted`, `unstable`, `running`, `queued`, `blocked`, `unknown`
- `url` is omitted when `state=succeeded` (no action needed)

---

## Error output

Errors use structured `key=value` format:

```
# Auth failure (401/403)
kind=auth status=401 url=https://jenkins.example.com/... auth_context=acceptance hint=verify --url points to the intended Jenkins instance and regenerate the token for that same instance

# Job not found
kind=not_found status=404 url=https://jenkins.example.com/job/myservice/ hint=check the job path and use '/job/' separators for nested folders (e.g. folder/job/pipeline)

# SSO redirect
kind=redirect status=302 url=https://jenkins.example.com/... location=https://sso.example.com/login hint=use the final Jenkins base URL directly (avoid SSO/login redirect endpoints)
```

When `kind=auth`, regenerate the token for the specific Jenkins instance in `--url`. Tokens are not shared across instances.

---

## Shell aliases (optional)

For convenience, define aliases that wire up instance-specific credentials:

```bash
alias jci='jenkinsctl --url "$JENKINS_CI_URL" --user "$JENKINS_USERNAME" --token "$JENKINS_CI_TOKEN"'
alias jacc='jenkinsctl --url "$JENKINS_ACCEPTANCE_URL" --user "$JENKINS_USERNAME" --token "$JENKINS_ACCEPTANCE_TOKEN"'
alias jprod='jenkinsctl --url "$JENKINS_PROD_URL" --user "$JENKINS_USERNAME" --token "$JENKINS_PROD_TOKEN"'
```

Then usage becomes:

```bash
jci doctor
jci build status "myservice/job/pre-prod-pipeline"
jacc build status "Deploy"
```
