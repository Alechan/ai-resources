#!/usr/bin/env bash
# install_cursor_skill.sh — install skills into Cursor user skill directories
#
# Usage:
#   bash scripts/install_cursor_skill.sh
#   bash scripts/install_cursor_skill.sh <skill-name> <source-skill-dir>
#
# Run from the ai-resources repo root.
#
# Cursor reads skills from ~/.cursor/skills/<skill-name>/SKILL.md
# Re-running is safe: existing skill directories or symlinks are replaced.
# Skills are symlinked so edits in this repository take effect immediately.

set -euo pipefail

CURSOR_SKILLS_DIR="${HOME}/.cursor/skills"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
AI_RESOURCES_SKILLS="${REPO_ROOT}/skills"

install_skill_dir() {
  local source_dir="$1"
  local skill_name
  skill_name="$(basename "${source_dir}")"
  local target="${CURSOR_SKILLS_DIR}/${skill_name}"

  if [[ ! -f "${source_dir}/SKILL.md" ]]; then
    echo "Error: SKILL.md not found in ${source_dir}" >&2
    exit 1
  fi

  source_dir="$(cd "${source_dir}" && pwd -P)"

  if [[ -L "${target}" ]]; then
    rm "${target}"
  elif [[ -d "${target}" ]]; then
    rm -rf "${target}"
  elif [[ -e "${target}" ]]; then
    echo "  SKIP  ${skill_name}  (${target} exists and is not a dir or symlink — remove it manually)"
    return
  fi

  ln -sfn "${source_dir}" "${target}"
  echo "  LINK  ${skill_name}  →  ${target}"
}

install_all_skills() {
  echo "Installing skills to ${CURSOR_SKILLS_DIR}"
  echo ""
  mkdir -p "${CURSOR_SKILLS_DIR}"
  for d in "${AI_RESOURCES_SKILLS}"/*/; do
    [[ -f "${d}SKILL.md" ]] && install_skill_dir "${d%/}"
  done
  echo ""
  echo "Done."
  echo "Cursor skills: $(find "${CURSOR_SKILLS_DIR}" -maxdepth 1 \( -type d -o -type l \) | tail -n +2 | wc -l | tr -d ' ')"
}

if [[ "$#" -eq 0 ]]; then
  install_all_skills
  exit 0
fi

if [[ "$#" -ne 2 ]]; then
  echo "Usage: $0 [<skill-name> <source-skill-dir>]" >&2
  exit 1
fi

skill_name="$1"
source_input="$2"

if [[ -d "${source_input}" ]]; then
  source_dir="${source_input%/}"
else
  source_dir="$(dirname "${source_input}")"
fi

if [[ "$(basename "${source_dir}")" != "${skill_name}" ]]; then
  echo "Error: skill directory name must match skill name (${skill_name})" >&2
  exit 1
fi

mkdir -p "${CURSOR_SKILLS_DIR}"
install_skill_dir "${source_dir}"
echo "Installed Cursor skill: ${CURSOR_SKILLS_DIR}/${skill_name}"
