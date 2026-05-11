#!/usr/bin/env bash
# install-skills.sh — copy skills from this repo into ~/.kiro/skills/
#
# Usage:
#   bash scripts/install-skills.sh
#
# Run from the ai-resources repo root.
#
# Kiro reads skills from ~/.kiro/skills/<skill-name>/SKILL.md
# Re-running is safe: existing skill directories are replaced in-place.
#
# NOTE: We copy instead of symlink because Kiro IDE does not follow symlinks.
# See https://github.com/kirodotdev/Kiro/issues/6401

set -euo pipefail

SKILLS_DIR="${HOME}/.kiro/skills"
AI_RESOURCES_SKILLS="$(cd "$(dirname "$0")/.." && pwd)/skills"

mkdir -p "${SKILLS_DIR}"

install_skill() {
  local skill_dir="$1"
  local skill_name
  skill_name="$(basename "${skill_dir}")"
  local target="${SKILLS_DIR}/${skill_name}"

  if [[ -L "${target}" ]]; then
    rm "${target}"
  elif [[ -d "${target}" ]]; then
    rm -rf "${target}"
  elif [[ -e "${target}" ]]; then
    echo "  SKIP  ${skill_name}  (${target} exists and is not a dir or symlink — remove it manually)"
    return
  fi

  cp -R "${skill_dir}" "${target}"
  echo "  COPY  ${skill_name}  ←  ${skill_dir}"
}

echo "Installing skills to ${SKILLS_DIR}"
echo ""

echo "[ai-resources]"
for d in "${AI_RESOURCES_SKILLS}"/*/; do
  [[ -f "${d}SKILL.md" ]] && install_skill "${d%/}"
done

echo ""
echo "Done. Installed $(find "${SKILLS_DIR}" -maxdepth 1 -type d | tail -n +2 | wc -l | tr -d ' ') skills."
