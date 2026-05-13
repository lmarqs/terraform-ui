---
title: External Plugins (gRPC)
status: idea
priority: low
created: 2026-05-11
effort: large
tags: [plugin, architecture, extensibility]
depends_on: []
---

## Summary

Allow third-party plugins that run as separate processes, communicating with tfui over gRPC.

## Need

All plugins must be compiled into the tfui binary. This means:
- Users can't add custom views for their org's specific needs (e.g., compliance checker, internal cost model)
- Contributing a plugin requires Go knowledge and forking the repo
- Plugin authors can't iterate independently of tfui's release cycle

Current workaround: fork the repo, add your plugin, maintain the fork forever.

## Expected UX

```yaml
# tfui.hcl
plugins:
  my-compliance:
    path: ~/.tfui/plugins/compliance-checker
    keybinding: "X"
```

- External plugins appear in the home menu alongside built-in ones
- Same keybindings, hints, navigation stack behavior
- Plugin crashes don't take down tfui (process isolation)
- `tfui plugin install <url>` fetches and registers a plugin binary

## Advantages

- Ecosystem growth without core team bottleneck
- Plugin authors choose any language (anything that speaks gRPC)
- Process isolation prevents plugin bugs from crashing tfui
- Independent versioning and release cycles
