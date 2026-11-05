"""Render the before/after captures into HTML includes and a data file for the landing page.

Usage: ANTHROPIC_API_KEY=... python3 render.py <capture-dir> <docs-dir>
The capture dir holds before.txt, human.txt, agent.txt and plan.json, written by generate.sh. Token counts come from
Anthropic's token counting endpoint, which reads the text and generates nothing.
"""
import html
import json
import os
import re
import sys
import urllib.request
from pathlib import Path

SYMBOL = re.compile(r"^(?P<lead>[\s│├└─\[\] *-]*?)(?P<sym>[+~-]) (?P<rest>.*)$")
BADGES = {
    "[low]": '<span class="meta">[low]</span>',
    "[medium]": '<span class="badge-t medium">[medium]</span>',
    "[HIGH]": '<span class="badge-t high">[HIGH]</span>',
    "[CRITICAL]": '<b class="crit">[CRITICAL]</b>',
}
SYMBOL_CLASS = {"+": "add", "~": "chg", "-": "del"}
DESTROY = ("will be destroyed", "not in configuration", "must be replaced")
ACTION = {("create",): "create", ("update",): "update", ("delete",): "delete"}
MAX_ATTRS = 6
ROW = re.compile(r"(?P<sym>[+~-]) (?P<name>\S+) \[(?P<risk>\w+)\]")
MODULE = re.compile(r"▼ (?P<module>module\.\S+) \(")
TOKEN_MODEL = "claude-opus-5-5"


def raw(body: str) -> str:
    # Plan text can hold Liquid-looking braces; keep Jekyll from reading any of it.
    return "{% raw %}" + body + "{% endraw %}\n"


def row(index: int, inner: str, cls: str = "row") -> str:
    return f'<span class="{cls}" style="--i: {index}">{inner}</span>'


def colour(line: str) -> str:
    text = html.escape(line, quote=False)
    for badge, markup in BADGES.items():
        text = text.replace(badge, markup)
    match = SYMBOL.match(text)
    if match and not match["lead"].strip(" ").endswith("["):
        cls = SYMBOL_CLASS[match["sym"]]
        text = f'{match["lead"]}<span class="{cls}">{match["sym"]}</span> {match["rest"]}'
    if text.startswith("Plan:"):
        text = f'<span class="ok">{text}</span>'
    text = re.sub(r"(Overall risk: |Risk: )(CRITICAL|critical)", r'\1<b class="crit">\2</b>', text)
    return text


def before(lines: list[str]) -> str:
    out, hit = [], False
    for line in lines:
        text = html.escape(line, quote=False)
        starts = line.strip().startswith("# ") and any(k in line for k in DESTROY)
        if starts:
            hit = True
        elif hit and not line.strip().startswith(("#", "- resource")):
            hit = False
        out.append(f'<span class="hit">{text}</span>' if hit else text)
    return "\n".join(out)


def flatten(values: dict, marks: dict) -> dict:
    """One level of map flattening (tags.Owner); marks carries after_unknown or after_sensitive flags."""
    flat = {}
    for key, value in sorted(values.items()):
        mark = marks.get(key) if isinstance(marks, dict) else None
        if isinstance(value, dict):
            for sub, item in sorted(value.items()):
                sub_mark = mark.get(sub) if isinstance(mark, dict) else mark
                flat[f"{key}.{sub}"] = (item, sub_mark)
        else:
            flat[key] = (value, mark)
    return flat


def shown(value, mark, unknown: str) -> str | None:
    if mark is True:
        return unknown
    if isinstance(value, bool):
        return json.dumps(value)
    if isinstance(value, (str, int, float)):
        return str(value)
    return None


def attributes(change: dict, action: str) -> list[dict]:
    before = change.get("before") or {}
    after = change.get("after") or {}
    unknown = change.get("after_unknown") or {}
    sensitive = change.get("after_sensitive") or {}
    attrs = []
    if action == "update":
        old, new = flatten(before, {}), flatten(after, unknown)
        hidden = flatten(sensitive, {})
        for key in sorted(set(old) | set(new)):
            if key.startswith("tags_all"):
                continue
            was = shown(*old.get(key, (None, None)), "(known after apply)")
            now = shown(*new.get(key, (None, None)), "(known after apply)")
            if was == now:
                continue
            if hidden.get(key, (False,))[0] is True:
                was = now = "(sensitive)"
            attrs.append({"key": key, "before": was, "after": now})
    else:
        values = flatten(after, {}) if action == "create" else flatten(before, {})
        for key, (value, _) in values.items():
            if isinstance(value, bool):
                continue  # provider defaults; the values the config sets read better
            text = shown(value, None, "")
            if text in (None, "", "0") or key in ("id", "arn") or key.startswith("tags_all"):
                continue
            entry = {"key": key, "before": None, "after": text} if action == "create" else {"key": key, "before": text, "after": None}
            attrs.append(entry)
    return attrs[:MAX_ATTRS]


def tour(plan: dict, human_txt: str, risk: str) -> dict:
    """The rows the landing page's interactive tour plays with, in the order tfui's tree shows them."""
    badges, module = {}, ""
    for line in human_txt.split("\n"):
        if m := MODULE.search(line):
            module = m["module"]
        elif m := ROW.search(line):
            badges[f"{module}.{m['name']}"] = m["risk"]
    by_address = {c["address"]: c for c in plan["resource_changes"]}
    changes = []
    for address, badge in badges.items():
        change = by_address[address]
        action = ACTION[tuple(change["change"]["actions"])]
        changes.append({
            "address": address,
            "module": change.get("module_address", ""),
            "name": address[len(change.get("module_address", "")) + 1:] if change.get("module_address") else address,
            "type": change["type"],
            "action": action,
            "risk": badge,
            "attrs": attributes(change["change"], action),
        })
    summary = {k: sum(c["action"] == a for c in changes) for k, a in (("add", "create"), ("change", "update"), ("destroy", "delete"))}
    return {"changes": changes, "summary": summary, "risk": risk.upper()}


def tokens(text: str) -> int:
    """Claude input tokens for text sent as one user message, as the agent's context would receive it."""
    request = urllib.request.Request(
        "https://api.anthropic.com/v1/messages/count_tokens",
        data=json.dumps({"model": TOKEN_MODEL, "messages": [{"role": "user", "content": text}]}).encode(),
        headers={
            "x-api-key": os.environ["ANTHROPIC_API_KEY"],
            "anthropic-version": "2023-06-01",
            "content-type": "application/json",
        },
    )
    with urllib.request.urlopen(request, timeout=30) as response:
        return json.load(response)["input_tokens"]


def main(capture: Path, docs: Path) -> None:
    before_txt = (capture / "before.txt").read_text()
    human_txt = (capture / "human.txt").read_text().rstrip("\n")
    agent_txt = (capture / "agent.txt").read_text().rstrip("\n")
    plan = json.loads((capture / "plan.json").read_text())

    actions = [c["change"]["actions"] for c in plan["resource_changes"] if c["change"]["actions"] != ["no-op"]]
    risk = re.search(r"^Risk: (\w+)$", agent_txt, re.M)[1]

    includes = docs / "_includes" / "before-after"
    includes.mkdir(parents=True, exist_ok=True)
    (includes / "before.html").write_text(raw(before(before_txt.rstrip("\n").split("\n"))))
    human = [row(i, colour(line)) for i, line in enumerate(human_txt.split("\n"))]
    (includes / "human.html").write_text(raw("\n".join(human)))
    agent = [row(i, colour(line)) for i, line in enumerate(agent_txt.split("\n"))]
    (includes / "agent.html").write_text(raw("\n".join(agent)))

    data = {
        "before_lines": before_txt.count("\n"),
        "before_bytes": len(before_txt.encode()),
        "agent_lines": agent_txt.count("\n") + 1,
        "agent_bytes": len((agent_txt + "\n").encode()),
        "before_tokens": tokens(before_txt),
        "agent_tokens": tokens(agent_txt + "\n"),
        "token_model": TOKEN_MODEL,
        "changes": len(actions),
        "add": sum(a == ["create"] for a in actions),
        "change": sum(a == ["update"] for a in actions),
        "destroy": sum(a == ["delete"] for a in actions),
        "risk": risk.upper(),
    }
    tour_json = json.dumps(tour(plan, human_txt, risk), indent=1, sort_keys=True, ensure_ascii=False)
    (docs / "_data" / "tour.json").write_text(tour_json + "\n")
    lines = ["# Generated by demo/before-after/generate.sh. Do not edit by hand."]
    lines += [f"{key}: {value}" for key, value in data.items()]
    (docs / "_data" / "before_after.yml").write_text("\n".join(lines) + "\n")


if __name__ == "__main__":
    main(Path(sys.argv[1]), Path(sys.argv[2]))
