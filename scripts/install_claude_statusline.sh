#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
source_file="$repo_root/tools/claude-statusline/src/statusline-command.sh"
dest_dir="$HOME/.claude"
dest_file="$dest_dir/statusline-command.sh"
settings_file="$dest_dir/settings.json"

if [ ! -f "$source_file" ]; then
  echo "Error: source status line script not found: $source_file" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "Error: jq is required to install claude-statusline." >&2
  exit 1
fi

mkdir -p "$dest_dir"

if [ -f "$settings_file" ]; then
  if ! jq -e 'type == "object"' "$settings_file" >/dev/null 2>&1; then
    echo "Error: Claude settings file must contain a JSON object: $settings_file" >&2
    exit 1
  fi

  if jq -e 'has("statusLine")' "$settings_file" >/dev/null 2>&1; then
    echo "Error: Claude settings already define statusLine: $settings_file" >&2
    exit 1
  fi
fi

ln -sf "$source_file" "$dest_file"

tmp_file="$(mktemp)"
if [ -f "$settings_file" ]; then
  jq \
    --arg command "bash $dest_file" \
    '. + {
      statusLine: {
        type: "command",
        command: $command,
        padding: 0
      }
    }' \
    "$settings_file" > "$tmp_file"
else
  jq -n \
    --arg command "bash $dest_file" \
    '{
      statusLine: {
        type: "command",
        command: $command,
        padding: 0
      }
    }' > "$tmp_file"
fi

mv "$tmp_file" "$settings_file"

echo "Installed Claude status line: $dest_file -> $source_file"
echo "Updated Claude settings: $settings_file"
