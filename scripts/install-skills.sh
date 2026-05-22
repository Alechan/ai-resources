#!/usr/bin/env bash
# install-skills.sh — install skills into Kiro and Copilot user skill directories
#
# Usage:
#   bash scripts/install-skills.sh
#
# Run from the ai-resources repo root.
#
# Kiro reads skills from ~/.kiro/skills/<skill-name>/SKILL.md
# Copilot reads skills from ~/.copilot/skills/<skill-name>/SKILL.md
# Re-running is safe: existing skill directories are replaced in-place.
#
# NOTE: We copy instead of symlink because Kiro IDE does not follow symlinks.
# See https://github.com/kirodotdev/Kiro/issues/6401

set -euo pipefail

KIRO_SKILLS_DIR="${HOME}/.kiro/skills"
COPILOT_SKILLS_DIR="${HOME}/.copilot/skills"
AI_RESOURCES_SKILLS="$(cd "$(dirname "$0")/.." && pwd)/skills"

mkdir -p "${KIRO_SKILLS_DIR}" "${COPILOT_SKILLS_DIR}"

install_skill_to_target() {
  local skill_dir="$1"
  local target_root="$2"
  local skill_name
  skill_name="$(basename "${skill_dir}")"
  local target="${target_root}/${skill_name}"

  if [[ -L "${target}" ]]; then
    rm "${target}"
  elif [[ -d "${target}" ]]; then
    rm -rf "${target}"
  elif [[ -e "${target}" ]]; then
    echo "  SKIP  ${skill_name}  (${target} exists and is not a dir or symlink — remove it manually)"
    return
  fi

  cp -R "${skill_dir}" "${target}"
  echo "  COPY  ${skill_name}  →  ${target_root}"
}

install_skill() {
  local skill_dir="$1"
  install_skill_to_target "${skill_dir}" "${KIRO_SKILLS_DIR}"
  install_skill_to_target "${skill_dir}" "${COPILOT_SKILLS_DIR}"
}

echo "Installing skills to:"
echo "  - ${KIRO_SKILLS_DIR}"
echo "  - ${COPILOT_SKILLS_DIR}"
echo ""

echo "[ai-resources]"
for d in "${AI_RESOURCES_SKILLS}"/*/; do
  [[ -f "${d}SKILL.md" ]] && install_skill "${d%/}"
done

# Also install from mytheresa_ecosystem if available
MYT_ECOSYSTEM_SKILLS="${HOME}/src/mytheresa_ecosystem/skills"
if [[ -d "${MYT_ECOSYSTEM_SKILLS}" ]]; then
  echo ""
  echo "[mytheresa_ecosystem]"
  for d in "${MYT_ECOSYSTEM_SKILLS}"/*/; do
    [[ -f "${d}SKILL.md" ]] && install_skill "${d%/}"
  done
fi

echo ""
echo "Done."
echo "Kiro skills:    $(find "${KIRO_SKILLS_DIR}" -maxdepth 1 -type d | tail -n +2 | wc -l | tr -d ' ')"
echo "Copilot skills: $(find "${COPILOT_SKILLS_DIR}" -maxdepth 1 -type d | tail -n +2 | wc -l | tr -d ' ')"
