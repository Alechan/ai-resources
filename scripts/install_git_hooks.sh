#!/usr/bin/env bash
# install_git_hooks.sh — install repository git hooks via symlinks
#
# Usage:
#   bash scripts/install_git_hooks.sh
#
# Run from the ai-resources repo root.

set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
hooks_src="${repo_root}/scripts/git-hooks"
hooks_dest="${repo_root}/.git/hooks"

if [[ ! -d "${hooks_src}" ]]; then
  echo "Error: hooks source directory not found: ${hooks_src}" >&2
  exit 1
fi

mkdir -p "${hooks_dest}"

for hook in "${hooks_src}"/*; do
  [[ -f "${hook}" ]] || continue
  hook_name="$(basename "${hook}")"
  target="${hooks_dest}/${hook_name}"

  if [[ -e "${target}" && ! -L "${target}" ]]; then
    echo "  SKIP  ${hook_name}  (${target} exists and is not a symlink)"
    continue
  fi

  ln -sfn "${hook}" "${target}"
  chmod +x "${hook}"
  echo "  LINK  ${hook_name}  →  ${target}"
done

echo "Git hooks installed."
