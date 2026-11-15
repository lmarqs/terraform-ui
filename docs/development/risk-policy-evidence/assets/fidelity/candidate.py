"""Disposable D-02a projection, not a production adapter or policy engine."""

import copy
import json


class Number(str):
    """Keep source number tokens separate from JSON strings."""


MISSING = object()


def load_plan(text):
    return json.loads(text, parse_int=Number, parse_float=Number)


def encode(value, unknown=False, sensitive=False):
    for mask in (unknown, sensitive):
        if not isinstance(mask, (bool, dict, list)):
            raise ValueError("invalid mask")
    if unknown is True:
        # A descendant sensitive mask also establishes sensitivity of this opaque subtree.
        def marked(mask):
            if isinstance(mask, dict):
                return any(marked(v) for v in mask.values())
            if isinstance(mask, list):
                return any(marked(v) for v in mask)
            return mask is True
        return {"kind": "sensitive_unknown" if marked(sensitive) else "unknown"}
    if sensitive is True:
        return {"kind": "sensitive"}
    for mask in (unknown, sensitive):
        if isinstance(mask, dict) and not isinstance(value, dict):
            raise ValueError("object mask on non-object")
        if isinstance(mask, list) and (not isinstance(value, list) or len(mask) != len(value)):
            raise ValueError("array mask shape mismatch")
    if isinstance(value, dict):
        um = unknown if isinstance(unknown, dict) else {}
        sm = sensitive if isinstance(sensitive, dict) else {}
        entries = {}
        for key in sorted(value.keys() | um.keys() | sm.keys()):
            entries[key] = encode(value.get(key, MISSING), um.get(key, False), sm.get(key, False))
        return {"kind": "object", "entries": entries}
    if isinstance(value, list):
        um = unknown if isinstance(unknown, list) else [False] * len(value)
        sm = sensitive if isinstance(sensitive, list) else [False] * len(value)
        return {"kind": "array", "items": [encode(v, um[i], sm[i]) for i, v in enumerate(value)]}
    if value is MISSING:
        raise ValueError("mask-only key has no established value")
    if value is None:
        return {"kind": "null"}
    if isinstance(value, bool):
        return {"kind": "boolean", "value": value}
    if isinstance(value, Number):
        return {"kind": "number", "value": str(value)}
    if isinstance(value, str):
        return {"kind": "string", "value": value}
    if isinstance(value, int):
        return {"kind": "number", "value": str(value)}
    raise ValueError("unsupported source value")


def project(resource):
    identity = {key: resource[key] for key in ("address", "mode", "type", "name", "provider_name")}
    identity["previous_address"] = resource.get("previous_address")
    identity["index"] = encode(resource["index"]) if "index" in resource else {"kind": "absent"}
    source = resource["change"]
    # Import masking requires separate acquisition evidence; do not silently guess it in this prototype.
    if source.get("importing") is not None:
        raise ValueError("import evidence outside this bounded prototype")
    actions = source["actions"]
    change = {"actions": copy.deepcopy(actions), "importing": None,
              "replace_paths": [[int(v) if isinstance(v, Number) else v for v in path]
                                for path in source.get("replace_paths", [])]}
    for side, absent_action in (("before", ["create"]), ("after", ["delete"])):
        value = source[side]
        unknown = source.get("after_unknown", False) if side == "after" else False
        sensitive = source.get(side + "_sensitive", False)
        if value is None and actions == absent_action and unknown is False and sensitive is False:
            change[side] = {"kind": "absent"}
        else:
            change[side] = encode(value, unknown, sensitive)
    return {"schema_version": "tfui.risk.input/v1", "resource": identity, "change": change}


def classify(envelope):
    """Synthetic observability probe; severities here are not proposed product rules."""
    entries = envelope["change"]["after"].get("entries", {})
    steps = entries.get("steps", {}).get("items", [])
    if steps and steps[0] == {"kind": "string", "value": "deny"}:
        level, code = "medium", "order-sensitive"
    elif "optional" not in entries:
        level, code = "high", "null-sensitive"
    else:
        level, code = "low", "baseline"
    return {"level": level, "reasons": [{"code": code, "message": "Fidelity probe", "paths": []}]}
