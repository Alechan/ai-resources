# Skills

Agent skills that can be loaded by Copilot CLI, Claude, and Codex to improve performance on specialized tasks.

---

## How Copilot CLI loads skills

Skills are folders containing a `SKILL.md` file. Copilot CLI reads skills from:

| Scope | Location |
|---|---|
| Personal (all projects) | `~/.copilot/skills/<skill-name>/` |
| Repository (current project only) | `.github/skills/<skill-name>/` |

When you submit a prompt, Copilot automatically decides which skills are relevant based on each skill's `description` field, and injects those `SKILL.md` files into context.

You can also invoke a skill explicitly:
```
Use the /ddctl-datadog-ops skill to check why the <service-name> monitor is alerting
```

### Required frontmatter

Every `SKILL.md` **must** have YAML frontmatter or Copilot will not load it:

```markdown
---
name: my-skill-name          # required — lowercase, hyphens
description: One sentence explaining what this skill does and when to use it.
---
```

Optional fields: `license`, `allowed-tools` (pre-approves shell tools — use with care).

---

## Skills in this repo

| Skill | Description |
|---|---|
| [`ddctl-datadog-ops`](./ddctl-datadog-ops/SKILL.md) | Query DataDog logs, metrics, and monitors via `ddctl` |
| [`gdrivectl-drive-ops`](./gdrivectl-drive-ops/SKILL.md) | Google Drive file operations via `gdrivectl` |
| [`datagrip-datasources`](./datagrip-datasources/SKILL.md) | Update DataGrip datasource definitions safely |
| [`vis-network-diagrams`](./vis-network-diagrams/SKILL.md) | Build interactive HTML network/graph diagrams using vis.js |
| [`go-tdd-workflow`](./go-tdd-workflow/SKILL.md) | Red-green-refactor TDD workflow for Go |
| [`jenkinsctl`](./jenkinsctl/SKILL.md) | Interact with Jenkins instances using the jenkinsctl CLI |
| [`lint-before-push`](./lint-before-push/SKILL.md) | Catch and fix lint issues before pushing a branch |
| [`pr-qlty-triage`](./pr-qlty-triage/SKILL.md) | Triage qlty.sh code quality findings for a PR |
| [`slackctl-conversation-ops`](./slackctl-conversation-ops/SKILL.md) | Safely export accessible Slack conversations with threads |
| [`gchatctl-conversation-ops`](./gchatctl-conversation-ops/SKILL.md) | Safely capture a Google Chat session, list spaces, search messages, and export from a permalink |

Skills from other repositories can be included by configuring additional skill-root directories; the installers use only this repository by default.

---

## Installing skills

Run the install script from this repo root:

```bash
bash scripts/install-skills.sh
```

The script copies skills from this repo into `~/.kiro/skills/` and `~/.copilot/skills/`. Set `ADDITIONAL_SKILL_SOURCES` to a newline-separated list of skill-root directories to include other sources. Matching destination skill directories are replaced, so back up local changes before re-running the script.

Re-running the script replaces matching skill directories.

If you add a new skill during an active Copilot session, reload without restarting:
```
/skills reload
```

---

## Adding a new skill

1. Create a directory: `skills/<your-skill-name>/`
2. Create `SKILL.md` with the required frontmatter (see above)
3. Run `bash scripts/install-skills.sh` to install it
4. Add it to `docs/RESOURCE_CATALOG.md`

Skill names must be lowercase and hyphen-separated. The `name` in frontmatter should match the directory name.

---

## Debugging

```
/skills list          # show all loaded skills and their source paths
/skills info <name>   # show description, location, and status of one skill
/skills               # toggle skills on/off interactively
```

If a skill isn't showing up, check that:
- The `SKILL.md` has valid YAML frontmatter with both `name` and `description`
- The skill directory is symlinked (or present) under `~/.copilot/skills/`
- You've run `/skills reload` if the skill was added mid-session
