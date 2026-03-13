# Next Steps Plan

Status: In Progress  
Last updated: 2026-03-13

## Goal

Complete post-bootstrap hardening now that `gdrivectl` is maintained in this repository.

## Completed

1. CI workflow added with exact Go pin `1.24.2` and checks:
   - `bash scripts/verify_repo.sh`
   - `cd tools/gdrivectl/src && go test ./...`

2. `gdrivectl` moved to monorepo-owned module path:
   - `module github.com/Alechan/ai-resources/tools/gdrivectl/src`
   - Internal imports updated to the new module path.

3. `tools/gdrivectl/src/go.mod` updated from `go 1.22` to `go 1.24` and validated with:
   - `cd tools/gdrivectl/src && go mod tidy`
   - `cd tools/gdrivectl/src && go test ./...`

4. DataGrip eval scaffold expanded with negative prompts for:
   - backup missing
   - scope creep
   - environment mismatch
   - missing SSL prerequisites

5. DataGrip v1 eval prompts reviewed and tightened; results captured in:
   - `skills/datagrip-datasources/evals/v1-results-2026-03-13.md`

6. DataGrip eval execution foundations implemented:
   - `skills/datagrip-datasources/evals/assertions-v1.json`
   - `skills/datagrip-datasources/evals/fixtures-v1.json`
   - `scripts/run_datagrip_evals.py`
   - `scripts/score_datagrip_evals.py`

7. DataGrip executed eval run published:
   - `skills/datagrip-datasources/evals/v1-results-executed-2026-03-13.md`
   - Artifacts: `skills/datagrip-datasources/evals/artifacts/v1-executed-20260313/`

8. DataGrip eval suite wired into CI as non-blocking:
   - `.github/workflows/ci.yml` runs eval runner + scorer with artifact upload.

9. DataGrip eval gate promoted to blocking in CI:
   - `.github/workflows/ci.yml` now fails PRs on required assertion failures.
10. First-push whole-repo review completed:
   - `docs/FIRST_PUSH_REVIEW.md`
   - Task tracker: `docs/FIRST_PUSH_TASKS.md`

## Remaining Plan

1. Execute DataGrip evals with a real harness (replace tabletop-only status):
   - Plan: `skills/datagrip-datasources/evals/EXECUTION_PLAN.md`
   - Tasks: `skills/datagrip-datasources/evals/TASKS.md`
   - Result target: `skills/datagrip-datasources/evals/v1-results-executed-YYYY-MM-DD.md`
   - TODO: replace deterministic mock runner in CI with a real LLM/runtime runner.

2. Keep first-push review artifacts in sync if additional pre-push changes are introduced:
   - `docs/FIRST_PUSH_TASKS.md`
   - `docs/FIRST_PUSH_REVIEW.md`
