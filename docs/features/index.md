---
layout: default
title: Features — Terraform Plan Analysis Tools
nav_order: 3
has_children: true
seo:
  description: Risk classification, phantom change detection, blast radius visualization, and AI agent integration for terraform plans. Automatic analysis with zero configuration.
description: "Lists the analyses tfui runs on a plan and the agent integration, one page each."
audience: "Users deciding what tfui adds over raw terraform"
---

# Features

tfui reads each plan and adds what raw terraform output leaves to you: a risk level per change, which diffs change nothing, and which modules a plan touches. It also gives agents a compact headless output.

- [Risk Analysis](risk-analysis.md) — Automatic classification of plan changes by risk level
- [Phantom Changes](phantom-changes.md) — Detect false-positive no-op changes
- [Blast Radius](blast-radius.md) — Visualize module-grouped impact of changes
- [AI Coding Agents](ai-agents.md) — Automatic headless mode with concise, truthful output for agents and CI
