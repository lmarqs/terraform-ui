---
title: Decisions
seo:
  description: The architecture decision records behind tfui, and the reasoning behind each one.
description: "Lists every decision record behind tfui's architecture, in order."
audience: "Contributors about to change a design and checking what was decided"
---

# Decisions

Find the record for the part you are changing. Every record has the same shape:

- The opening paragraphs state the decision, then the problem that forced it.
- `## Consequences` says what the decision costs and what it requires of code written under it.
- `## Considered Options` lists the alternatives weighed and why each lost.

A record that a later record changed names the later one in its first paragraph.

{% assign adrs = site.pages | where_exp: "p", "p.path contains 'adr/0'" | sort: "path" %}
{% for adr in adrs %}
- [{{ adr.title }}]({{ adr.url | relative_url }})
{%- endfor %}
