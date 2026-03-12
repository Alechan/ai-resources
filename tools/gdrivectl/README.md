# gdrivectl Tool Resource

## Install

```bash
go install github.com/Alechan/gdrivectl/cmd/gdrivectl@latest
```

## Verify

```bash
gdrivectl --help
```

## Troubleshooting

- `command not found`: ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `PATH`.
- Authentication errors: verify active Google credentials and required Drive permissions.
- API/network failures: retry with stable connectivity and inspect the command output for rate-limit or auth hints.
