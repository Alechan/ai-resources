#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "Usage: $0 <source-agent-file> <destination-agent-name>" >&2
  exit 1
fi

source_file="$1"
destination_name="$2"

if [ ! -f "$source_file" ]; then
  echo "Error: source agent file not found: $source_file" >&2
  exit 1
fi

destination_name="${destination_name%.md}"
source_dir="$(cd "$(dirname "$source_file")" && pwd -P)"
source_abs="$source_dir/$(basename "$source_file")"

dest_dir="$HOME/.claude/agents"
dest_file="$dest_dir/$destination_name.md"

mkdir -p "$dest_dir"
ln -sf "$source_abs" "$dest_file"

echo "Installed Claude agent: $dest_file -> $source_abs"
