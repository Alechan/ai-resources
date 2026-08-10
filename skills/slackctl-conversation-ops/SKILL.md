---
name: slackctl-conversation-ops
description: Safely initialize slackctl, export accessible Slack conversations with threads, and validate local raw, JSON, and Markdown results.
---

# slackctl conversation operations

## Purpose

Use the read-only `slackctl` CLI to preserve an accessible Slack conversation
as raw API pages, normalized JSON, and chronological Markdown.

## When To Use

Use this skill when a user asks to export Slack messages, include complete
threads, save a private conversation locally, convert history to JSON or
Markdown, resume an interrupted export, or troubleshoot `slackctl`
authentication.

## Inputs

- A Slack conversation URL or conversation ID.
- The workspace host.
- Either all accessible history or explicit inclusive time bounds.
- A private local output directory and selected formats.
- Whether an incomplete result may be accepted.

## Workflow

1. Confirm the conversation reference, range, output location, and formats.
2. If initialization is needed, instruct the user to copy a successful
   read-only browser request and run `pbpaste | slackctl init`. Never request
   that the cURL or credentials be pasted into chat.
3. Validate access:

   ```bash
   slackctl doctor --workspace alpha.slack.com
   ```

4. Export all accessible history:

   ```bash
   slackctl conversation export C22222222 \
     --workspace alpha.slack.com \
     --all \
     --include-threads \
     --format raw,json,markdown \
     --output ./conversation-export
   ```

   For a bounded export, replace `--all` with `--from` and optionally `--to`.
   Resume an interrupted filesystem export with `--resume`.

5. Read `manifest.json`. Verify `complete` is true, requested and exported
   bounds are consistent, and root, reply, and participant counts are present.
6. Report only output paths, counts, completeness, and warnings. Do not quote
   private message text.

Supported operational commands are `slackctl init`, `slackctl doctor`, and
`slackctl conversation export`.

## Validation

- Run `slackctl doctor` before an export.
- Require a successful command exit unless the user explicitly approved
  `--allow-partial`.
- Confirm `manifest.json` has schema version 1 and `complete: true`.
- Confirm requested formats exist and raw page directories exist when `raw` was
  selected.
- Treat the built-in credential leak scan as mandatory; never bypass a failure.

## Safety

- Never ask for or display copied cURL, tokens, cookies, or authorization data.
- Never execute copied cURL through a shell.
- Export only conversations available to the authenticated user.
- Keep exports local and outside source repositories by default.
- Never upload, commit, summarize, or analyze private content unless separately
  and explicitly requested.
- Keep raw responses for reproducibility unless the user selects other formats.
- Use `--allow-partial` only with explicit acceptance of incomplete data.

## References

- [`tools/slackctl/README.md`](../../tools/slackctl/README.md)
- CLI help: `slackctl --help`
- Validation: `cd tools/slackctl/src && go test ./...`
