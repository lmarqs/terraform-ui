---
layout: default
title: Roadmap
description: Planned features and enhancements for terraform-ui
---

# Roadmap

{% assign active = site.roadmap | where: "status", "active" %}
{% assign planned = site.roadmap | where: "status", "planned" %}
{% assign ideas = site.roadmap | where: "status", "idea" %}
{% assign completed = site.roadmap | where: "status", "completed" %}

{% if active.size > 0 %}
## Active

{% for item in active %}
- [{{ item.title }}]({{ item.url }}) — {{ item.priority }} priority, {{ item.effort }} effort
{% endfor %}
{% endif %}

{% if planned.size > 0 %}
## Planned

{% for item in planned %}
- [{{ item.title }}]({{ item.url }}) — {{ item.priority }} priority, {{ item.effort }} effort
{% endfor %}
{% endif %}

{% if ideas.size > 0 %}
## Ideas

{% for item in ideas %}
- [{{ item.title }}]({{ item.url }}) — {{ item.priority }} priority
{% endfor %}
{% endif %}

{% if completed.size > 0 %}
## Completed

{% for item in completed %}
- [{{ item.title }}]({{ item.url }})
{% endfor %}
{% endif %}
