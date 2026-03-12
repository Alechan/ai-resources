#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "Usage: $0 <skill-name> <source-skill-md-or-skill-dir>" >&2
  exit 1
fi

skill_name="$1"
source_input="$2"

if [ -d "$source_input" ]; then
  source_file="${source_input%/}/SKILL.md"
else
  source_file="$source_input"
fi

if [ ! -f "$source_file" ]; then
  echo "Error: source skill file not found: $source_file" >&2
  exit 1
fi

source_dir="$(cd "$(dirname "$source_file")" && pwd -P)"
source_abs="$source_dir/$(basename "$source_file")"

dest_dir="$HOME/.codex/skills/$skill_name"
dest_file="$dest_dir/SKILL.md"

mkdir -p "$dest_dir"
ln -sf "$source_abs" "$dest_file"

echo "Installed Codex skill: $dest_file -> $source_abs"
