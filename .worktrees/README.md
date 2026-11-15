# Worktrees

This directory holds the repository's linked Git worktrees: extra checkouts of other branches that share this clone's
history, branches and stash. Git ignores everything here except this file and the two scripts below, so a worktree
never shows up in the main checkout's `git status`.

## The rule

A worktree's path is its branch: branch `<branch>` is checked out at `.worktrees/<branch>`. People, agents and loop
runners all follow it, so anyone can find a branch's worktree, and a branch with a `/`, such as `fix/login`, sits in
a subdirectory.

```sh
git worktree add .worktrees/<branch> -b <branch>   # a new branch, from the current commit
git worktree add .worktrees/<branch> <branch>      # an existing branch
git worktree remove .worktrees/<branch>            # refused while it has changes; --force removes them
git branch -d <branch>                             # once merged; -D deletes an unmerged branch
git worktree prune                                 # forgets worktrees whose directory was deleted by hand
```

## Setup and teardown

A new worktree holds only the committed files. `.worktrees/setup` makes the worktree it runs in ready to run: the team
adds the steps a fresh checkout needs, such as installing dependencies and tools, and copying or creating untracked
local files like `.env`. `.worktrees/teardown` stops and deletes what a worktree keeps outside Git, such as
containers, databases or caches. Both start out doing nothing, and both are committed, so each worktree carries its
own copy. Each runs from the worktree's root:

```sh
git worktree add .worktrees/<branch> -b <branch>
(cd .worktrees/<branch> && .worktrees/setup)
(cd .worktrees/<branch> && .worktrees/teardown) && git worktree remove .worktrees/<branch>
```

A loop runner or an agent harness runs them in the worktrees it creates and removes when its own configuration names
them.

## Traps

- Branches, the stash and anything else kept in the clone are shared by every worktree: stashing or deleting a
  branch in one reaches all of them.
