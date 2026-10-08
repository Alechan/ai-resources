# Google Chat capture tool (`gchatctl`)

`gchatctl` is a read-only CLI that stores a Google Chat browser session from a
local HAR or cURL capture. It can replay a stored `list_topics` request, list
spaces from a learned `paginated_world` body, get a space from `get_group`,
list members from `list_members`, search messages from a learned
`batchexecute` body, and write local exports. It never executes copied cURL
commands or reads browser cookie databases.

## Install

```bash
go install github.com/Alechan/ai-resources/tools/gchatctl/src/cmd/gchatctl@latest
gchatctl --help
```

For local development:

```bash
cd tools/gchatctl/src
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/gchatctl
```

The test suite includes the repository safety scan. For an additional local
denylist, set `GCHATCTL_SAFETY_DENYLIST` to a newline-delimited file of values
that must never appear in the tool or skill tree; values are never printed.

Version 1 uses macOS Keychain. The Keychain service is `gchatctl`; the account
is the Google Chat authuser key (`0` by default).

## Initialize credentials

In browser developer tools, open the Network panel. In Network settings, enable
**Allow to generate HAR with sensitive data** (otherwise Chrome strips cookies
and `gchatctl init` cannot store a session). Open a space so a successful
`list_topics` POST appears, then copy that request as cURL.

```bash
pbpaste | gchatctl init
gchatctl init --har-file ./chat.har
gchatctl init --curl-file ./copied-request.txt
```

`topics list` replays the stored POST, so the capture must be `list_topics`
and must include the request body. Chrome HAR exports often omit cookies;
prefer a copied cURL. Re-run `init` after upgrading so the body is stored.

Select or clear an account explicitly:

```bash
gchatctl init --clear --account 0
GCHATCTL_ACCOUNT=0 gchatctl doctor
```

## Check access

```bash
gchatctl doctor
gchatctl doctor --live
gchatctl doctor --account 0 --json
```

`doctor` confirms that credentials exist, the stored host is a Google Chat
host, a cookie is present, and whether the stored capture can replay
`list_topics`. `paginated_world ready` is true after `gchatctl learn
paginated_world` stored a world request body. `search ready` is true after
`gchatctl learn search.messages` stored a `batchexecute` form body.
`list_members ready` and `get_group ready` are true after those HAR bodies
are learned. `body shape` is a redacted slot
map of the stored POST body (types and string lengths only). `--live` sends
the stored `list_topics` request and reports `ok`, `http_401`, `http_403`, or
`http_other`; it does not print cookies or XSRF values. After HTTP 401/403,
gchatctl tries once to refresh `x-framework-xsrf-token` from Chat HTML using
the stored cookies. Refresh the capture when `--live` still reports
`http_401` or after the browser session expires.

## Export from a permalink

Start at a copied Chat message or thread link. The ID is an inclusive lower
bound: that thread is kept in full, plus later topics in the same space.

```bash
gchatctl conversation export \
  'https://chat.google.com/room/SPACEID/THREADID/MESSAGEID' \
  --output ./conversation-export
```

This rewrites the stored `list_topics` body for the permalink space and pages
older until the starting message is included. Command text output is counts
and file paths. `--allow-partial` keeps whatever was fetched if the start
message is not found before `--max-pages`.

## List topics

Replay the stored `list_topics` POST, or fetch a space with `--space`.
`--contains` filters parsed message text on the client; it is not Google Chat
search.

```bash
gchatctl topics list --output ./topics-export
gchatctl topics list --contains nps --format json --stdout
gchatctl topics list --space SPACEID --output ./topics-export
```

`--output` writes `topics.json` and `topics.md` unless `--format` selects one.
`--stdout` requires exactly one format. Command text output is counts and file
paths; message text stays in the export files.

`--space` rewrites the stored `list_topics` body for that space and pages
until Chat reports complete (or `--max-pages`). Without `--space`, the stored
capture is replayed as-is.

## Which command

| Ask | Command |
| --- | --- |
| Permalink dump | `conversation export <CHAT-URL>` |
| Captured room history | `topics list` |
| History of space ID | `topics list --space ID` |
| Filter listed text | `topics list --contains` (not Chat search) |
| Search Chat | `search messages --query` after `learn search.messages` |
| List rooms | `spaces list` after `learn paginated_world` |
| What is this space | `spaces get ID` after `learn get_group` |
| Who is in this space | `members list --space ID` after `learn list_members` |

## List spaces

Learn `paginated_world` from a HAR or cURL, then list spaces. Cookie-stripped
HAR files can still supply the request body; cookies stay on the `list_topics`
init and are not replaced.

```bash
gchatctl learn paginated_world --har-file ./chat.har
gchatctl spaces list --output ./spaces-export
gchatctl spaces list --format json --stdout
```

`--output` writes `spaces.json` and `spaces.md` unless `--format` selects one.
Command text output is the space count, IDs, and file paths. Space names stay
in the export files.

## Get a space

Learn `get_group` from a HAR, then fetch one space record. Cookie-stripped HAR
files can still supply the request body.

```bash
gchatctl learn get_group --har-file ./chat.har
gchatctl spaces get SPACEID --output ./space-export
gchatctl spaces get --space SPACEID --format json --stdout
```

Command text output is the space ID and file paths. The display name stays in
the export files.

## List members

Learn `list_members` from a HAR, then list members for a space ID.

```bash
gchatctl learn list_members --har-file ./chat.har
gchatctl members list --space SPACEID --output ./members-export
gchatctl members list --space SPACEID --format json --stdout
```

Command text output is the member count, member IDs, space ID, and file paths.
Display names stay in the export files.

## Search messages

Learn `search.messages` from a successful Chat search `batchexecute` cURL
(cookies required), then replay it with a new `--query`. This is Chat search.
`topics list --contains` remains a local filter of already-exported text.

```bash
gchatctl learn search.messages --curl-file ./copied-request.txt
gchatctl search messages --query nps --output ./search-export
gchatctl search messages --query nps --format json --stdout
```

`--output` writes `search.json` and `search.md` unless `--format` selects one.
Command text output is the hit count, query, and file paths. Snippets stay in
the export files. When a hit includes thread and message web IDs, JSON and
Markdown also include a `https://chat.google.com/room/...` permalink for
`conversation export`.

## Learn an operation

After capturing a later UI action (list spaces, open a space, search), bind
that request shape to a name:

```bash
gchatctl learn paginated_world --har-file ./chat.har
gchatctl learn list_members --har-file ./chat.har
gchatctl learn get_group --har-file ./chat.har
gchatctl learn search.messages --curl-file ./copied-request.txt
```

## Redacted fixture

Write a shareable request inventory with cookies, tokens, query values, and
bodies removed:

```bash
gchatctl fixture --har-file ./chat.har --out ./redacted.json
```

The fixture keeps hosts, methods, redacted URL paths, query names, and header
names. That file may be shared in chat. The original HAR may not.

## Privacy and troubleshooting

Exports and captures may contain confidential or personal data. Keep them in a
private local directory, outside repositories, and do not upload or commit them
without explicit authorization.

- **Credentials missing:** run `gchatctl init`.
- **Host rejected:** capture a request to `chat.google.com`,
  `clients6.google.com`, or `chat.googleapis.com`.
- **Authentication expired:** copy a fresh successful `list_topics` request.
- **list_topics not ready:** copy a `list_topics` cURL (including the body) and
  rerun `gchatctl init`.
- **paginated_world not ready:** run `gchatctl learn paginated_world` from a
  HAR or `paginated_world` cURL. This does not replace the stored
  `list_topics` body.
- **list_members not ready:** run `gchatctl learn list_members` from a HAR.
- **get_group not ready:** run `gchatctl learn get_group` from a HAR.
- **search not ready:** copy a successful Chat search `batchexecute` cURL and
  run `gchatctl learn search.messages`. This does not replace the stored
  `list_topics` body.
- **Need a new RPC:** capture that UI action and run `gchatctl learn`.
- **Search:** use `gchatctl search messages --query`. `topics list --contains`
  is still a local text filter.

Automated tests use synthetic fixtures only. They never contact Google Chat.
