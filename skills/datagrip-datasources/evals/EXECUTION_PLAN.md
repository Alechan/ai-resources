# DataGrip Evals Execution Plan

Status: In Progress  
Last updated: 2026-03-13

## Objective

Replace tabletop-only eval review with executed eval runs and deterministic pass/fail scoring.

## Scope

- Prompt set: `skills/datagrip-datasources/evals/v1-prompts.md` (prompts 1-8).
- Policy source of truth:
  - `skills/datagrip-datasources/SKILL.md`
  - `agents/datagrip-datasources.md`
  - `playbooks/datagrip-datasource-update.md`

## Execution Approach

1. Define scoring contract
   - Convert each prompt expected behavior into explicit assertions.
   - Mark each assertion as `required` or `advisory`.

2. Build deterministic fixtures
   - Create prompt input fixtures (including config context for negative cases).
   - Include at least one fixture per prompt.

3. Implement eval runner
   - Create a script to run all fixtures and capture model outputs.
   - Persist raw run artifacts to `skills/datagrip-datasources/evals/artifacts/`.

4. Implement scorer
   - Evaluate outputs against assertions and emit per-prompt pass/fail.
   - Emit machine-readable summary (`JSON`) and human summary (`Markdown`).

5. Publish executed results
   - Write `skills/datagrip-datasources/evals/v1-results-executed-<date>.md`.
   - Keep tabletop file as design-review history; do not overwrite it.

## Exit Criteria

- All 8 prompts have executed runs with stored raw artifacts.
- Every prompt has deterministic scoring output.
- Failures, if any, include exact missing assertions.
- Repository docs point to executed results as current status.

## Progress Snapshot

- Completed:
  - Assertion matrix (`assertions-v1.json`)
  - Fixture set (`fixtures-v1.json`)
  - Runner script (`scripts/run_datagrip_evals.py`) with dry-run artifact generation
  - Scorer script (`scripts/score_datagrip_evals.py`) with required-failure exit behavior
  - Executed result publication (`v1-results-executed-2026-03-13.md`)
  - CI non-blocking integration (`.github/workflows/ci.yml`)
  - CI blocking gate (`.github/workflows/ci.yml`)
- Next:
  - TODO: replace deterministic mock runner in CI with a real LLM/runtime runner
