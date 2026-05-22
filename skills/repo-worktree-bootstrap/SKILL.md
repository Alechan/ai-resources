---
name: repo-worktree-bootstrap
description: Generic Git cloning/bootstrap convention using a bare clone plus branch worktrees and a shared non-repo folder.
---

# repo-worktree-bootstrap

## Purpose

Provide a reusable local clone layout that scales to multiple active branches and avoids a single special checkout.

---

## Recommended local layout

Use one container directory per repository:

```
~/src/<repos-root>/<repo-name>/
  repo.git
  main
  <branch-name>
  wt-shared
```

Where:
- `repo.git` is the bare Git storage.
- `main` is the main-branch worktree.
- `<branch-name>` entries are feature worktrees.
- `wt-shared` stores non-repo local artifacts shared across worktrees.
- Each worktree should also contain a symlink named `wt-shared` pointing to `../wt-shared` so IDEs like GoLand show the shared folder when opening the worktree root.

---

## Working conventions

1. Run code/read commands from a worktree path (`.../main` or another branch worktree), not from the container root.
2. Keep `wt-shared` out of Git and use it for notes, generated artifacts, and temporary analysis outputs.
3. Create or refresh the `wt-shared` symlink inside every worktree before opening it in the IDE.
4. Ensure the repo-local exclude file contains `.idea/` and `wt-shared`; check first and append only if the patterns are missing.
5. Use absolute paths for destructive filesystem operations.
6. Clarify flatten operations explicitly:
   - **Recursive flatten:** move all nested files to root.
   - **One-level collapse:** move only immediate children of first-level directories.

---

## Minimal bootstrap flow

```bash
# Create bare repo
git clone --bare <repo-url> ~/src/<repos-root>/<repo-name>/repo.git

# Add main worktree
git --git-dir=~/src/<repos-root>/<repo-name>/repo.git \
  worktree add ~/src/<repos-root>/<repo-name>/main main

# Add feature worktree
git --git-dir=~/src/<repos-root>/<repo-name>/repo.git \
  worktree add ~/src/<repos-root>/<repo-name>/<branch-name> <branch-name>

# Create shared folder
mkdir -p ~/src/<repos-root>/<repo-name>/wt-shared

# Link shared folder into each worktree for IDE visibility
ln -sfn ../wt-shared ~/src/<repos-root>/<repo-name>/main/wt-shared
ln -sfn ../wt-shared ~/src/<repos-root>/<repo-name>/<branch-name>/wt-shared

# Add repo-local ignores once per repo container if missing
append_ignore() {
  local pattern="$1"
  local exclude_file="${HOME}/src/<repos-root>/<repo-name>/repo.git/info/exclude"
  grep -qxF "${pattern}" "${exclude_file}" || printf '%s\n' "${pattern}" >> "${exclude_file}"
}
append_ignore '.idea/'
append_ignore 'wt-shared'
```

---

## Notes

Org-specific path conventions (for example Mytheresa local clone roots) should be documented in the org/environment skill and can reference this skill for the generic pattern.
