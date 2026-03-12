# AGENTS

## Start Here (Required)

Every new chat working in this repository must read these files first, in order:

1. `README.md`
2. `docs/CONVENTIONS.md`
3. `docs/RESOURCE_CATALOG.md`

## Operating Rules

- Use `docs/CONVENTIONS.md` for naming, file section requirements, and safety constraints.
- Keep reusable skills in `skills/<skill-name>/SKILL.md`.
- Keep Claude-facing agent specs in `agents/<agent-name>.md`.
- Keep tool wrappers and install notes in `tools/<tool-name>/README.md`.
- Keep operational runbooks in `playbooks/*.md`.
- Any new skill or agent must be added to `docs/RESOURCE_CATALOG.md`.
- Run `bash scripts/verify_repo.sh` after changes and before commit.
