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
Use the /ddctl-datadog-ops skill to check why the tapir monitor is alerting
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

Skills specific to the Mytheresa ecosystem live in the `mytheresa_ecosystem` repo under `skills/`.

---

## Installing skills

Run the install script from this repo root:

```bash
bash scripts/install-skills.sh
```

This creates symlinks in `~/.copilot/skills/` pointing back to the source directories in both this repo and the sibling `mytheresa_ecosystem` repo. Symlinks mean edits to `SKILL.md` files take effect immediately without re-running the script.

Re-running the script is safe — existing symlinks are replaced.

If you add a new skill during an active Copilot session, reload without restarting:
```
/skills reload
```

---

## Adding a new skill

1. Create a directory: `skills/<your-skill-name>/`
2. Create `SKILL.md` with the required frontmatter (see above)
3. Run `bash scripts/install-skills.sh` to symlink it
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
