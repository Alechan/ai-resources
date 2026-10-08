# Skills Loading Guide

## Codex skill (this repo)

Skill path:

- `skills/gdrivectl-drive-ops/SKILL.md`

Recommended trigger phrase in chat:

- "Use the gdrivectl-drive-ops skill"

Operational standard:

- Prefer installed binary invocation (`gdrivectl ...`).
- Use source fallback (`go run ./cmd/gdrivectl ...`) only when binary is missing and you are in the gdrivectl repo root.

## Claude subagent (this repo)

Project subagent path:

- `.claude/agents/gdrivectl-drive-ops.md`

Claude can auto-delegate when task matches, or you can ask explicitly:

- "Use the gdrivectl-drive-ops subagent"

Operational standard is the same:

- binary-first, source fallback only when needed.

## Validation prompts

- `skills/gdrivectl-drive-ops/evals/v1-prompts.md`
