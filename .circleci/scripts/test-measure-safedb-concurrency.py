import importlib.util
import json
import os
from pathlib import Path
import sys
import subprocess
import tempfile
import unittest
from unittest.mock import patch

specification = importlib.util.spec_from_file_location("measurement", Path(__file__).with_name("measure-safedb-concurrency.py"))
measurement = importlib.util.module_from_spec(specification)
specification.loader.exec_module(measurement)


class MeasurementTests(unittest.TestCase):
    def run_measurement(self, mode, normal_exit=0, resync_exit=0, collector_error=False, oom=False):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            cgroup = root / "cgroup"
            cgroup.mkdir()
            for name, value in {
                "memory.current": "100",
                "memory.events": "oom 0\noom_kill 0\n",
                "memory.max": "1000",
                "memory.peak": "120",
                "memory.stat": "anon 80\n",
            }.items():
                (cgroup / name).write_text(value)
            commands = {
                name: [sys.executable, "-c", f"import time; time.sleep(0.1); raise SystemExit({exit_code})"]
                for name, exit_code in [("normal", normal_exit), ("resync", resync_exit)]
            }
            environments = {name: dict(os.environ) for name in commands}
            original_finish = measurement.CgroupMemorySampler.finish

            def finish(sampler):
                if oom:
                    (cgroup / "memory.events").write_text("oom 1\noom_kill 1\n")
                result = original_finish(sampler)
                if collector_error:
                    raise RuntimeError("injected collector failure")
                return result

            with patch.object(measurement.CgroupMemorySampler, "finish", finish):
                summary = measurement.run_trial(mode, commands, environments, root / "output", cgroup)
            self.assertEqual(summary, json.loads((root / "output/summary.json").read_text()))
            self.assertTrue((root / "output/memory.csv").exists())
            for name in commands:
                self.assertTrue((root / f"output/{name}.log").exists())
            return summary

    def test_execution_modes(self):
        for mode in ["sequential", "parallel", "parallel-bounded"]:
            with self.subTest(mode=mode):
                summary = self.run_measurement(mode)
                self.assertTrue(summary["success"])
                if mode == "sequential":
                    self.assertLessEqual(summary["phases"]["resync"]["finished_monotonic"], summary["phases"]["normal"]["started_monotonic"])
                    self.assertEqual(summary["overlap_seconds"], 0)
                else:
                    self.assertGreater(summary["overlap_seconds"], 0)

    def test_child_failures_preserve_both_results(self):
        for mode in ["sequential", "parallel"]:
            for normal_exit, resync_exit in [(1, 0), (0, 1), (1, 1)]:
                with self.subTest(mode=mode, normal_exit=normal_exit, resync_exit=resync_exit):
                    summary = self.run_measurement(mode, normal_exit, resync_exit)
                    self.assertFalse(summary["success"])
                    self.assertEqual(summary["phases"]["normal"]["exit_code"], normal_exit)
                    self.assertEqual(summary["phases"]["resync"]["exit_code"], resync_exit)

    def test_collector_failure_invalidates_measurement(self):
        summary = self.run_measurement("parallel", collector_error=True)
        self.assertFalse(summary["success"])
        self.assertIn("injected collector failure", summary["measurement_error"])

    def test_oom_invalidates_measurement(self):
        summary = self.run_measurement("parallel", oom=True)
        self.assertFalse(summary["success"])
        self.assertEqual(summary["memory"]["counter_deltas"]["oom_kill"], 1)

    def test_event_selection_includes_skips_and_excludes_subtests(self):
        with tempfile.TemporaryDirectory() as temporary:
            events = Path(temporary) / "events.json"
            events.write_text("\n".join(json.dumps(event) for event in [
                {"Action": "run", "Test": "TestOne"},
                {"Action": "run", "Test": "TestOne/child"},
                {"Action": "skip", "Test": "TestTwo"},
                {"Action": "pass", "Test": "TestOne"},
            ]))
            result = measurement.count_tests(events)
        self.assertEqual(result["names"], ["TestOne", "TestTwo"])
        self.assertEqual(result["counts"], {"fail": 0, "pass": 1, "run": 1, "skip": 1})

    def test_phase_environment_preserves_client_overrides_through_bash(self):
        with tempfile.TemporaryDirectory() as temporary:
            bash_environment = Path(temporary) / "bash-env"
            bash_environment.write_text("export DEVSTACK_L2EL_KIND=op-reth\n")
            with patch.dict(os.environ, {"BASH_ENV": str(bash_environment), "DEVSTACK_L2EL_KIND": "op-reth"}):
                for phase, expected_client in [("normal", "op-reth"), ("resync", "op-geth")]:
                    with self.subTest(phase=phase):
                        environment = measurement.phase_environment(phase, temporary)
                        result = subprocess.check_output(["bash", "-c", 'printf %s "$DEVSTACK_L2EL_KIND"'], env=environment, text=True)
                        self.assertEqual(result, expected_client)
                        self.assertEqual(environment["GOCACHE"], temporary)

    def test_process_launch_failure_is_retained(self):
        with tempfile.TemporaryDirectory() as temporary:
            result = measurement.run_phase("missing", ["/nonexistent-safedb-benchmark-command"], dict(os.environ), Path(temporary) / "phase.log")
        self.assertEqual(result["exit_code"], 127)


if __name__ == "__main__":
    unittest.main()
