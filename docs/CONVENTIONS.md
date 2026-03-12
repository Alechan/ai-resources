# Conventions

## Naming Rules

- Use `kebab-case` for directories, resource names, and markdown filenames.
- Use `SKILL.md` as the canonical filename for each skill definition.
- Use `<agent-name>.md` for agent files in `agents/`.
- Use sentence case section headers (for example: `## Operating Procedure`).

## Required Sections

Every `SKILL.md` must include:

1. `## Purpose`
2. `## When To Use`
3. `## Inputs`
4. `## Workflow`
5. `## Validation`
6. `## Safety`
7. `## References`

Every agent file in `agents/` must include:

1. `## Role`
2. `## Scope`
3. `## Required Context`
4. `## Operating Procedure`
5. `## Safety Guardrails`
6. `## Output Format`
7. `## Validation Checklist`

## Safety Requirements

- Do not perform destructive actions without explicit user intent.
- Always propose and confirm risky operations (deletes, irreversible edits, production-impacting changes) before execution.
- Apply only user-requested changes; do not introduce hidden scope.

## Install Path Conventions

- Codex skill target path: `~/.codex/skills/<skill-name>/SKILL.md`
- Claude agent target path: `~/.claude/agents/<agent-name>.md`
- Installers should default to symlink mode for easier updates from this repository.
