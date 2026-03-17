#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

if ! command -v shellcheck >/dev/null 2>&1; then
  echo "shellcheck not installed; skipping shell lint."
  exit 0
fi

shell_files=()
while IFS= read -r -d '' file; do
  shell_files+=("$file")
done < <(find "$repo_root" -type f -name '*.sh' -print0 | sort -z)

if [ "${#shell_files[@]}" -eq 0 ]; then
  echo "No shell scripts found."
  exit 0
fi

shellcheck "${shell_files[@]}"

echo "Shell lint passed."
