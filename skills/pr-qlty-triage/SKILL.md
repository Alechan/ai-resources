---
name: pr-qlty-triage
description: Triage qlty.sh code quality findings for a pull request. Use when checking qlty issues on a PR, deciding which findings to fix vs ignore, and presenting findings to the user.
---

# pr-qlty-triage

## Purpose

Retrieve and triage qlty.sh findings for a PR, distinguish pre-existing issues from new ones, and decide what to fix.

---

## Getting the findings

### If authenticated

Navigate to:
```
https://qlty.sh/gh/<org>/projects/<repo>/pull/<PR_NUMBER>/issues
```

### If unauthenticated (common in CI or agent context)

Ask the user to download the page HTML (`Save As…` in browser) to the repo's `tmp/` folder. Then extract text:

```bash
cat "tmp/qlty-pr-<NUMBER>.html" | python3 -c "
import sys
from html.parser import HTMLParser

class T(HTMLParser):
    def __init__(self):
        super().__init__()
        self.text = []
        self.skip = False
    def handle_starttag(self, tag, attrs):
        if tag in ('script','style','svg'): self.skip = True
    def handle_endtag(self, tag):
        if tag in ('script','style','svg'): self.skip = False
    def handle_data(self, data):
        if not self.skip:
            s = data.strip()
            if s: self.text.append(s)

p = T()
p.feed(sys.stdin.read())
print('\n'.join(p.text))
"
```

---

## Triage table

Present all findings to the user in this format before touching any code:

| # | File | Line | Rule | Finding | Pre-existing? | Recommend |
|---|---|---|---|---|---|---|
| 1 | `renderer_test.go` | 46 | `S1192` | Duplicate literal `"payment-instructions"` (13×) | No | Fix — extract constant |
| 2 | `renderer_test.go` | 218 | `S100` | Function name has underscores | No | Ignore — valid Go test convention |
| 3 | `order.yml` | — | `yamllint:document-start` | Missing `---` | No | Fix — trivial |

**Always check pre-existing before recommending a fix.** Fixing pre-existing issues in a feature PR adds noise and makes diffs harder to review.

---

## Checking if a finding is pre-existing

```bash
# Does the flagged line appear in the diff introduced by this branch?
git diff main -- <file> | grep "<flagged string or pattern>"

# If the output has a leading '+', it's new (our change).
# If there's no output, the issue existed before us — don't fix it in this PR.
```

---

## Rule-by-rule guidance

### `yamllint:document-start`
**Fix.** Add `---` as the first line of every YAML file. Trivial, zero-risk.

```yaml
---
request:
  ...
```

### `yamllint:line-length`
**Fix if easy.** In YAML `body: >` (folded) blocks, splitting a long line at the same indentation level is safe — the fold replaces the newline with a space, keeping the JSON valid. In block scalars, YAML comments (`#`) are not available mid-block; add a top-of-file `# yamllint disable-line` only at the document level if splitting is not practical.

### `S1192` — duplicate string literal
**Fix if it's our code.** Extract into a package-level constant (test files: alongside other test constants). See [`go-tdd-workflow`](../go-tdd-workflow/SKILL.md) for the constant pattern.

**Skip if pre-existing.** Filing it as a separate refactor PR is fine.

### `S100` — function naming (underscores)
**Ignore.** Go's own testing convention uses `TestFoo_Scenario` names. radarlint's `S100` rule is misconfigured for Go test files. Add `//nolint:revive` only if CI blocks on it; otherwise leave the name as-is.

### `unparam` — parameter always receives the same value
**Usually ignore.** The linter only sees current call sites. If the parameter is intentionally generic (designed for future call sites, or the function is exported), add:

```go
//nolint:unparam // <param> is always X today but kept generic for future callers
func myFunc(param string) string {
```

**Fix** only if the parameter genuinely will never vary (e.g. a private helper that could be simplified to a constant).

---

## Applying fixes

1. Fix trivial issues (whitespace, YAML `---`, constants) first — they have zero risk.
2. For logic issues flagged by qlty: use TDD (see [`go-tdd-workflow`](../go-tdd-workflow/SKILL.md)).
3. Run tests after each change:
   ```bash
   go test ./...
   ```
4. Commit fixes before pushing to avoid a pile of fixup commits (see [`lint-before-push`](../lint-before-push/SKILL.md)).
