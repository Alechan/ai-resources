---
name: lint-before-push
description: Workflow for catching and fixing lint issues before pushing a branch or after a CI lint failure. Use when golangci-lint or yamllint findings need triaging, or to batch lint fixes into a clean commit.
---

# lint-before-push

## Purpose

Catch lint issues before they appear in CI, or triage them after a CI failure. Batch all fixes into a single commit so the PR history stays clean.

---

## Run lint locally first

```bash
# Go lint (requires golangci-lint installed)
golangci-lint run ./...

# Only new issues relative to main (mirrors CI behaviour)
golangci-lint run --new-from-rev=main ./...

# YAML lint
yamllint configs/
```

If golangci-lint is not installed locally, read the CI logs directly:

```bash
gh run view <run-id> --log-failed 2>&1 | grep -E "\.go:|\.yml:" | head -50
```

---

## Triage table

Before touching anything, build a table:

| # | File | Line | Rule | Finding | Fix? |
|---|---|---|---|---|---|
| 1 | `renderer.go` | 157 | `wsl_v5` | Missing blank line above `if` | Yes — trivial |
| 2 | `renderer_test.go` | 198 | `unparam` | param always receives same value | nolint with reason |
| 3 | `order.yml` | 1 | `yamllint:document-start` | Missing `---` | Yes — trivial |

Always confirm whether the flagged line is in your diff (`git diff main -- <file>`) before spending time on it.

---

## Rule reference

### `wsl_v5` — whitespace linter
Add a blank line before `if`, `for`, `return`, or assignment blocks when there are multiple statements above them.

```go
// ❌ wsl_v5 violation
instructionsKey := strings.ToLower(methodCode) + ".instructions"
instructions := ResolveByKey(instructionsKey, lang)
if instructions == instructionsKey {

// ✅ fixed
instructionsKey := strings.ToLower(methodCode) + ".instructions"

instructions := ResolveByKey(instructionsKey, lang)

if instructions == instructionsKey {
```

### `unparam` — parameter always receives the same value
The linter only sees current call sites. If the param is intentionally generic, suppress with a comment explaining why:

```go
//nolint:unparam // lang is always "en" today but kept generic for future locales
func resolveInstructions(methodCode, lang string) string {
```

### `S1192` — duplicate string literal
Extract into a named constant. In test files, place it alongside other test constants at the top of the file:

```go
const (
    cssClassPaymentInstructions = "payment-instructions"
    langEN                      = "en"
)
```

### `yamllint:document-start`
Add `---` as the very first line of every YAML file.

### `yamllint:line-length`
In YAML `body: >` (folded scalar) blocks, split the long line at the same indentation — the fold replaces the newline with a space, keeping JSON payloads valid:

```yaml
  body: >
    {
      "units": [{"unit_id": "unit-1", "location_code": "DE",
      "shop_item_source_location": "DE"}],
    }
```

### `revive` / `S100` — function naming (underscores in Go test names)
Ignore. `TestFoo_Scenario` is the accepted Go convention for sub-scenario test names. If CI enforces it, add `//nolint:revive`.

---

## Applying and committing

1. Fix trivial issues in bulk (whitespace, YAML headers, constants).
2. Run tests to confirm nothing broke:
   ```bash
   go test ./...
   ```
3. Commit all lint fixes in a single focused commit before pushing:
   ```bash
   git add -p   # review each hunk
   git commit -m "G2-XXXXX: fix golangci-lint and yamllint findings"
   ```

**Do not mix lint fixes with feature changes** in the same commit — it makes review harder and obscures the diff.

---

## After pushing — CI still fails?

If CI flags a finding that doesn't reproduce locally:

```bash
# Check the exact golangci-lint version used in CI
gh run view <run-id> --log-failed 2>&1 | grep "golangci-lint-version"

# Run locally with that version pinned
golangci-lint run --new-from-rev=main ./... # with matching version
```

CI uses `--new-from-issues` or `--new-from-rev` mode — it only reports findings on lines changed by the PR. If a finding appears in CI but not locally, confirm you're diffing against the same base ref.
