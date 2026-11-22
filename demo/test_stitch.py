"""Tests for stitch.py. Run with: mise run test:demo"""

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

STITCH = Path(__file__).resolve().parent / "stitch.py"


def stitch(frame_dir: Path, output: Path) -> subprocess.CompletedProcess:
    return subprocess.run(
        [sys.executable, str(STITCH), str(frame_dir), str(output)], capture_output=True, text=True
    )


def record(frame_dir: Path, names: list[str]) -> None:
    """Write a manifest that lists every name, and a frame file for each."""
    manifest = {
        "width": 20,
        "height": 3,
        "frames": [{"file": n, "delay_ms": 100, "width": 20, "height": 3} for n in names],
    }
    (frame_dir / "manifest.json").write_text(json.dumps(manifest))
    for name in names:
        (frame_dir / name).write_text(f"frame {name}")


class StitchTest(unittest.TestCase):
    def setUp(self) -> None:
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.frame_dir = Path(tmp.name)
        self.output = self.frame_dir / "out" / "demo.gif"

    def test_complete_recording_creates_the_gif(self) -> None:
        record(self.frame_dir, ["frame_0001.txt", "frame_0002.txt"])

        result = stitch(self.frame_dir, self.output)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(self.output.exists())

    def test_missing_frame_fails_naming_it_and_creates_no_gif(self) -> None:
        record(self.frame_dir, ["frame_0001.txt", "frame_0002.txt", "frame_0003.txt"])
        (self.frame_dir / "frame_0002.txt").unlink()

        result = stitch(self.frame_dir, self.output)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("frame_0002.txt", result.stderr)
        self.assertFalse(self.output.exists())


if __name__ == "__main__":
    unittest.main()
