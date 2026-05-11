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

Allow third-party plugins via hashicorp/go-plugin gRPC protocol. Plugins run as separate processes and communicate with tfui over a socket.

## Problem

All plugins must be compiled into the tfui binary. Users can't extend functionality without forking the project.

## Design (sketch)

- Use `hashicorp/go-plugin` (already proven in terraform, vault, packer)
- Plugin SDK as a separate Go module that external authors import
- Plugin discovery: scan `~/.tfui/plugins/` or configured paths
- Lifecycle: tfui spawns plugin process, negotiates protocol, sends messages
