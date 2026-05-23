# App struct decomposition

## Problem

`internal/ui/app.go` is too large — it handles routing, context management, header rendering, busy-guard, plugin lifecycle, config resolution, and navigation state in a single struct. Hard to reason about and test in isolation.

## Direction

Extract cohesive responsibilities into smaller collaborators:
- Context management (rebuild, replace, config resolution) → own type
- Navigation state (activePlugin, returnTo, navStack) → own type
- Header rendering → own type (partially done with header struct)
- Busy-guard logic → could live on registry or a dedicated guard

Keep App as the orchestrator that delegates to these collaborators.
