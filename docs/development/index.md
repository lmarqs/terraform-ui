---
layout: default
title: Development
nav_order: 6
has_children: true
---

# Development

Internal architecture, design decisions, and testing strategy for contributors.

- [Architecture](architecture.md) — Plugin system, event bus, navigation model
- [Testing](testing.md) — Behavioral testing standard and layered test architecture
- [Architecture Decision Records](../adr/) — Design decisions and rationale

## Documentation Site Structure

The docs site at `docs/` is published to GitHub Pages using Jekyll with the [just-the-docs](https://just-the-docs.com/) theme. Navigation is controlled via front matter fields.

```
docs/
├── index.md                  # Home (landing page with demo GIFs)
├── roadmap.md                # Roadmap (top-level)
├── guides/                   # Getting Started section
│   ├── index.md              #   Section parent (nav_order: 2)
│   ├── getting-started.md    #   Quick Start guide
│   └── configuration.md     #   tfui.hcl reference
├── features/                 # Features section
│   ├── index.md              #   Section parent (nav_order: 3)
│   ├── risk-analysis.md
│   ├── phantom-changes.md
│   └── blast-radius.md
├── plugins/                  # Plugins section
│   ├── index.md              #   Section parent (nav_order: 4)
│   └── *.md                  #   One page per plugin
├── reference/                # Reference section
│   ├── index.md              #   Section parent (nav_order: 5)
│   ├── cli-reference.md
│   ├── cli-io-contract.md
│   ├── macro-language.md
│   ├── cli-ux.md
│   └── tui-ux.md
├── development/              # Development section (this page)
│   ├── index.md              #   Section parent (nav_order: 6)
│   ├── architecture.md
│   └── testing.md
├── adr/                      # ADRs (grandchildren of Development > Architecture)
├── _roadmap/                 # Jekyll collection (rendered by roadmap.md)
├── _config.yml               # Jekyll configuration
└── assets/demo/              # Demo GIFs referenced by index.md
```

### Navigation conventions

- **Section parents** use `has_children: true` in front matter
- **Child pages** use `parent: <Section Title>` matching the parent's `title:`
- **ADRs** use `grand_parent: Development` + `parent: Architecture` for 3-level nesting
- **Ordering** is controlled by `nav_order:` within each level
- **Plugin pages** use additional metadata: `id`, `key`, `category`
