# DataGrip Eval Tasks

Status date: 2026-03-13

## Task List

- [x] `DG-EVAL-01` Define assertion matrix for prompts 1-8
  - Output: `skills/datagrip-datasources/evals/assertions-v1.json`
  - Acceptance: every expected behavior maps to explicit assertion keys.
  - Completed: 2026-03-13

- [x] `DG-EVAL-02` Create fixture inputs for all prompts
  - Output: `skills/datagrip-datasources/evals/fixtures-v1.json`
  - Acceptance: includes backup-missing, scope-creep, env-mismatch, and missing-SSL cases.
  - Completed: 2026-03-13

- [x] `DG-EVAL-03` Implement runner script
  - Output: `scripts/run_datagrip_evals.py`
  - Acceptance: runs full suite and writes raw artifacts under `skills/datagrip-datasources/evals/artifacts/`.
  - Completed: 2026-03-13 (dry-run smoke executed to `/tmp/datagrip-eval-artifacts/local-smoke`)

- [x] `DG-EVAL-04` Implement scorer script
  - Output: `scripts/score_datagrip_evals.py`
  - Acceptance: produces per-prompt pass/fail JSON and non-zero exit on required assertion failure.
  - Completed: 2026-03-13 (validated with `--fail-on-required`, exit code `1` on required failures)

- [x] `DG-EVAL-05` Execute eval suite and publish results
  - Output: `skills/datagrip-datasources/evals/v1-results-executed-2026-03-13.md`
  - Acceptance: references raw artifact paths and includes failure details (if any).
  - Completed: 2026-03-13 (run id: `v1-executed-20260313`, required failures: `0`)

- [x] `DG-EVAL-06` Wire into CI (non-blocking first pass)
  - Output: CI step in `.github/workflows/ci.yml`
  - Acceptance: eval run + score jobs execute on PRs; may be warning-only initially.
  - Completed: 2026-03-13 (runner + scorer steps added with `continue-on-error: true` and artifact upload)

- [x] `DG-EVAL-07` Promote CI eval gate to blocking
  - Output: CI config update
  - Acceptance: PR fails when required eval assertions fail.
  - Completed: 2026-03-13 (`continue-on-error` removed; scorer step is now a blocking gate)

- [ ] `DG-EVAL-08` TODO: replace deterministic mock runner with real LLM runner
  - Output: CI runner command updates and runtime integration docs.
  - Acceptance: CI evals run against a real LLM/runtime while preserving scorer gate behavior.
