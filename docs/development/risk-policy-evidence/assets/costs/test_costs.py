"""Validate complete D-03c observations, without imposing performance budgets.

Usage: python3 test_costs.py [output-directory]
"""
import json
from pathlib import Path
import sys
import unittest


OUTPUT = Path(sys.argv.pop(1)) if len(sys.argv) > 1 else Path(__file__).with_name("output")
TARGETS = {"linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64"}


class CostEvidence(unittest.TestCase):
    def setUp(self):
        path = OUTPUT / "results.json"
        self.assertTrue(path.is_file(), "cost experiment has no recorded results.json observations")
        self.results = json.loads(path.read_text())

    def sample(self, sample):
        self.assertEqual(sample["status"], "ok", "failed samples cannot establish completed cost evidence")
        self.assertEqual(sample["exit"], 0)
        self.assertEqual(sample["external_limit"], "none")
        self.assertGreater(sample["wall_seconds"], 0)
        self.assertGreater(sample["peak_rss_kib"], 0)

    def test_four_release_targets_have_five_cold_and_warm_build_pairs(self):
        rows = self.results["builds"]
        expected = {(t, v, p) for t in TARGETS for v in ("baseline", "candidate") for p in ("cold", "warm")}
        self.assertEqual(len(rows), len(expected), "duplicate or missing build rows")
        self.assertEqual({(r["target"], r["variant"], r["phase"]) for r in rows}, expected)
        for row in rows:
            with self.subTest(target=row["target"], variant=row["variant"], phase=row["phase"]):
                self.assertEqual(len(row["samples"]), 5)
                for sample in row["samples"]:
                    self.sample(sample)
                    self.assertGreater(sample["size_bytes"], 0)
                    self.assertRegex(sample["sha256"], r"^[0-9a-f]{64}$")

    def test_native_runtime_has_independent_cold_and_warm_process_samples(self):
        rows = self.results["runtime"]
        expected = {(v, n, p) for v in ("candidate", "standalone", "overhead") for n in (1, 100, 1000) for p in ("cold", "warm")}
        self.assertEqual(len(rows), len(expected), "duplicate or missing runtime rows")
        self.assertEqual({(r["variant"], r["workload_n"], r["phase"]) for r in rows}, expected)
        for row in rows:
            with self.subTest(variant=row["variant"], n=row["workload_n"], phase=row["phase"]):
                self.assertEqual(row["target"], "linux/amd64")
                self.assertEqual(len(row["samples"]), 5)
                for sample in row["samples"]:
                    self.sample(sample)
                    metrics = sample["metrics"]
                    loops = 100 if row["phase"] == "warm" else 1
                    self.assertEqual(metrics["loops"], loops)
                    self.assertEqual(metrics["prepare_count"], 0 if row["variant"] == "overhead" else 1)
                    self.assertEqual(metrics["correct_count"], loops * row["workload_n"])
                    for field in ("parse_ns", "prepare_ns", "first_eval_ns", "eval_total_ns"):
                        self.assertGreaterEqual(metrics[field], 0, field)
                    if row["variant"] != "overhead":
                        self.assertGreater(metrics["prepare_ns"], 0)
                        self.assertGreater(metrics["first_eval_ns"], 0)
                        self.assertGreater(metrics["eval_total_ns"], 0)

    def test_external_limits_are_verified_before_measurement(self):
        controls = self.results["controls"]
        self.assertTrue(controls["allocation_rejected"])
        self.assertEqual(controls["descendant_running_after_kill"], [])

    def test_comparison_records_resolved_graphs_toolchain_and_input_identity(self):
        provenance = self.results["provenance"]
        self.assertRegex(provenance["baseline_revision"], r"^[0-9a-f]{40}$")
        self.assertIn("go1.25.5", json.dumps(provenance["toolchain"]))
        self.assertEqual(set(provenance["targets"]), TARGETS)
        self.assertEqual(set(provenance["graphs"]), {"baseline", "candidate"})
        for variant, graph in provenance["graphs"].items():
            self.assertTrue(graph, f"{variant} resolved graph is missing")
        self.assertTrue(provenance["hashes"], "source, fixture and harness hashes must be recorded")
        for name, digest in provenance["hashes"].items():
            self.assertRegex(digest, r"^[0-9a-f]{64}$", name)


if __name__ == "__main__":
    unittest.main(verbosity=2)
