# AI Resources

Reusable AI resources for Codex and Claude: skills, agents, tool wrappers, playbooks, and installers.

This repository is also the canonical home for `gdrivectl` source code at `tools/gdrivectl/src`.

## Start Here

New chats in this repository should read these files first, in order:

1. `README.md`
2. `docs/CONVENTIONS.md`
3. `docs/RESOURCE_CATALOG.md`

## Directory Map

```text
.
├── README.md
├── AGENTS.md
├── CHANGELOG.md
├── docs/
├── skills/
├── agents/
├── tools/
├── playbooks/
└── scripts/
```

## Quick Start

1. Verify repository integrity:

```bash
bash scripts/verify_repo.sh
```

2. Build and verify local `gdrivectl` from this repo:

```bash
cd tools/gdrivectl/src
go test ./...
go install github.com/Alechan/ai-resources/tools/gdrivectl/src/cmd/gdrivectl@latest
gdrivectl --help
```

3. Install a Codex skill from this repo:

```bash
bash scripts/install_codex_skill.sh gdrivectl-drive-ops skills/gdrivectl-drive-ops/SKILL.md
```

4. Install a Claude agent from this repo:

```bash
bash scripts/install_claude_agent.sh agents/gdrivectl-drive-ops.md gdrivectl-drive-ops
```

## Maintenance

- Keep naming and file format rules in `docs/CONVENTIONS.md`.
- Register every resource in `docs/RESOURCE_CATALOG.md`.
- Add user-visible changes to `CHANGELOG.md` under `Unreleased`.
