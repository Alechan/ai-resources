---
name: gchatctl-conversation-ops
description: Route Google Chat read requests to gchatctl without a GCP project. Initialize from a local HAR or cURL, then list spaces, get a space, list members, search, export a permalink, or replay list_topics.
---

# gchatctl conversation operations

## Purpose

Use read-only `gchatctl` as the Chat UI for reads the CLI supports. Store a
browser session from a local capture. Never paste captures into chat. Report
counts and file paths; do not quote private message text, space names, member
names, or search snippets unless the user asks.

## When To Use

Use this skill when the user wants to read Google Chat (history, a permalink,
search, the space list, a space record, or space members) without a Google
Cloud project, or to initialize, check, or fixture a `gchatctl` session.

## Inputs

- A local HAR or copied cURL. Prefer a `list_topics` cURL for `init`.
- Optional authuser key (default `0`).
- A private `--output` directory (or `--stdout` with exactly one `--format`).
- For permalink export: a `https://chat.google.com/room/...` URL.
- For Chat search: a learned `search.messages` capture and `--query TEXT`.
- For the space directory: a learned `paginated_world` capture.
- For a space record: a learned `get_group` capture and a space ID.
- For space members: a learned `list_members` capture and a space ID.

## Workflow

1. Keep captures on disk. Never ask the user to paste HAR, cURL, cookies, or
   Authorization headers into chat. Enable Chrome **Allow to generate HAR with
   sensitive data** only when using HAR; prefer cURL for cookies.

   ```bash
   pbpaste | gchatctl init
   gchatctl init --curl-file ./copied-request.txt
   gchatctl init --har-file ./chat.har
   gchatctl doctor
   gchatctl doctor --live
   ```

   Require `credentials found`, allowed host, and `cookie present`.
   `list_topics ready` is required for `topics list` exact replay and is the
   usual `init` target. `--live` must be `ok` before blaming parsers; `http_401`
   means recapture `list_topics`. Never print secret values.

2. Route the user ask. Run at most the matching command. Write `--output`
   under a private local path.

   | User ask | Doctor / learn | Command | Stdout may contain |
   | --- | --- | --- | --- |
   | Dump this `chat.google.com/room` link | `list_topics ready` (cookies enough to synthesize a body) | `gchatctl conversation export '<permalink>' --output ./conversation-export` | topic/message counts, paths, complete |
   | History of the captured room (no other ID) | `list_topics ready` | `gchatctl topics list --output ./topics-export` | topic/message counts, paths |
   | History of space `SPACEID` | `list_topics ready` | `gchatctl topics list --space SPACEID --output ./topics-export` | counts, paths. `--space` fetches that space. |
   | Filter already-listed message text | same as topics list | `gchatctl topics list --contains NEEDLE --output ./topics-export` | counts, paths. This is not Chat search. |
   | Find text across Chat | `search ready`; else `learn search.messages` from a `batchexecute` `SBNmJb` cURL | `gchatctl search messages --query TEXT --output ./search-export` | hit count, query, paths |
   | What rooms do I have | `paginated_world ready`; else `learn paginated_world` from HAR | `gchatctl spaces list --output ./spaces-export` | space count, space IDs, paths |
   | What is this space | `get_group ready`; else `learn get_group` from HAR | `gchatctl spaces get SPACEID --output ./space-export` | space ID, paths. Name stays in files. |
   | Who is in this space | `list_members ready`; else `learn list_members` from HAR | `gchatctl members list --space SPACEID --output ./members-export` | member count, member IDs, space, paths. Names stay in files. |
   | Bind a new RPC | credentials exist | `gchatctl learn OPERATION --har-file PATH` or `--curl-file PATH` | operation name. HAR without cookies still stores the body template. |
   | Discuss request inventory | — | `gchatctl fixture --har-file ./chat.har --out ./redacted.json` | path, request count, hosts. Read the fixture only. |

3. Learn when doctor says the RPC is not ready. Cookie-stripped HAR is enough
   Cookie-stripped HAR is enough for `paginated_world`, `list_members`, and
   `get_group`. Search needs a cookie-bearing `batchexecute` cURL.

   ```bash
   gchatctl learn paginated_world --har-file ./chat.har
   gchatctl learn list_members --har-file ./chat.har
   gchatctl learn get_group --har-file ./chat.har
   gchatctl learn search.messages --curl-file ./copied-request.txt
   ```

4.    After search, if the user wants a thread opened, use `conversation export`
   with a `permalink` from the export files when that field is present. If it
   is empty, web IDs were not in the search hit; do not paste snippets to
   reconstruct the thread. History of that space is `topics list --space`.

5. Report only command, exit, counts, IDs when the command prints them, and
   paths. Do not quote message text, snippets, or space names unless asked.

Supported operational commands are `gchatctl init`, `gchatctl doctor`,
`gchatctl learn`, `gchatctl fixture`, `gchatctl topics list`,
`gchatctl spaces list`, `gchatctl spaces get`, `gchatctl members list`,
`gchatctl search messages`, and `gchatctl conversation export`.

## Validation

- `gchatctl doctor` after `init`; `--live` before live reads.
- Command exit 0, or recapture on HTTP 401/403.
- Stdout is counts and paths (and space IDs for `spaces list` / `spaces get`,
  member IDs for `members list`), not bodies.
- Fixture has no cookies, Authorization values, or message bodies.
- Never bypass the credential leak scan.

## Safety

- Never ask for or display copied cURL, HAR, tokens, cookies, or authorization
  data.
- Never execute copied cURL through a shell.
- Never read browser cookie databases.
- Keep captures and exports local; do not commit original HARs or topic
  exports.
- A redacted fixture may be shared. Original captures and topic exports may
  not.
- Do not quote private Google Chat message text, search snippets, space
  names, or member names in chat replies unless the user asks.

## References

- [`tools/gchatctl/README.md`](../../tools/gchatctl/README.md)
- CLI help: `gchatctl --help`
- Validation: `cd tools/gchatctl/src && go test ./...`
