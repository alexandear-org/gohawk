"""Check Make's instrumentation guard without running instrumented Go builds."""

import os
import importlib.util
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch


MAKEFILE = Path(__file__).resolve().parents[1] / "Makefile"
REPOSITORY = MAKEFILE.parent
REPLAY_SPEC = importlib.util.spec_from_file_location("precision_regression", REPOSITORY / "scripts/precision-regression.py")
REPLAY = importlib.util.module_from_spec(REPLAY_SPEC)
REPLAY_SPEC.loader.exec_module(REPLAY)


class CIPolicyTest(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name)
        (self.root / "Makefile").write_text(MAKEFILE.read_text())
        self.marker = self.root / "go-invoked"
        fake_go = self.root / "fake-go"
        fake_go.write_text('#!/bin/sh\nprintf "%s\\n" "$*" >> go-invoked\n')
        fake_go.chmod(0o700)
        (self.root / "scripts").mkdir()
        (self.root / "scripts/hyrule.sh").write_text(
            '#!/bin/sh\nprintf "%s\\n" "$*" >> generation-invoked\n'
        )
        self.environment = os.environ.copy()
        for name in ("CI", "GITHUB_ACTIONS", "MAKEFLAGS", "MFLAGS", "MAKELEVEL", "MAKEOVERRIDES", "GOFLAGS", "HYRULE_READY"):
            self.environment.pop(name, None)

    def run_make(self, *arguments, markers=None):
        return subprocess.run(
            ["make", "--no-print-directory", "GO=./fake-go", *arguments],
            cwd=self.root,
            env={**self.environment, **(markers or {})},
            text=True,
            capture_output=True,
            timeout=10,
        )

    def test_local_instrumentation_rejected_before_any_work(self):
        for target in ("coverage", "test-race", "ci", "verify-timed-coverage", "verify-timed-test-race", "verify-timed-ci"):
            for markers in ({}, {"CI": "true"}, {"GITHUB_ACTIONS": "true"}, {"CI": "false", "GITHUB_ACTIONS": "true"}):
                with self.subTest(target=target, markers=markers):
                    result = self.run_make("-j4", "build", target, markers=markers)
                    self.assertNotEqual(result.returncode, 0, result.stdout)
                    self.assertIn("CI-only", result.stderr)
                    self.assertFalse(self.marker.exists(), result.stdout)
                    self.assertFalse((self.root / ".build").exists())

    def test_local_instrumentation_flags_are_rejected(self):
        for flag in ("-race", "-cover", "-covermode=count", "-coverpkg=./...", "-coverprofile=coverage.out"):
            with self.subTest(flag=flag):
                result = self.run_make("test", "GOFLAGS=" + flag)
                self.assertNotEqual(result.returncode, 0, result.stdout)
                self.assertIn("CI-only", result.stderr)
                self.assertFalse(self.marker.exists())

    def test_github_actions_can_plan_instrumented_targets(self):
        # Dry runs verify CI wiring; no race or coverage test is executed here.
        for target, expected in (("coverage", "-coverprofile=coverage.out"), ("test-race", "test -race"), ("ci", "test -race")):
            with self.subTest(target=target):
                result = self.run_make("-n", target, markers={"CI": "true", "GITHUB_ACTIONS": "true"})
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertIn(expected, result.stdout)
                self.assertFalse(self.marker.exists())

    def test_ordinary_local_tests_remain_available(self):
        result = self.run_make("test")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.marker.read_text(), "run ./tools/gendocs -generated-markdown\ntest ./...\n")
        self.assertEqual((self.root / "generation-invoked").read_text(), "generate\n")


class GeneratedSourceBuildTest(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name)
        (self.root / "scripts").mkdir()
        (self.root / "benchmarks").mkdir()
        (self.root / "benchmarks/repositories.tsv").touch()
        self.marker = self.root / "order"
        self.generator = self.root / "scripts/hyrule.sh"
        self.fake_go = self.root / "go"
        self.fake_go.write_text('#!/bin/sh\nprintf "build\\n" >> "$BUILD_ORDER"\nexit "${BUILD_STATUS:-0}"\n')
        self.fake_go.chmod(0o700)
        self.environment = dict(os.environ, PATH=str(self.root) + os.pathsep + os.environ["PATH"], BUILD_ORDER=str(self.marker))
        for name in ("BUILD_STATUS", "GENERATION_STATUS"):
            self.environment.pop(name, None)
        self.generator.write_text('#!/bin/sh\nprintf "generate\\n" >> "$BUILD_ORDER"\nexit "${GENERATION_STATUS:-0}"\n')

    def run_replay_build(self):
        def run(command, **kwargs):
            return subprocess.run(command, env=self.environment, text=True, capture_output=True, **kwargs)
        with patch.object(REPLAY, "run", side_effect=run):
            REPLAY.build_gohawk(self.root, self.root / "binary")

    def test_replay_generates_before_building(self):
        self.run_replay_build()
        self.assertEqual(self.marker.read_text(), "generate\nbuild\n")

    def test_replay_stops_when_generation_fails(self):
        self.environment["GENERATION_STATUS"] = "19"
        with self.assertRaisesRegex(SystemExit, "source generation failed"):
            self.run_replay_build()
        self.assertEqual(self.marker.read_text(), "generate\n")

    def test_replay_keeps_build_failure(self):
        self.environment["BUILD_STATUS"] = "23"
        with self.assertRaisesRegex(SystemExit, "build failed"):
            self.run_replay_build()
        self.assertEqual(self.marker.read_text(), "generate\nbuild\n")

    def test_benchmark_generation_gates_its_build(self):
        subprocess.run(["git", "init", "-q", str(self.root)], check=True, capture_output=True)
        for generation_status, expected in (("19", "generate\n"), ("0", "generate\nbuild\n")):
            with self.subTest(generation_status=generation_status):
                self.marker.unlink(missing_ok=True)
                result = subprocess.run(
                    ["bash", str(REPOSITORY / "scripts/benchmark-dogfood.sh"), "--output", str(self.root / generation_status)],
                    cwd=self.root,
                    env=dict(self.environment, GENERATION_STATUS=generation_status, BUILD_STATUS="23"),
                    text=True, capture_output=True, timeout=10,
                )
                self.assertEqual(result.returncode, 19 if generation_status == "19" else 23, result.stderr)
                self.assertEqual(self.marker.read_text(), expected)


if __name__ == "__main__":
    unittest.main()
