# Resource Catalog

| name | type | status | owner | install target | validation command | source/canonical repo |
| --- | --- | --- | --- | --- | --- | --- |
| gdrivectl-drive-ops | skill | active | platform | `~/.codex/skills/gdrivectl-drive-ops/SKILL.md` | `bash scripts/verify_repo.sh` | this repo |
| datagrip-datasources | skill | draft | platform | `~/.codex/skills/datagrip-datasources/SKILL.md` | `bash scripts/verify_repo.sh` | this repo |
| gdrivectl-drive-ops | agent | active | platform | `~/.claude/agents/gdrivectl-drive-ops.md` | `bash scripts/verify_repo.sh` | this repo |
| datagrip-datasources | agent | draft | platform | `~/.claude/agents/datagrip-datasources.md` | `bash scripts/verify_repo.sh` | this repo |
| claude-statusline | tool | active | platform | `~/.claude/statusline-command.sh` + `~/.claude/settings.json` | `bash scripts/verify_repo.sh` | this repo |
| gdrivectl | tool | active | platform | `go install github.com/Alechan/ai-resources/tools/gdrivectl/src/cmd/gdrivectl@latest` | `cd tools/gdrivectl/src && go test ./...` | this repo |
| datagrip-datasource-update | playbook | draft | data-platform | `playbooks/datagrip-datasource-update.md` | `bash scripts/verify_repo.sh` | this repo |
| ddctl-datadog-ops | skill | active | platform | `~/.codex/skills/ddctl-datadog-ops/SKILL.md` | `bash scripts/verify_repo.sh` | this repo |
| ddctl | tool | active | platform | `go install github.com/Alechan/ai-resources/tools/ddctl/src/cmd/ddctl@latest` | `cd tools/ddctl/src && go test ./...` | this repo |
| miro | tool | active | platform | `tools/miro/README.md` (no install — procedure doc) | `bash scripts/verify_repo.sh` | this repo |
| jenkinsctl | skill | active | platform | `~/.codex/skills/jenkinsctl/SKILL.md` | `bash scripts/verify_repo.sh` | this repo |
| jenkinsctl | tool | active | platform | `go install github.com/Alechan/ai-resources/tools/jenkinsctl/src/cmd/jenkinsctl@latest` | `cd tools/jenkinsctl/src && go test ./...` | this repo |
| go-tdd-workflow | skill | active | platform | `~/.codex/skills/go-tdd-workflow/SKILL.md` | `bash scripts/verify_repo.sh` | this repo |
| repo-worktree-bootstrap | skill | active | platform | `~/.codex/skills/repo-worktree-bootstrap/SKILL.md` | `bash scripts/verify_repo.sh` | this repo |
| lint-before-push | skill | active | platform | `~/.codex/skills/lint-before-push/SKILL.md` | `bash scripts/verify_repo.sh` | this repo |
| pr-qlty-triage | skill | active | platform | `~/.codex/skills/pr-qlty-triage/SKILL.md` | `bash scripts/verify_repo.sh` | this repo |
| vis-network-diagrams | skill | active | platform | `~/.codex/skills/vis-network-diagrams/SKILL.md` | `bash scripts/verify_repo.sh` | this repo |
