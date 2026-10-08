#!/usr/bin/env bash
# install-skills.sh — install skills into Kiro and Copilot user skill directories
#
# Usage:
#   bash scripts/install-skills.sh
#   ADDITIONAL_SKILL_SOURCES="$HOME/other-repo/skills" bash scripts/install-skills.sh
#
# Run from the ai-resources repo root.
# Set ADDITIONAL_SKILL_SOURCES to newline-separated skill-root directories to
# install skills from other sources after the repository's own skills.
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

install_skills_from_root() {
  local skills_root="$1"
  local d
  if [[ ! -d "${skills_root}" ]]; then
    echo "  SKIP  skills source not found: ${skills_root}" >&2
    return
  fi

  for d in "${skills_root}"/*/; do
    [[ -f "${d}SKILL.md" ]] && install_skill "${d%/}"
  done
}

echo "Installing skills to:"
echo "  - ${KIRO_SKILLS_DIR}"
echo "  - ${COPILOT_SKILLS_DIR}"
echo ""

echo "[ai-resources]"
install_skills_from_root "${AI_RESOURCES_SKILLS}"

if [[ -n "${ADDITIONAL_SKILL_SOURCES:-}" ]]; then
  while IFS= read -r skills_root; do
    [[ -n "${skills_root}" ]] || continue
    echo ""
    echo "[additional skills]"
    install_skills_from_root "${skills_root}"
  done <<< "${ADDITIONAL_SKILL_SOURCES}"
fi

echo ""
echo "Done."
echo "Kiro skills:    $(find "${KIRO_SKILLS_DIR}" -maxdepth 1 -type d | tail -n +2 | wc -l | tr -d ' ')"
echo "Copilot skills: $(find "${COPILOT_SKILLS_DIR}" -maxdepth 1 -type d | tail -n +2 | wc -l | tr -d ' ')"
