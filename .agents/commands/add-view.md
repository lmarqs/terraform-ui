---
allowed-tools: Bash(find:*), Read, Write, Edit, Bash(mise run:*)
description: Add a new frame (sub-view) to an existing plugin
---

## Add a new frame to an existing plugin

Create a new `Frame` within an existing plugin's frame stack. Frames are sub-views for inspect, filter, confirm, or custom interactions.

### Steps

1. **Identify the target plugin** in `plugins/<name>/`

2. **Create the frame file** at `plugins/<name>/<frame_name>_frame.go`

3. **Implement the Frame interface** (defined in `pkg/sdk/frame.go`):

   ```go
   type Frame interface {
       ID() string
       Update(msg tea.Msg) (Frame, tea.Cmd)
       View(width, height int) string
       Hints() []KeyHint
   }
   ```

   Reference: `pkg/sdk/frames/inspect.go` or `pkg/sdk/frames/confirm.go`

4. **Push the frame from the plugin's Update()**:

   ```go
   return p, func() tea.Msg { return sdk.FramePushMsg{Frame: newMyFrame()} }
   ```

5. **Handle frame lifecycle**:
   - Return `nil` from `Update` to pop (go back)
   - Return a different Frame to replace in-place
   - Return self for no change

6. **Run `mise run check:build && mise run check:lint` to verify**

### Key patterns

- Frames capture ALL input while active (topmost frame owns input)
- `esc` should pop the frame (return nil)
- Reusable frames in `pkg/sdk/frames/`: FilterFrame, InspectFrame, ConfirmFrame, ActionFrame, FormFrame
- Prefer reusable frames over custom ones when possible
- Hints are frame-specific and override the plugin's hints while active
