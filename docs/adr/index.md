---
title: Decisions
seo_description: The architecture decision records behind tfui, and the reasoning behind each one.
---

# Decisions

Each record states one decision, the context that forced it, and its consequences. Read them in order to see how the architecture got its shape.

{% assign adrs = site.pages | where_exp: "p", "p.path contains 'adr/0'" | sort: "path" %}
{% for adr in adrs %}
- [{{ adr.title }}]({{ adr.url | relative_url }})
{%- endfor %}
