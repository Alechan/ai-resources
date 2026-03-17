#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
catalog="$repo_root/docs/RESOURCE_CATALOG.md"
errors=0

check_file() {
  local rel="$1"
  if [ ! -f "$repo_root/$rel" ]; then
    echo "[missing file] $rel"
    errors=1
  fi
}

check_dir() {
  local rel="$1"
  if [ ! -d "$repo_root/$rel" ]; then
    echo "[missing dir] $rel"
    errors=1
  fi
}

check_executable() {
  local rel="$1"
  if [ ! -x "$repo_root/$rel" ]; then
    echo "[not executable] $rel"
    errors=1
  fi
}

require_catalog_entry() {
  local name="$1"
  local type="$2"
  local needle="| $name | $type |"
  if ! grep -Fq "$needle" "$catalog"; then
    echo "[catalog missing] $type $name"
    errors=1
  fi
}

required_dirs=(
  "docs"
  "skills"
  "skills/gdrivectl-drive-ops"
  "skills/datagrip-datasources"
  "skills/datagrip-datasources/evals"
  "agents"
  "tools"
  "tools/claude-statusline"
  "tools/claude-statusline/src"
  "tools/gdrivectl"
  "tools/gdrivectl/src"
  "tools/gdrivectl/src/cmd"
  "tools/gdrivectl/src/cmd/gdrivectl"
  "tools/gdrivectl/src/internal"
  "playbooks"
  "scripts"
)

required_files=(
  "README.md"
  "AGENTS.md"
  "CHANGELOG.md"
  "docs/CONVENTIONS.md"
  "docs/RESOURCE_CATALOG.md"
  "skills/gdrivectl-drive-ops/SKILL.md"
  "skills/datagrip-datasources/SKILL.md"
  "skills/datagrip-datasources/evals/v1-prompts.md"
  "agents/gdrivectl-drive-ops.md"
  "agents/datagrip-datasources.md"
  "tools/claude-statusline/README.md"
  "tools/claude-statusline/src/statusline-command.sh"
  "tools/gdrivectl/README.md"
  "tools/gdrivectl/src/go.mod"
  "tools/gdrivectl/src/cmd/gdrivectl/main.go"
  "playbooks/datagrip-datasource-update.md"
  "scripts/install_codex_skill.sh"
  "scripts/install_claude_agent.sh"
  "scripts/install_claude_statusline.sh"
  "scripts/lint_shell.sh"
  "scripts/verify_repo.sh"
  "scripts/run_datagrip_evals.py"
  "scripts/score_datagrip_evals.py"
  "scripts/mock_datagrip_eval_runner.py"
)

for dir in "${required_dirs[@]}"; do
  check_dir "$dir"
done

for file in "${required_files[@]}"; do
  check_file "$file"
done

for skill_dir in "$repo_root"/skills/*; do
  [ -d "$skill_dir" ] || continue
  skill_name="$(basename "$skill_dir")"
  if [ ! -f "$skill_dir/SKILL.md" ]; then
    echo "[missing file] skills/$skill_name/SKILL.md"
    errors=1
  fi
  require_catalog_entry "$skill_name" "skill"
done

for agent_file in "$repo_root"/agents/*.md; do
  [ -f "$agent_file" ] || continue
  agent_name="$(basename "$agent_file" .md)"
  require_catalog_entry "$agent_name" "agent"
done

check_executable "scripts/install_codex_skill.sh"
check_executable "scripts/install_claude_agent.sh"
check_executable "scripts/install_claude_statusline.sh"
check_executable "scripts/lint_shell.sh"
check_executable "scripts/verify_repo.sh"
check_executable "scripts/run_datagrip_evals.py"
check_executable "scripts/score_datagrip_evals.py"
check_executable "scripts/mock_datagrip_eval_runner.py"
check_executable "tools/claude-statusline/src/statusline-command.sh"

require_catalog_entry "claude-statusline" "tool"

if ! "$repo_root/scripts/lint_shell.sh"; then
  errors=1
fi

if [ "$errors" -ne 0 ]; then
  echo "Repository verification failed."
  exit 1
fi

echo "Repository verification passed."
