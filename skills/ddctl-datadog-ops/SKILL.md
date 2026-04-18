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

### Step 1 — One-time setup: extract cookies from Chrome

**Critical: you must copy a cURL from a POST request on the Logs Explorer page.**  
Not all requests carry the full auth cookies. GET requests (e.g. feature flags, settings pages) are
missing `dd_csrf_token` and possibly `DD_S`, which will cause HTTP 401 on logs queries.

1. Open Chrome and navigate to https://app.datadoghq.com/logs (the **Logs Explorer**).
2. Open DevTools → **Network** tab.
3. Wait for the page to load fully so that log queries fire.
4. In the Network filter box, type `logs-analytics` to find the right request.
   - Look for a **POST** to `/api/v1/logs-analytics/list?type=logs`.
   - If you see it, right-click → **Copy** → **Copy as cURL (bash)** — this is the ideal request.
   - If you can't find `logs-analytics`, any POST to `app.datadoghq.com` from the Logs Explorer page works.
5. Paste the cURL into the chat (as a code block is fine).

**Why this specific request?**
The browser's Logs Explorer sends `dd_csrf_token` in its cookie jar, which the init step needs.
This cookie is only present on pages that show interactive DataDog content — it won't appear
in requests made from settings pages or on initial page load.

**cURL format note:**
Chrome may produce either `-H 'Cookie: ...'` or `-b '...'` form — `ddctl init` handles both.
However, **shell-escaping a complex multi-line cURL** is error-prone. Prefer:
```
ddctl init --cookie '<paste just the cookie string here>'
```
To extract the cookie string: find the `-b '...'` or `Cookie:` value in the cURL and paste only that value.

### Step 2 — Initialize credentials

Run in the terminal (not the chat, to avoid shell escaping issues):
```
ddctl init --cookie '<cookie_string>'
```

Or, if the cURL shell-escaping works:
```
ddctl init --curl '<full cURL command>'
```

The skill can tell the user to run this command; it cannot execute interactive terminal commands itself.

### Step 3 — Verify

```
ddctl doctor
```

Expected output: `credentials found: true`, `datadog reachable: true`.
If `datadog reachable: false` or you get HTTP 401, the cookies are expired — go back to Step 1.

### Step 4 — Query logs

```
ddctl logs-query --query "service:<name> status:error" --from now-1h
ddctl logs-query --query "*" --from now-4h --limit 50 --json
```

Supported `--from`/`--to` formats: `now`, `now-1h`, `now-30m`, `now-2d`, Unix milliseconds, RFC3339.

### Step 5 — Iterate and summarize

Refine the query based on results; summarize findings with timestamps, services, and log lines.

## Validation

- `ddctl doctor` shows `credentials found: true` and `datadog reachable: true`.
- `ddctl logs-query --query "*" --limit 1` returns at least one log event or empty result without error.

## Known obstacles and workarounds

### HTTP 401 from logs-query even after a successful doctor

`ddctl doctor` only checks GET reachability. The logs query uses a POST endpoint that requires
`dd_csrf_token` cookie. If that cookie is absent, the POST returns 401.

**Fix:** Re-run `ddctl init` with a cURL copied from the Logs Explorer page (not any other page).

### Chrome HAR exports strip cookies (do not use HAR files for init)

Chrome's "Save all as HAR" redacts all `Cookie:` headers from the export for privacy.
Even if you export a HAR file with dozens of requests, all cookie fields will be empty.

**Only "Copy as cURL" from a single request preserves the cookie string.**

### ddctl init --curl hangs in the terminal

If the pasted cURL contains single quotes inside a single-quoted shell argument, the shell
treats the command as incomplete and hangs.

**Workaround:** Extract the cookie string from the cURL and pass it directly:
```
ddctl init --cookie '<extracted_cookie_string>'
```

### Ideal request to copy is /api/v1/logs-analytics/list

This is the exact endpoint `ddctl logs-query` calls. Copying a cURL from this request guarantees:
- All required cookies are present (`dd_csrf_token`, `DD_S`, `dogweb`, `_dd_s_v2`, etc.)
- The CSRF token is visible in the request body as `_authentication_token` (for debugging)
- The cookie string is confirmed to be fresh and working

## Safety

- **Read-only**: `ddctl` performs no writes or mutations in DataDog.
- Stop and report to the user if DataDog returns authentication errors (HTTP 401/403).
- Do not store or log raw cookie values.
- Do not use `ddctl` to access DataDog data outside the scope authorized for the current session.
- Cookie strings contain full session credentials — do not paste them into public channels.

## References

- `tools/ddctl/README.md`
- `docs/CONVENTIONS.md`
