# Resource Catalog

| name | type | status | owner | install target | validation command | source/canonical repo |
| --- | --- | --- | --- | --- | --- | --- |
| gdrivectl-drive-ops | skill | active | platform | `~/.codex/skills/gdrivectl-drive-ops/SKILL.md` | `bash scripts/verify_repo.sh` | this repo |
| datagrip-datasources | skill | draft | platform | `~/.codex/skills/datagrip-datasources/SKILL.md` | `bash scripts/verify_repo.sh` | this repo |
| gdrivectl-drive-ops | agent | active | platform | `~/.claude/agents/gdrivectl-drive-ops.md` | `bash scripts/verify_repo.sh` | this repo |
| datagrip-datasources | agent | draft | platform | `~/.claude/agents/datagrip-datasources.md` | `bash scripts/verify_repo.sh` | this repo |
| gdrivectl | tool | active | platform | `go install github.com/Alechan/ai-resources/tools/gdrivectl/src/cmd/gdrivectl@latest` | `cd tools/gdrivectl/src && go test ./...` | this repo |
| datagrip-datasource-update | playbook | draft | data-platform | `playbooks/datagrip-datasource-update.md` | `bash scripts/verify_repo.sh` | this repo |
