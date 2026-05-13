// Package sdk provides the public contract for terraform-ui plugins.
//
// # Plugin Lifecycle
//
//	New(svc) → Init(ctx) → Activate() → Update(msg) loop → DeactivateMsg
//
// # Core Interfaces
//
//   - Plugin: base contract (ID, Name, Init, Update, View, Configure, Ready)
//   - Activatable: optional lifecycle hook for plugins that load data on entry
//   - Countable: reports filtered/total counts for border title display
//   - Hintable: supplies state-aware key hints for the status bar
//   - Stackable: exposes internal frame stack for navigation routing
//
// # Navigation (Stack + Frames)
//
//   - Frame: composable view layer (ID, Update, View, Hints)
//   - Stack: LIFO frame manager — Push/Pop/Peek/Update/View/Hints
//   - Return nil from Frame.Update() to pop (back navigation)
//   - Return a different Frame to replace in-place
//
// # Reusable Frames (pkg/sdk/frames/)
//
//   - FilterFrame: fzf-style live filtering, consumes all printable keys
//   - InspectFrame: scrollable detail view with configurable action keys
//   - ConfirmFrame: y/n modal that blocks all other input
//   - FormFrame: labeled fields with j/k navigation and selectable actions
//
// # UI Primitives (pkg/sdk/ui/)
//
//   - Cursor: index-based selection with bounds checking and viewport windowing
//   - ExpandSet: tracks which list indices are expanded (showing detail)
//   - FuzzyFilter[T]: fzf-based filtering with multi-term AND matching
//   - tree.Tree: hierarchical navigation with expand/collapse/pin state
//
// # Shared Services
//
//   - ChdirGuard: detects chdir changes in Activate(), eliminates context-scoping boilerplate
//   - PinService: thread-safe pinning backed by Session, shared across plugins
//   - StalenessGuard: TTL-based cache freshness validation for destructive ops
//
// # Status Lifecycle
//
//   - StatusIdle → StatusLoading → StatusDone / StatusError
//   - Plugins extend from offset 10+: StatusShowingDetail = sdk.Status(10)
//
// # Reference Implementation
//
// See plugins/state/ for canonical usage of all SDK primitives.
package sdk
