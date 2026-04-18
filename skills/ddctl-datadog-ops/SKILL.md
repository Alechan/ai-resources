# ddctl-datadog-ops

## Purpose

Provide a repeatable procedure for querying DataDog logs and checking DataDog health via `ddctl`.

## When To Use

- Querying logs with a search filter (service, status, environment, custom attributes).
- Checking DataDog reachability and verifying session cookie authentication.
- Iterative log investigation: narrowing down a time window or refining a query based on results.
- Confirming DataDog connectivity before beginning a deeper investigation.

## Inputs

- **cURL command from Chrome DevTools** (for init), or confirmation that `ddctl init` was already run.
- **Query string**: DataDog log search syntax (e.g. `service:my-service status:error env:prod`).
- **Time range**: relative (e.g. `now-1h`, `now-4h`) or ISO-8601 timestamps.
- **Output format**: text (default) or `--json` for structured output.
- **Site**: DataDog site domain (default: `datadoghq.com`; override with `--site` or `DDCTL_SITE`).

## Workflow

1. **One-time setup — extract cookies from Chrome:**
   a. Open Chrome and log in to https://app.datadoghq.com
   b. Open DevTools (Cmd+Option+I) → Network tab
   c. Filter by "Fetch/XHR", then reload the page or click any DataDog UI element
   d. Right-click any request to app.datadoghq.com → Copy → **Copy as cURL**
   e. Paste the cURL into the chat; the skill will run `ddctl init --curl '<pasted cURL>'`

2. Verify setup: `ddctl doctor`

3. Query logs: `ddctl logs-query --query "<filter>" --from now-1h`

4. Narrow and iterate based on results.

5. Summarize findings with timestamps, services, and relevant log lines.

## Validation

- `ddctl doctor` shows `credentials found: true` and `datadog reachable: true`.
- `ddctl logs-query --query "*" --limit 1` returns at least one log event or an empty result without error.

## Safety

- **Read-only**: `ddctl` performs no writes or mutations in DataDog.
- Stop and report to the user if DataDog returns authentication errors (HTTP 401/403).
- Do not store or log raw cookie values.
- Do not use `ddctl` to access DataDog data outside the scope authorized for the current session.

## References

- `tools/ddctl/README.md`
- `docs/CONVENTIONS.md`
