---
name: go-tdd-workflow
description: Red-green-refactor TDD workflow for Go. Use when asked to fix a bug or add a feature with tests first, or when writing table-driven tests, test helpers, or shouldContain/shouldNotContain assertion patterns.
---

# go-tdd-workflow

## Purpose

Write Go tests before code. Confirm the test fails for the right reason, then implement the fix, then confirm green.

---

## The cycle

```
1. Write the failing test  →  go test ./... (confirm RED, right error message)
2. Write the minimal fix   →  go test ./... (confirm GREEN)
3. Refactor                →  go test ./... (still GREEN)
```

Never skip step 1. A test that was never red proves nothing.

---

## Table-driven tests

Prefer table-driven tests for any function with multiple input/output cases.

```go
func TestResolvePaymentInstructions(t *testing.T) {
    tests := []struct {
        name       string
        methodCode string
        lang       string
        want       string
    }{
        {
            name:       "known method returns instructions",
            methodCode: "cc_shop_login",
            lang:       "en",
            want:       "some expected text",
        },
        {
            name:       "unknown method returns empty",
            methodCode: "unknown_method",
            lang:       "en",
            want:       "",
        },
        {
            name:       "uppercase method code normalised",
            methodCode: "CC_SHOP_LOGIN",
            lang:       "en",
            want:       "some expected text",
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            got := resolvePaymentInstructions(tc.methodCode, tc.lang)
            if got != tc.want {
                t.Errorf("resolvePaymentInstructions(%q, %q) = %q, want %q",
                    tc.methodCode, tc.lang, got, tc.want)
            }
        })
    }
}
```

---

## Test naming for specific bug regressions

When adding a test to cover a specific bug, name it descriptively using `_` to separate the scenario:

```go
func TestResolvePaymentInstructions_UppercaseMethodCode(t *testing.T) { ... }
func TestBuildPaymentVM_CardSuppressesInstructions(t *testing.T) { ... }
```

Note: `golangci-lint` rule `S100` (radarlint) flags underscores in function names. This is a misconfiguration — Go's own testing convention uses underscores in test names. Add `//nolint:funlen` or ignore the finding. Do **not** rename the test to satisfy the linter.

---

## Test helpers: panic, don't return empty

Test helpers that load fixtures or compute expected values should **panic** on error, not return empty or `t.Fatal`. Panics produce a clear stack trace at the call site and fail fast during package init (if called in struct literals).

```go
// ✅ correct — panics give clear attribution
func getExpectedInstructions(lang, methodCode string) string {
    data, err := emails.FS.ReadFile(lang + ".json")
    if err != nil {
        panic(fmt.Sprintf("getExpectedInstructions: could not read %s.json: %v", lang, err))
    }
    // ...
    raw, ok := translations[key]
    if !ok {
        panic(fmt.Sprintf("getExpectedInstructions: key %q not found in %s.json", key, lang))
    }
    return raw
}

// ❌ wrong — silent empty masks the real problem
func getExpectedInstructions(lang, methodCode string) string {
    data, err := emails.FS.ReadFile(lang + ".json")
    if err != nil {
        return ""
    }
    // ...
}
```

---

## shouldContain / shouldNotContain pattern

When testing rendered HTML output, avoid `strings.Contains` scattered across assertions. Use a struct field pattern:

```go
type renderCase struct {
    name            string
    // ... inputs ...
    shouldContain    []string
    shouldNotContain []string
}

// In the test loop:
for _, s := range tc.shouldContain {
    if !strings.Contains(got, s) {
        t.Errorf("case %q: expected output to contain %q\ngot: %s", tc.name, s, got)
    }
}
for _, s := range tc.shouldNotContain {
    if strings.Contains(got, s) {
        t.Errorf("case %q: expected output NOT to contain %q\ngot: %s", tc.name, s, got)
    }
}
```

This makes test cases self-documenting and easy to extend with new assertions.

---

## Constants for repeated test literals

If `golangci-lint` / qlty flags a string literal used 3+ times in tests (`S1192`), extract it as a package-level constant alongside other test constants:

```go
const (
    langEN                      = "en"
    methodCCShopLogin           = "cc_shop_login"
    cssClassPaymentInstructions = "payment-instructions"
)
```

---

## Running tests

```bash
# Run all tests in the package under development
go test ./app/my/package/...

# Run a single test by name
go test -run TestResolvePaymentInstructions_UppercaseMethodCode ./app/my/package/...

# Verbose output (see t.Log, t.Error details)
go test -v -run TestRenderPaymentHTML ./app/my/package/...

# Race detector (always use for concurrent code)
go test -race ./...
```

---

## Confirm RED before fixing

When adding a regression test, confirm it actually fails **before** applying the fix:

```bash
go test -run TestMyNewTest ./... 2>&1
# Should print: FAIL (with the expected error message)
# If it passes immediately — the test is wrong or the bug is already fixed
```

Only proceed to the fix once you've seen the failure.
