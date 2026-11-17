"""Assert recorded D-03b observations; this is a disposable evidence check.

Usage: python3 test_controls.py [output-directory]
The runner must write results.json before these acceptance checks can pass.
"""
import json
from pathlib import Path
import sys
import unittest


OUTPUT = Path(sys.argv.pop(1)) if len(sys.argv) > 1 else Path(__file__).with_name("output")
MARKER = "D03_SYNTHETIC_SECRET_ONLY"
CHANNELS = ("stdout", "stderr", "print", "trace")


class BoundaryControls(unittest.TestCase):
    def setUp(self):
        evidence = OUTPUT / "results.json"
        self.assertTrue(evidence.is_file(), "boundary controls have no recorded results.json observations")
        self.results = json.loads(evidence.read_text())

    def case(self, name):
        self.assertIn(name, self.results["cases"], f"missing executed control: {name}")
        case = self.results["cases"][name]
        for field in ("prepare_error", "eval_error", "eval_error_code", "result_count",
                      "expression_count", "value", "validation", "external_limit", *CHANNELS):
            self.assertIn(field, case, f"{name} did not capture {field}")
        self.assertEqual(case["external_limit"], "none", f"{name} was externally limited, not completed")
        self.assertEqual(case["surviving_running_group_members"], [], name)
        self.assertTrue(case["pipes_drained"], name)
        self.assertEqual(case["truncated"], {"stdout": False, "stderr": False}, name)
        for channel in CHANNELS:
            self.assertIsInstance(case[channel], str)
        return case

    def prepared_and_evaluated(self, name):
        case = self.case(name)
        self.assertEqual(case["prepare_error"], "", name)
        self.assertEqual(case["eval_error"], "", name)
        return case

    def test_defined_valid_object_is_distinct_from_undefined_and_empty_values(self):
        valid = self.prepared_and_evaluated("valid")
        self.assertEqual(valid["validation"], "valid")
        self.assertEqual(valid["result_count"], 1)
        self.assertEqual(valid["expression_count"], 1)
        self.assertEqual(set(valid["value"]), {"level", "reasons"})
        undefined = self.prepared_and_evaluated("undefined")
        self.assertEqual(undefined["result_count"], 0)
        self.assertEqual(undefined["expression_count"], 0)
        self.assertEqual(undefined["validation"], "undefined")
        for name, value in (("empty_array", []), ("empty_object", {}), ("null", None), ("scalar", 42)):
            with self.subTest(name=name):
                case = self.prepared_and_evaluated(name)
                self.assertEqual(case["result_count"], 1)
                self.assertEqual(case["expression_count"], 1)
                self.assertEqual(case["value"], value)
                self.assertEqual(case["validation"], "invalid")

    def test_malformed_results_never_become_successful_assessments(self):
        for name in ("missing_reasons", "unknown_level"):
            with self.subTest(name=name):
                case = self.prepared_and_evaluated(name)
                self.assertEqual(case["result_count"], 1)
                self.assertEqual(case["validation"], "invalid")

    def test_compilation_failure_is_distinct_from_runtime_complete_rule_conflict(self):
        for name in ("parse_error", "unknown_builtin"):
            with self.subTest(name=name):
                case = self.case(name)
                self.assertTrue(case["prepare_error"])
                self.assertEqual(case["eval_error"], "")
                self.assertEqual(case["validation"], "not_run")
        conflict = self.case("conflict")
        self.assertEqual(conflict["prepare_error"], "")
        self.assertTrue(conflict["eval_error"])
        self.assertEqual(conflict["eval_error_code"], "eval_conflict_error")
        self.assertEqual(conflict["validation"], "not_run")

    def test_strict_builtin_errors_require_execution(self):
        strict = self.case("builtin_executed_strict")
        self.assertEqual(strict["prepare_error"], "")
        self.assertTrue(strict["eval_error"])
        self.assertEqual(strict["eval_error_code"], "eval_builtin_error")
        self.assertEqual(strict["validation"], "not_run")
        nonstrict = self.prepared_and_evaluated("builtin_executed_nonstrict")
        self.assertEqual(nonstrict["result_count"], 0)
        self.assertEqual(nonstrict["validation"], "undefined")
        for name in ("builtin_unreachable_strict", "builtin_unreachable_nonstrict"):
            with self.subTest(name=name):
                unreachable = self.prepared_and_evaluated(name)
                self.assertEqual(unreachable["result_count"], 1)
                self.assertEqual(unreachable["validation"], "valid")

    def test_denied_capabilities_fail_preparation_even_in_unqueried_rules(self):
        for builtin in ("http", "dns", "time", "rand", "uuid", "runtime", "trace"):
            for reachability in ("executed", "unreachable"):
                name = f"denied_{builtin}_{reachability}"
                with self.subTest(name=name):
                    case = self.case(name)
                    self.assertTrue(case["prepare_error"], name)
                    self.assertEqual(case["eval_error"], "")
                    self.assertEqual(case["validation"], "not_run")
                    for channel in CHANNELS:
                        self.assertEqual(case[channel], "", f"{name}: unexpected {channel}")
        self.assertEqual(self.results["http_requests"], 0)

    def test_print_erasure_is_recorded_separately_from_explicit_source_rejection(self):
        raw = self.prepared_and_evaluated("print_raw")
        self.assertEqual(raw["validation"], "valid")
        for channel in CHANNELS:
            self.assertEqual(raw[channel], "", f"print_raw: unexpected {channel}")
        rejected = self.case("print_rejected")
        self.assertTrue(rejected["prepare_error"])
        self.assertEqual(rejected["validation"], "not_run")
        for channel in CHANNELS:
            self.assertEqual(rejected[channel], "")

    def test_marked_values_are_absent_but_unmarked_and_source_literals_can_disclose(self):
        for name in ("secret_marked_echo", "secret_marked_interpolation"):
            with self.subTest(name=name):
                case = self.case(name)
                self.assertNotIn(MARKER, json.dumps(case), "marked value escaped redaction")
                self.assertEqual(case["prepare_error"], "")
        for name in ("secret_unmarked_interpolation", "secret_literal"):
            with self.subTest(name=name):
                case = self.prepared_and_evaluated(name)
                self.assertEqual(case["validation"], "valid")
                self.assertIn(MARKER, json.dumps(case["value"]), "trust-limit positive control did not disclose")


if __name__ == "__main__":
    unittest.main(verbosity=2)
