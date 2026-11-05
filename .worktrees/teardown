#!/bin/sh
# Stops and deletes what the worktree it runs in keeps outside Git, before `git worktree remove`. It runs from the
# worktree's root, by hand or from a runner or hook configured with it, which keeps the worktree when it exits
# non-zero. Add the steps that undo what setup and use started: containers, databases, caches.
set -eu
