# Contributing

## Development setup

1. Install Go 1.22+.
2. Clone the repository.
3. Run `go build ./...`.
4. Run `go test ./...`.

## Code standards

- Format with `gofmt`.
- Keep changes scoped and incremental.
- Preserve command behavior and exit code contract.

## Pull requests

- Include tests for behavior changes.
- Update docs (`README.md`, `docs/TEST_PLAN.md`, `docs/STATUS.md`) when command behavior changes.
- Describe what changed and why.

## Reporting issues

- Include reproduction steps.
- Include command, flags, and expected/actual output.
