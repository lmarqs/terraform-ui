"""D-02a bounded candidate experiment, specified before prototype execution.

Pass: every hand-derived node and envelope matches, all numeric source tokens
survive exactly, arrays/null/absence/masks remain distinct, redacted markers never
reach candidate input/results, and neighbor insertion/removal leaves both equal.
Falsification: any silent loss outside opaque sensitive/unknown subtrees, a
leaked synthetic marker, a guessed inconsistent mask, neighbor dependence, or
phantom normalization erasing an order/null distinction rejects this candidate.

This uses stdlib unittest and an experimental deterministic classifier, NOT OPA
or production risk semantics. Classifier probe precedence: an after.steps array
starting with 'deny' returns medium/order-sensitive; absent after.optional returns
high/null-sensitive; otherwise low/baseline. Results have exactly level/reasons;
each reason has code, message='Fidelity probe', paths=[]. These deliberately
simple rules demonstrate observability of evidence, not risk correctness.

Run: python3 -B -m unittest discover -s
docs/development/risk-policy-evidence/assets/fidelity -p test_fidelity.py -v
"""

import copy
import importlib
import json
from pathlib import Path
import unittest


try:
    candidate = importlib.import_module("candidate")
except ModuleNotFoundError as error:
    if error.name != "candidate":
        raise
    candidate = None


SOURCE = Path(__file__).with_name("source-plan.json").read_text()


def result(level, code):
    return {"level": level, "reasons": [
        {"code": code, "message": "Fidelity probe", "paths": []}]}


class FidelityTests(unittest.TestCase):
    def setUp(self):
        self.assertIsNotNone(candidate, "candidate projection prototype is missing")
        self.resource = candidate.load_plan(SOURCE)["resource_changes"][0]

    def test_Project_GivenSourceEvidence_ShouldRetainExactEnvelope(self):
        envelope = candidate.project(self.resource)
        self.assertEqual(set(envelope), {"schema_version", "resource", "change"})
        self.assertEqual(envelope["schema_version"], "tfui.risk.input/v1")
        self.assertEqual(envelope["resource"], {
            "address": "module.app.aws_example.worker[9007199254740993]",
            "previous_address": "module.old.aws_example.worker[9007199254740993]",
            "mode": "managed", "type": "aws_example", "name": "worker",
            "provider_name": "registry.terraform.io/hashicorp/aws",
            "index": {"kind": "number", "value": "9007199254740993"}})
        self.assertEqual(set(envelope["change"]), {
            "actions", "importing", "replace_paths", "before", "after"})
        self.assertEqual(envelope["change"]["actions"], ["update"])
        self.assertIsNone(envelope["change"]["importing"])
        self.assertEqual(envelope["change"]["replace_paths"],
                         [["nested", "known"], ["sequence", 2]])
        self.assertEqual(envelope["change"]["before"], {"kind": "object", "entries": {
            "large": {"kind": "number", "value": "9007199254740993"},
            "exponent": {"kind": "number", "value": "1e3"},
            "steps": {"kind": "array", "items": [
                {"kind": "string", "value": "allow"},
                {"kind": "string", "value": "deny"}]},
            "optional": {"kind": "null"}, "enabled": {"kind": "boolean", "value": False},
            "secret": {"kind": "sensitive"}}})
        self.assertEqual(envelope["change"]["after"], {"kind": "object", "entries": {
            "large": {"kind": "number", "value": "9007199254740993"},
            "exponent": {"kind": "number", "value": "1e3"},
            "steps": {"kind": "array", "items": [
                {"kind": "string", "value": "allow"},
                {"kind": "string", "value": "deny"}]},
            "optional": {"kind": "null"}, "enabled": {"kind": "boolean", "value": True},
            "secret": {"kind": "sensitive"}, "future": {"kind": "unknown"},
            "secret_future": {"kind": "sensitive_unknown"},
            "nested": {"kind": "object", "entries": {
                "known": {"kind": "string", "value": "kept"},
                "password": {"kind": "sensitive"}, "later": {"kind": "unknown"}}},
            "sequence": {"kind": "array", "items": [
                {"kind": "null"}, {"kind": "string", "value": "visible"},
                {"kind": "sensitive"}]}, "opaque": {"kind": "sensitive"}}})

    def test_Project_GivenSensitiveMasks_ShouldExposeNoSyntheticMarkers(self):
        envelope = candidate.project(self.resource)
        capture = json.dumps({"input": envelope, "result": candidate.classify(envelope)})
        self.assertNotIn("SYNTHETIC_", capture)
        self.assertNotIn("hidden_key", capture)

    def test_Encode_GivenNestedMasks_ShouldRetainOnlyEstablishedShape(self):
        cases = [
            ({"x": None}, {"x": True}, False,
             {"kind": "object", "entries": {"x": {"kind": "unknown"}}}),
            ([None, "visible"], [True, False], [True, False],
             {"kind": "array", "items": [{"kind": "sensitive_unknown"},
                                          {"kind": "string", "value": "visible"}]}),
            ({"x": "hidden"}, True, False, {"kind": "unknown"}),
            ({"x": "hidden"}, False, True, {"kind": "sensitive"}),
            ({"x": "hidden"}, True, True, {"kind": "sensitive_unknown"}),
            ({}, {"future": True}, {"future": True},
             {"kind": "object", "entries": {"future": {"kind": "sensitive_unknown"}}}),
            ({}, False, {"secret": True},
             {"kind": "object", "entries": {"secret": {"kind": "sensitive"}}}),
        ]
        for value, unknown, sensitive, expected in cases:
            with self.subTest(expected=expected):
                self.assertEqual(candidate.encode(value, unknown=unknown, sensitive=sensitive), expected)

    def test_Encode_GivenInconsistentMask_ShouldFailInsteadOfGuessing(self):
        for value, unknown, sensitive in [("scalar", {"x": True}, False),
                                           ([1], [False, True], False),
                                           ({"x": 1}, False, [True])]:
            with self.subTest(value=value):
                with self.assertRaises(ValueError):
                    candidate.encode(value, unknown=unknown, sensitive=sensitive)

    def test_Project_GivenCreateDeleteOrNull_ShouldDistinguishAbsentSide(self):
        for actions, side, expected in [(["create"], "before", {"kind": "absent"}),
                                        (["delete"], "after", {"kind": "absent"}),
                                        (["update"], "before", {"kind": "null"}),
                                        (["update"], "after", {"kind": "null"})]:
            with self.subTest(actions=actions, side=side):
                resource = copy.deepcopy(self.resource)
                resource["change"]["actions"] = actions
                resource["change"][side] = None
                resource["change"][side + "_sensitive"] = False
                if side == "after":
                    resource["change"]["after_unknown"] = False
                self.assertEqual(candidate.project(resource)["change"][side], expected)

    def test_Project_GivenWholeUnknownAfter_ShouldCollapseDescendants(self):
        resource = copy.deepcopy(self.resource)
        resource["change"]["after_unknown"] = True
        for sensitive, expected in [(False, {"kind": "unknown"}),
                                     (True, {"kind": "sensitive_unknown"})]:
            resource["change"]["after_sensitive"] = sensitive
            self.assertEqual(candidate.project(resource)["change"]["after"], expected)

    def test_Project_GivenReplacementOrders_ShouldRetainBothSequences(self):
        for actions in [["delete", "create"], ["create", "delete"]]:
            with self.subTest(actions=actions):
                resource = copy.deepcopy(self.resource)
                resource["change"]["actions"] = actions
                self.assertEqual(candidate.project(resource)["change"]["actions"], actions)

    def test_Project_GivenIndexVariants_ShouldRetainIdentity(self):
        for index, expected in [(None, {"kind": "absent"}),
                                ("blue", {"kind": "string", "value": "blue"})]:
            resource = copy.deepcopy(self.resource)
            resource.pop("previous_address")
            resource.pop("index")
            if index is not None:
                resource["index"] = index
            identity = candidate.project(resource)["resource"]
            self.assertEqual(identity["index"], expected)
            self.assertIsNone(identity["previous_address"])

    def test_Classify_GivenNeighborInsertionRemoval_ShouldKeepInputAndResult(self):
        plan = candidate.load_plan(SOURCE)
        original = copy.deepcopy(plan)
        baseline = candidate.project(plan["resource_changes"][0])
        expected = result("low", "baseline")
        self.assertEqual(candidate.classify(baseline), expected)
        neighbor = copy.deepcopy(self.resource)
        neighbor["address"] = "aws_example.neighbor"
        neighbor["change"]["after"]["steps"] = ["deny", "allow"]
        for resources in [[neighbor, self.resource], [self.resource, neighbor], [self.resource]]:
            plan["resource_changes"] = resources
            target = next(r for r in resources if r["address"] == self.resource["address"])
            envelope = candidate.project(target)
            self.assertEqual(envelope, baseline)
            self.assertEqual(json.dumps(envelope, sort_keys=True), json.dumps(baseline, sort_keys=True))
            self.assertEqual(candidate.classify(envelope), expected)
        self.assertEqual(self.resource, original["resource_changes"][0])
        self.assertEqual(baseline, candidate.project(self.resource))

    def test_Classify_GivenPhantomArrayOrderCounterexample_ShouldSeeDistinctEvidence(self):
        baseline = candidate.project(self.resource)
        resource = copy.deepcopy(self.resource)
        resource["change"]["after"]["steps"] = ["deny", "allow"]
        changed = candidate.project(resource)
        self.assertNotEqual(changed, baseline)
        self.assertEqual(candidate.classify(baseline), result("low", "baseline"))
        self.assertEqual(candidate.classify(changed), result("medium", "order-sensitive"))

    def test_Classify_GivenPhantomNullCounterexample_ShouldSeeDistinctEvidence(self):
        baseline = candidate.project(self.resource)
        resource = copy.deepcopy(self.resource)
        del resource["change"]["after"]["optional"]
        changed = candidate.project(resource)
        self.assertNotIn("optional", changed["change"]["after"]["entries"])
        self.assertEqual(baseline["change"]["after"]["entries"]["optional"], {"kind": "null"})
        self.assertEqual(candidate.classify(baseline), result("low", "baseline"))
        self.assertEqual(candidate.classify(changed), result("high", "null-sensitive"))


if __name__ == "__main__":
    unittest.main()
