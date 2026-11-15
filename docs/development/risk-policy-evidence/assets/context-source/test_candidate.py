"""Behavioral experiment for the source snapshot/publication candidate, not production tests."""
import hashlib
import importlib.util
from pathlib import Path
import shutil
import tempfile
import unittest
from unittest.mock import patch

MODULE = Path(__file__).with_name("candidate.py")
if MODULE.exists():
    spec = importlib.util.spec_from_file_location("candidate", MODULE)
    candidate = importlib.util.module_from_spec(spec)
    import sys
    sys.modules[spec.name] = candidate
    spec.loader.exec_module(candidate)
else:
    # Test-local red adapter: current absence means no snapshot or accepted publication.
    class candidate:
        @staticmethod
        def snapshot(path):
            return type("Unavailable", (), {"modules": {}, "digest": "unavailable"})()
        @staticmethod
        def select(*args, **kwargs):
            return type("Selection", (), {"path": None, "origin": None})()
        class Gate:
            result = None
            def begin(self, context, plan_id, source_path):
                return type("Ticket", (), {"snapshot": candidate.snapshot(source_path)})()
            def publish(self, ticket, result):
                return False

class SourceTests(unittest.TestCase):
    def setUp(self):
        scratch = Path(__file__).resolve().parents[5] / ".ralph-tui" / "d02c" / "tmp"
        scratch.mkdir(parents=True, exist_ok=True)
        self.tmp = tempfile.TemporaryDirectory(dir=scratch)
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.source = self.root / "policy"
        self.source.mkdir()
        (self.source / "main.rego").write_bytes(b"package x\n")

    def test_selection_preserves_origin_and_complete_replacement(self):
        member = self.root / "services" / "api"
        member.mkdir(parents=True)
        inherited = candidate.select(self.root, "policies/root", member)
        self.assertEqual(inherited.path, self.root / "policies/root")
        self.assertEqual(inherited.origin, self.root)
        override = candidate.select(self.root, "policies/root", member, "policies/member")
        self.assertEqual(override.path, member / "policies/member")
        self.assertEqual(override.origin, member)
        self.assertIsNone(candidate.select(self.root, "policies/root", member, "").path)
        self.assertIsNone(candidate.select(self.root).path)
        entries = [("p*", "first"), ("prod*", "second"), ("prod", "exact"), ("dev-*", "")]
        exact = candidate.select(self.root, "root", member, None, entries, "prod")
        self.assertEqual(exact.path, member / "exact")
        glob = candidate.select(self.root, "root", member, None, entries, "preview")
        self.assertEqual(glob.path, member / "first")
        self.assertIsNone(candidate.select(self.root, "root", member, None, entries, "dev-local").path)
        for invalid in ["   ", 12, False, []]:
            with self.subTest(invalid=invalid), self.assertRaises(ValueError):
                candidate.select(self.root, invalid)

    def test_selection_loads_winning_bytes_from_declaring_directory(self):
        member = self.root / "member"
        member.mkdir()
        for base, name, data in [(self.root, "root.rego", b"root"), (member, "member.rego", b"member"), (member, "prod.rego", b"prod")]:
            (base / name).write_bytes(data)
        for args, expected in [
            ((self.root, "root.rego", member), {"root.rego": b"root"}),
            ((self.root, "root.rego", member, "member.rego"), {"member.rego": b"member"}),
            ((self.root, "root.rego", member, "member.rego", [("prod", "prod.rego")], "prod"), {"prod.rego": b"prod"}),
        ]:
            self.assertEqual(candidate.snapshot(candidate.select(*args).path).modules, expected)

    def test_repeated_read_detects_mixed_generation_and_snapshot_is_immutable(self):
        with patch.object(candidate, "_read", side_effect=[{"main.rego": b"old"}, {"main.rego": b"new"}]):
            with self.assertRaises(ValueError):
                candidate.snapshot(self.source)
        snap = candidate.snapshot(self.source)
        with self.assertRaises(TypeError):
            snap.modules["main.rego"] = b"modified"
        helper = self.source / "helper.rego"
        helper.write_bytes(b"helper")
        with_helper = candidate.snapshot(self.source).digest
        helper.unlink()
        self.assertNotEqual(candidate.snapshot(self.source).digest, with_helper)
        helper.symlink_to(self.source / "main.rego")
        with self.assertRaises(ValueError):
            candidate.snapshot(self.source)

    def test_selection_edit_during_evaluation_rejects_publication_even_with_identical_bytes(self):
        other = self.root / "other"
        shutil.copytree(self.source, other)
        current = [self.source]
        # Adapt the old constructor only to expose the missing behavioral check as red.
        try:
            gate = candidate.Gate(resolve=lambda: current[0])
        except TypeError:
            gate = candidate.Gate()
        ticket = gate.begin(("project", "A", "prod"), "plan", current[0])
        current[0] = other
        self.assertFalse(gate.publish(ticket, "old selection risk"), "changed selection must reject publication even for identical module bytes")
        self.assertIsNone(gate.result)

    def test_snapshot_preserves_exact_bytes_and_framed_identity(self):
        snap = candidate.snapshot(self.source)
        self.assertEqual(snap.modules, {"main.rego": b"package x\n"})
        frame = b"tfui-risk-source-v1\0" + (1).to_bytes(8, "big")
        frame += (9).to_bytes(8, "big") + b"main.rego" + (10).to_bytes(8, "big") + b"package x\n"
        self.assertEqual(snap.digest, "sha256:" + hashlib.sha256(frame).hexdigest())

    def test_identity_tracks_production_bytes_and_names_not_location_or_test_files(self):
        original = candidate.snapshot(self.source).digest
        other = self.root / "other"
        shutil.copytree(self.source, other)
        self.assertEqual(candidate.snapshot(other).digest, original)
        (other / "main_test.rego").write_text("ignored")
        (other / "README").write_text("ignored")
        self.assertEqual(candidate.snapshot(other).digest, original)
        for name, data in [("main.rego", b"package x\r\n"), ("helper.rego", b"package helper\n")]:
            with self.subTest(change=name):
                (other / name).write_bytes(data)
                self.assertNotEqual(candidate.snapshot(other).digest, original)
        (self.source / "main.rego").rename(self.source / "renamed.rego")
        self.assertNotEqual(candidate.snapshot(self.source).digest, original)

    def test_directory_selection_ignores_rego_named_subdirectory_without_recursion(self):
        original = candidate.snapshot(self.source)
        nested = self.source / "ignored.rego"
        nested.mkdir()
        (nested / "nested.rego").write_bytes(b"package nested\n")
        actual = candidate.snapshot(self.source)
        self.assertEqual(actual.modules, {"main.rego": b"package x\n"})
        self.assertEqual(actual.digest, original.digest)

    def test_invalid_deleted_empty_and_symlink_source_never_fallback(self):
        empty = self.root / "empty"
        empty.mkdir()
        link = self.root / "linked"
        link.symlink_to(self.source, target_is_directory=True)
        (self.source / "test_test.rego").write_text("package x")
        paths = [self.root / "missing", empty, link, self.source / "test_test.rego"]
        ancestor = self.root / "ancestor"
        ancestor.symlink_to(self.root, target_is_directory=True)
        paths.append(ancestor / "policy")
        for path in paths:
            with self.subTest(path=path.name), self.assertRaises(ValueError):
                candidate.snapshot(path)
        (self.source / "main.rego").unlink()
        with self.assertRaises(ValueError):
            candidate.snapshot(self.source)

    def test_switch_a_to_b_rejects_late_a_even_when_cancellation_is_ignored(self):
        gate = candidate.Gate()
        a = gate.begin(("project", "A", "prod"), "plan-a", self.source)
        b = gate.begin(("project", "B", "prod"), "plan-b", self.source)
        self.assertFalse(gate.publish(a, "A high"))
        self.assertIsNone(gate.result)
        self.assertTrue(gate.publish(b, "B low"))
        self.assertEqual(gate.result, "B low")
        self.assertFalse(gate.publish(a, "A late"))
        self.assertEqual(gate.result, "B low")

    def test_source_edit_keeps_ticket_bytes_but_rejects_publication(self):
        gate = candidate.Gate()
        ticket = gate.begin(("project", "A", "prod"), "plan", self.source)
        (self.source / "main.rego").write_bytes(b"package changed\n")
        self.assertEqual(ticket.snapshot.modules, {"main.rego": b"package x\n"})
        self.assertFalse(gate.publish(ticket, "old result"))
        self.assertIsNone(gate.result)

    def test_new_generation_clears_old_risk_and_deleted_source_cannot_publish(self):
        gate = candidate.Gate()
        context = ("project", "A", "prod")
        first = gate.begin(context, "plan", self.source)
        self.assertTrue(gate.publish(first, "old risk"))
        second = gate.begin(context, "plan", self.source)
        self.assertIsNone(gate.result)
        self.assertFalse(gate.publish(first, "late same-context risk"))
        (self.source / "main.rego").unlink()
        self.assertFalse(gate.publish(second, "deleted source risk"))
        self.assertIsNone(gate.result)

if __name__ == "__main__":
    unittest.main(verbosity=2)
