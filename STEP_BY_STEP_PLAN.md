# AI Resources Bootstrap Plan (Standalone)

Status: Draft  
Last updated: 2026-03-11

## Purpose

Bootstrap this repository as a single source of truth for reusable AI resources (skills, agents, tool wrappers, playbooks, and installers).

## Assumptions

- This repo currently has no users.
- Backward compatibility is not required yet.
- You can reorganize structure now without migration constraints.
- `gdrivectl` is maintained directly in this repository (under `tools/gdrivectl/src`).

## End State (Definition)

By the end, this repo has:

- a clear folder structure
- onboarding docs (`README.md`, `AGENTS.md`)
- conventions and resource catalog
- install scripts for Codex/Claude resources
- initial `gdrivectl` resource entry
- initial DataGrip skill scaffold
- basic validation checks

## Target Structure

```text
.
├── README.md
├── AGENTS.md
├── CHANGELOG.md
├── docs/
│   ├── CONVENTIONS.md
│   └── RESOURCE_CATALOG.md
├── skills/
│   ├── gdrivectl-drive-ops/
│   │   └── SKILL.md
│   └── datagrip-datasources/
│       ├── SKILL.md
│       └── evals/
│           └── v1-prompts.md
├── agents/
│   ├── gdrivectl-drive-ops.md
│   └── datagrip-datasources.md
├── tools/
│   └── gdrivectl/
│       └── README.md
├── playbooks/
│   └── datagrip-datasource-update.md
└── scripts/
    ├── install_codex_skill.sh
    ├── install_claude_agent.sh
    └── verify_repo.sh
```

## Step-by-Step Execution

1. Initialize repo metadata files

Create:

- `README.md`: purpose + directory map + quick start
- `AGENTS.md`: rules for future chats in this repo (what to read first, where skills live)
- `CHANGELOG.md`: start with `Unreleased`

2. Create directory skeleton

From repo root:

```bash
mkdir -p docs skills/gdrivectl-drive-ops skills/datagrip-datasources/evals agents tools/gdrivectl playbooks scripts
```

3. Write conventions

Create `docs/CONVENTIONS.md` with:

- naming rules (`kebab-case`, file names, section headers)
- required sections for `SKILL.md` and agent files
- safety requirements (no destructive action without explicit user intent)
- install path conventions for Codex and Claude

4. Create resource catalog

Create `docs/RESOURCE_CATALOG.md` with a table:

- `name`
- `type` (skill/agent/tool/playbook)
- `status` (draft/active)
- `owner`
- `install target`
- `validation command`
- `source/canonical repo`

5. Integrate gdrivectl resource

Create:

- `tools/gdrivectl/README.md` with:
  - install command: `go install github.com/Alechan/ai-resources/tools/gdrivectl/src/cmd/gdrivectl@latest`
  - verification: `gdrivectl --help`
  - known troubleshooting pointers
- `skills/gdrivectl-drive-ops/SKILL.md` and `agents/gdrivectl-drive-ops.md` (copy or adapt from canonical source)

6. Add DataGrip datasource resource scaffold

Create:

- `skills/datagrip-datasources/SKILL.md`
- `agents/datagrip-datasources.md`
- `playbooks/datagrip-datasource-update.md`
- `skills/datagrip-datasources/evals/v1-prompts.md`

Minimum required behavior:

- backup/export current datasource config first
- validate intended changes before apply
- apply only explicit user-requested changes
- verify resulting connection settings

7. Add installer scripts

Create:

- `scripts/install_codex_skill.sh`
  - inputs: skill name + source path
  - target: `~/.codex/skills/<skill>/SKILL.md`
- `scripts/install_claude_agent.sh`
  - inputs: agent file + destination name
  - target: `~/.claude/agents/<agent>.md`

Script behavior requirements:

- `set -euo pipefail`
- validate source exists
- create destination folders if missing
- install via symlink by default (`ln -sf`)
- print final installed path

8. Add repository verification script

Create `scripts/verify_repo.sh` to check:

- required files/dirs exist
- every skill has matching catalog entry
- every agent has matching catalog entry
- installer scripts are executable

9. Make new-chat onboarding explicit

In `README.md` and `AGENTS.md`, add:

- “Start here” order: `README.md` -> `docs/CONVENTIONS.md` -> `docs/RESOURCE_CATALOG.md`
- instruction that new chats should read those files first

10. Validate and commit

Run:

```bash
chmod +x scripts/*.sh
bash scripts/verify_repo.sh
git add .
git commit -m "chore: bootstrap ai-resources repository structure"
```

## Quick Acceptance Checklist

- [ ] `README.md` exists and explains repo purpose
- [ ] `AGENTS.md` exists and is actionable
- [ ] `docs/CONVENTIONS.md` exists with concrete rules
- [ ] `docs/RESOURCE_CATALOG.md` exists and has initial entries
- [ ] `gdrivectl` skill + agent + tool README are present
- [ ] DataGrip skill scaffold exists
- [ ] install scripts exist and are executable
- [ ] `scripts/verify_repo.sh` passes

## Important Note

This file is the execution plan. A new chat will only use it automatically if your repo onboarding points to it (via `README.md` and `AGENTS.md`).
