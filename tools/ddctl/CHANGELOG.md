# ddctl CHANGELOG

## Unreleased

### Breaking

- `ddctl logs query --json` no longer returns `data[].attributes`. Events are flat objects with a full `custom` map copied from Datadog `event.custom`.

### Added

- `ddctl logs export` for NDJSON and CSV incident exports.
- `ddctl logs get <event_id>` for single-event lookup via `@evt.id`.
- `--fields` JSON projection on `logs query` and `logs get`.
- `--verbose` / `-v` and `--verbose-keys` for richer text output.

### Fixed

- Log `message` extraction prefers `msg` over `body` and includes `error`.
