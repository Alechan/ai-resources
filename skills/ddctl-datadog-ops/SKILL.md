# ddctl-datadog-ops

## Purpose

Provide a repeatable procedure for querying DataDog logs and checking DataDog health via `ddctl`.

## When To Use

- Querying logs with a search filter (service, status, environment, custom attributes).
- Checking DataDog reachability and verifying Chrome cookie authentication.
- Iterative log investigation: narrowing down a time window or refining a query based on results.
- Confirming DataDog connectivity before beginning a deeper investigation.

## Inputs

- **Query string**: DataDog log search syntax (e.g. `service:my-service status:error env:prod`).
- **Time range**: relative (e.g. `now-1h`, `now-4h`) or ISO-8601 timestamps.
- **Output format**: text (default) or `--json` for structured output.
- **Site**: DataDog site domain (default: `datadoghq.com`; override with `--site` or `DDCTL_SITE`).

## Workflow

1. Run `ddctl doctor` to verify Chrome cookies are present and DataDog is reachable.
   - If `cookies file found: false`, ensure Chrome has been used to visit `app.datadoghq.com`.
   - If `session cookies: 0`, refresh the session by visiting DataDog in Chrome.
   - If `datadog reachable: false`, check network connectivity.
2. Run `ddctl logs-query --query "..." --from now-1h --to now` to fetch recent logs.
3. Review results; narrow the query or time range as needed and iterate.
4. Use `--json` for machine-readable output when post-processing results.
5. Summarize findings: notable patterns, error counts, relevant log lines.

## Validation

- `ddctl --help` executes without error.
- `ddctl doctor` reports `cookies file found: true` and `datadog reachable: true`.
- `ddctl logs-query --query "*" --limit 1` returns at least one log event or an empty result without error.

## Safety

- **Read-only**: `ddctl` performs no writes or mutations in DataDog.
- Stop and report to the user if DataDog returns authentication errors (HTTP 401/403).
- Do not store or log raw cookie values.
- Do not use `ddctl` to access DataDog data outside the scope authorized for the current session.

## References

- `tools/ddctl/README.md`
- `docs/CONVENTIONS.md`
