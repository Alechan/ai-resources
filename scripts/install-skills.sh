#!/usr/bin/env bash
# install-skills.sh — symlink all skills from both repos into ~/.copilot/skills/
#
# Usage:
#   bash scripts/install-skills.sh
#
# Run from the ai-resources repo root. The script also picks up skills from
# the sibling mytheresa_ecosystem repo if it exists at ~/src/mytheresa_ecosystem.
#
# Copilot reads skills from ~/.copilot/skills/<skill-name>/SKILL.md
# Re-running is safe: existing symlinks are replaced in-place.

set -euo pipefail

SKILLS_DIR="${HOME}/.copilot/skills"
AI_RESOURCES_SKILLS="$(cd "$(dirname "$0")/.." && pwd)/skills"
MYT_ECOSYSTEM="${HOME}/src/mytheresa_ecosystem/skills"

mkdir -p "${SKILLS_DIR}"

install_skill() {
  local skill_dir="$1"
  local skill_name
  skill_name="$(basename "${skill_dir}")"
  local target="${SKILLS_DIR}/${skill_name}"

  if [[ -L "${target}" ]]; then
    rm "${target}"
  elif [[ -e "${target}" ]]; then
    echo "  SKIP  ${skill_name}  (${target} exists and is not a symlink — remove it manually)"
    return
  fi

  ln -s "${skill_dir}" "${target}"
  echo "  LINK  ${skill_name}  →  ${skill_dir}"
}

echo "Installing skills to ${SKILLS_DIR}"
echo ""

echo "[ai-resources]"
for d in "${AI_RESOURCES_SKILLS}"/*/; do
  [[ -f "${d}SKILL.md" ]] && install_skill "${d%/}"
done

echo ""
if [[ -d "${MYT_ECOSYSTEM}" ]]; then
  echo "[mytheresa_ecosystem]"
  for d in "${MYT_ECOSYSTEM}"/*/; do
    [[ -f "${d}SKILL.md" ]] && install_skill "${d%/}"
  done
else
  echo "[mytheresa_ecosystem] not found at ${MYT_ECOSYSTEM} — skipping"
fi

echo ""
echo "Done. Run '/skills reload' in Copilot CLI to pick up changes."
