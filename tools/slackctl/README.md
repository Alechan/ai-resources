# Slack conversation exporter (`slackctl`)

`slackctl` is a read-only CLI for exporting Slack conversations that the
authenticated user can already access. It preserves raw API pages, creates a
small versioned JSON document, and renders a chronological Markdown transcript.
It never executes copied cURL commands or reads browser cookie databases.

## Install

```bash
go install github.com/Alechan/ai-resources/tools/slackctl/src/cmd/slackctl@latest
slackctl --help
```

For local development:

```bash
cd tools/slackctl/src
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/slackctl
```

The test suite includes the repository safety scan. For an additional local
denylist, set `SLACKCTL_SAFETY_DENYLIST` to a newline-delimited file of values
that must never appear in the tool or skill tree; values are never printed.

Version 1 uses macOS Keychain. The Keychain service is `slackctl`; the account
is the workspace host, allowing independent records for multiple workspaces.

## Initialize credentials

In browser developer tools, open the Network panel, choose a successful
read-only Slack API request, and use **Copy as cURL**. Keep the copied request
local: never paste it into chat, a ticket, source control, or a shell command.
Pipe it to the parser as data:

```bash
pbpaste | slackctl init
slackctl init --curl-file ./copied-request.txt
```

The parser accepts cookies from `-b` or a `Cookie` header and tokens from
URL-encoded, raw, or multipart form data. It validates the Slack host and
credentials, saves one JSON credential record in Keychain, then discards its
input. It does not invoke `curl` or any shell.

Select or clear a workspace explicitly:

```bash
slackctl init --clear --workspace alpha.slack.com
SLACKCTL_WORKSPACE=alpha.slack.com slackctl doctor
```

## Check access

```bash
slackctl doctor --workspace alpha.slack.com
slackctl doctor --workspace alpha.slack.com --json
```

The check confirms that credentials exist, authentication succeeds, the user is
identified, and a read-only conversation endpoint is available. Refresh the
copied request when authentication expires. Permission failures mean that the
authenticated user cannot access the requested conversation; the tool does not
attempt a workaround.

## Export

Export all accessible history:

```bash
slackctl conversation export \
  https://app.slack.com/client/T11111111/C22222222 \
  --workspace alpha.slack.com \
  --all \
  --include-threads \
  --output ./conversation-export
```

Export an inclusive bounded range:

```bash
slackctl conversation export C22222222 \
  --workspace alpha.slack.com \
  --from 2026-01-01T00:00:00Z \
  --to 2026-01-07T23:59:59Z \
  --format raw,json,markdown \
  --output ./conversation-export
```

Times accept RFC3339, `now`, or relative values such as `now-2d`. Use
`--resume` after an interruption. Use `--allow-partial` only when an incomplete
result is explicitly acceptable. `--page-size` defaults to 100 and
`--request-delay` defaults to 500ms. The client honors `Retry-After` and applies
bounded retries to network and server failures.

Emit exactly one format without filesystem output:

```bash
slackctl conversation export C22222222 \
  --workspace alpha.slack.com \
  --from now-1d \
  --format markdown \
  --stdout
```

## Output

An output directory contains:

```text
manifest.json
conversation.md
messages_with_threads.json
raw_history/page_0001.json
raw_threads/thread_100_000001_page_0001.json
```

`manifest.json` schema version 1 is flat. Read these top-level fields directly:
`schema_version`, `workspace_host`, `workspace_id`, `conversation_id`,
`conversation_type`, `exported_at`, `requested_from`, `requested_to`,
`oldest_exported`, `newest_exported`, `root_message_count`,
`thread_reply_count`, `participant_count`, and `complete`. It does not contain
nested `requested`, `exported`, `counts`, `warnings`, or `files` objects.

The normalized JSON keeps only conversation identity, resolved participants,
chronological messages, and nested chronological replies. Unknown Slack fields
remain in immutable raw pages. Markdown preserves line breaks and renders Slack
links and mentions without summarizing or interpreting content.

### Markdown ambiguity

`conversation.md` adds Markdown syntax for presentation. In particular, thread
replies are rendered as blockquotes using `>`. Original message text may contain
the same characters, so the visual transcript is not an unambiguous data format.
For example, this original root-message text:

```text
Deployment finished.

> This blockquote is part of the original message.
```

followed by a thread reply appears as:

```markdown
**2026-01-01 10:00 UTC — Example User**

Deployment finished.

> This blockquote is part of the original message.

> **2026-01-01 10:05 UTC — Another User**
>
> This is an actual thread reply.
```

Use `messages_with_threads.json` when the distinction must be unambiguous:
original content is stored in `text`, while actual replies are separate objects
under `thread_replies`. Use `raw_history/` and `raw_threads/` when the original
Slack API responses are required.

On completion, `slackctl` checks ordering, bounds, thread coverage, counts, and
the output tree for exact credential values. Unsafe affected files are removed
and the command fails. Raw pages support deterministic resumption.

## Privacy and troubleshooting

Exports may contain confidential or personal data. Keep them in a private local
directory, outside repositories, and do not upload or commit them without
explicit authorization. The tool does not alter message bodies.

- **Credentials missing:** run `slackctl init`.
- **Authentication failed:** copy a fresh successful read-only request.
- **Permission denied:** verify membership and the workspace/conversation ID.
- **Incomplete export:** inspect `manifest.json`, correct the cause, then use
  `--resume`; do not treat `--allow-partial` output as complete.
- **Rate limited:** leave the command running; it honors Slack's retry interval.

Automated tests use synthetic fixtures and local HTTP servers only. They never
contact Slack.
