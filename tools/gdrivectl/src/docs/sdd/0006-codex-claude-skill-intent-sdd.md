# SDD-0006: Codex/Claude Skill Intent for gdrivectl

Status: Accepted  
Version: 0.1  
Last updated: 2026-03-05

## 1. Purpose

Define the intent and baseline architecture for a reusable skill that helps Codex and Claude agents operate `gdrivectl` safely and consistently for Google Drive/Docs workflows.

## 2. Context

This repository already has:

- a stable CLI command surface (`doctor`, `search`, `file-meta`, `doc-tabs`, `doc-export`)
- deterministic exit-code contract
- test coverage for core parsing and API mapping paths
- spec-driven docs under `docs/sdd`, `docs/rfc`, and `docs/adr`

The next step is a skill/subagent layer so agent workflows can reliably use this CLI in multi-step tasks.

## 3. Goals

- G1: Provide one reusable operational guide for agent-driven Drive/Docs tasks.
- G2: Preserve safety boundaries (read-first, explicit approval for high-impact actions).
- G3: Keep instructions concise and triggerable via clear metadata.
- G4: Make the artifact shareable in version control across local/dev/team contexts.
- G5: Support both Codex and Claude with minimal duplication.

## 4. Non-goals

- Replacing `gdrivectl` CLI behavior or API logic.
- Designing a generalized autonomous workflow engine.
- Supporting every possible Drive/Docs operation in v1 skill scope.

## 5. Best-practice constraints (from current docs)

- Skill metadata quality is critical (`name`, `description`, and trigger keywords drive invocation).
- Skills should be modular and focused on specific workflows, not broad catch-all behavior.
- Keep instructions concise and use progressive disclosure (metadata -> `SKILL.md` -> optional resources/scripts).
- For deterministic behavior, explicitly request skill use in prompts when needed.
- Restrict tools/permissions to what the workflow requires (least privilege).
- Keep skills/subagents in version control for team reuse.
- Treat skill content as privileged/untrusted until reviewed; gate high-impact actions with explicit approval.
- Use eval loops (with-skill vs without-skill baselines) for reliability before wider rollout.

## 6. Proposed design

### 6.1 Source strategy

Use two separate artifacts to reduce product-compatibility risk:

- Codex skill package: `skills/gdrivectl-drive-ops/SKILL.md`
- Claude subagent: `.claude/agents/gdrivectl-drive-ops.md`

These must share the same operational contract and command mapping.

### 6.2 Codex integration

- Use `skills/gdrivectl-drive-ops/SKILL.md` directly.
- Prefer explicit skill invocation in prompts for deterministic behavior.

### 6.3 Claude integration

- Provide `.claude/agents/gdrivectl-drive-ops.md` with:
  - focused role
  - triggerable description
  - least-privilege tools
  - same command/remediation contract as Codex skill

## 7. Functional requirements

- FR-1: Skill must include explicit "when to use" and "when not to use" rules.
- FR-2: Skill must codify required preflight (`doctor`, auth/scope checks).
- FR-3: Skill must encode command selection logic:
  - discovery/search -> `search`
  - metadata -> `file-meta`
  - tab structure -> `doc-tabs`
  - content export -> `doc-export`
- FR-4: Skill must include exit-code-aware retry/remediation flow.
- FR-5: Skill must require explicit user confirmation before destructive or high-impact steps (if future write operations are added).
- FR-5a: Skill should follow least-privilege-first execution posture where tooling supports sandboxing, and escalate only on deterministic failure criteria.
- FR-6: Skill must include a persistent-auth-failure branch for constrained/sandboxed execution:
  - if exit code `3` persists after scope re-auth
  - run token diagnostics (`gcloud auth print-access-token`; optional ADC check)
  - recommend escalated/unsandboxed retry when environment blocks token refresh.
- FR-6a: Skill must include explicit config-store failure handling under exit code `2`:
  - if error indicates gcloud auth/config store unavailability under sandbox constraints
  - rerun the same target command unsandboxed/escalated
  - report the before/after outcomes.
- FR-7: Skill must report escalation retries explicitly:
  - original command
  - failing exit code/category
  - escalated retry command
  - final outcome (including export output path for `doc-export`).
- FR-8: Skill must be invocation-mode aware:
  - prefer `gdrivectl ...` when binary exists in PATH
  - fallback to `go run ./cmd/gdrivectl ...` only when binary is unavailable and repo-root execution is possible
  - explicitly state which mode was used in responses.

## 8. Non-functional requirements

- NFR-1: Keep `SKILL.md` concise (operationally minimal, no redundant theory).
- NFR-2: Skill instructions must be auditable in repository history.
- NFR-3: Skill must be safe by default (read-only posture unless explicitly expanded).
- NFR-4: Skill updates should be validated with a small eval set before adoption.

## 9. Validation plan

- V1: Author 5-10 representative prompts for with-skill evaluation.
- V2: Run with-skill vs without-skill comparisons for correctness and command selection.
- V3: Check safety outcomes:
  - no unauthorized write actions
  - correct handling of auth/scope/network failures
- V4: Accept only if skill improves consistency without increasing harmful/tool misuse behavior.
- V5: Include constrained-environment eval cases where:
  - `doctor` and some read commands pass
  - export workflow fails with repeated auth classification due to runtime constraints
  - skill chooses escalation guidance rather than repetitive re-auth loops.
- V6: Include constrained-environment eval cases where:
  - command fails with config classification tied to gcloud config-store access
  - skill escalates from least-privilege mode to unsandboxed retry
  - final response records both attempts and rationale.

## 10. Deliverables

- D1: Codex skill under `skills/gdrivectl-drive-ops/`.
- D2: Claude subagent under `.claude/agents/`.
- D3: Eval prompt set under `skills/gdrivectl-drive-ops/evals/`.
- D4: Loading guide in `docs/SKILLS.md`.

## 11. Chosen defaults

1. Location: keep both artifacts in this repo for easy chat loading.
2. Scope: read-only v1 (`doctor/search/file-meta/doc-tabs/doc-export`).
3. Claude compatibility: separate subagent file (no forced shared runtime format).
4. Tool permissions (Claude): `Read`, `Grep`, `Glob`, `Bash`.
5. Activation: explicit trigger phrase recommended; auto-delegation allowed where platform supports it.
6. Evaluation: lightweight manual eval with 10 representative prompts.

## 12. References

- OpenAI Codex app announcement (skills overview): https://openai.com/index/introducing-the-codex-app/
- OpenAI Skills guide (API/docs): https://developers.openai.com/api/docs/guides/tools-skills
- OpenAI AGENTS.md guidance: https://developers.openai.com/codex/guides/agents-md
- OpenAI Codex usage best practices: https://openai.com/business/guides-and-resources/how-openai-uses-codex/
- Anthropic Claude subagents docs: https://docs.anthropic.com/en/docs/claude-code/sub-agents
- Agent Skills specification: https://agentskills.io/specification
- Agent Skills “what are skills”: https://agentskills.io/what-are-skills
- Agent Skills evaluation guidance: https://agentskills.io/skill-creation/evaluating-skills
