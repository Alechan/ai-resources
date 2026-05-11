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

## Required Environment Variables

| Variable | Description |
|----------|-------------|
| `JENKINS_USERNAME` | Your Jenkins username |
| `JENKINS_API_TOKEN` | Your personal Jenkins API token |

---

## Commands

### Connectivity Check

Verify connectivity to a Jenkins instance:

```bash
jenkinsctl doctor --url <JENKINS_URL>
```

### List Jobs

List all available jobs:

```bash
jenkinsctl job list --url <JENKINS_URL>
```

### Build Status

Retrieve the status of the last build for a specific job:

```bash
jenkinsctl build status <JOB_NAME> --url <JENKINS_URL>
```

**Job name format for nested jobs:** Use `/job/` as the path separator.

```bash
# Top-level job
jenkinsctl build status "<SERVICE>" --url $JENKINS_URL

# Nested job (folder/pipeline)
jenkinsctl build status "<SERVICE>/job/<PIPELINE>" --url $JENKINS_URL
```

---

## Examples

```bash
# Check if Jenkins is reachable
jenkinsctl doctor --url $JENKINS_URL

# List all jobs
jenkinsctl job list --url $JENKINS_URL

# Check the latest build of a pipeline
jenkinsctl build status "<SERVICE>/job/<PIPELINE>" --url $JENKINS_URL
# Output: Build #1: SUCCESS
```
