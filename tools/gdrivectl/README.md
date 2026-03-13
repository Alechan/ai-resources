# gdrivectl Tool Resource

`gdrivectl` is maintained in this repository under `tools/gdrivectl/src`.

## Build and Install

```bash
go install github.com/Alechan/ai-resources/tools/gdrivectl/src/cmd/gdrivectl@latest
```

Local contributor install from this repository:

```bash
cd tools/gdrivectl/src
go install ./cmd/gdrivectl
```

## Validate Source

```bash
cd tools/gdrivectl/src
go test ./...
```

## Verify

```bash
gdrivectl --help
```

## Deprecation Note

- The previous standalone `gdrivectl` repository is deprecated.
- This repository is now the canonical source for code, docs, and automation around `gdrivectl`.

## Troubleshooting

- `command not found`: ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `PATH`.
- Authentication errors: verify active Google credentials and required Drive permissions.
- API/network failures: retry with stable connectivity and inspect the command output for rate-limit or auth hints.
