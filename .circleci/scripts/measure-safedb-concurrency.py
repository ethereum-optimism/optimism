#!/usr/bin/env python3
import argparse
import concurrent.futures
import csv
import hashlib
import json
import os
import re
from pathlib import Path
import subprocess
import threading
import tempfile
import time


class CgroupMemorySampler:
    def __init__(self, cgroup_path, output_path):
        if (cgroup_path / "memory.current").exists():
            self.anon_key = "anon"
            self.counter_path = cgroup_path / "memory.events"
            self.current_path = cgroup_path / "memory.current"
            self.limit_path = cgroup_path / "memory.max"
            self.peak_path = cgroup_path / "memory.peak"
        else:
            self.anon_key = "total_rss"
            self.counter_path = cgroup_path / "memory.failcnt"
            self.current_path = cgroup_path / "memory.usage_in_bytes"
            self.limit_path = cgroup_path / "memory.limit_in_bytes"
            self.peak_path = cgroup_path / "memory.max_usage_in_bytes"
        self.stat_path = cgroup_path / "memory.stat"
        self.memory_limit = int(self.limit_path.read_text().strip())
        self.output_path = output_path
        self.stop_event = threading.Event()
        self.thread = None
        self.error = None
        self.peak_anon = 0
        self.peak_current = 0
        self.samples = 0

    def counters(self):
        text = self.counter_path.read_text().strip()
        if self.counter_path.name == "memory.failcnt":
            return {"allocation_failures": int(text)}
        return {key: int(value) for key, value in (line.split() for line in text.splitlines())}

    def kernel_peak(self):
        return int(self.peak_path.read_text().strip()) if self.peak_path.exists() else None

    def sample(self):
        current = int(self.current_path.read_text().strip())
        stats = dict(line.split() for line in self.stat_path.read_text().splitlines())
        anon = int(stats[self.anon_key])
        self.peak_anon = max(self.peak_anon, anon)
        self.peak_current = max(self.peak_current, current)
        self.samples += 1
        self.writer.writerow([time.time(), current, anon, self.memory_limit])
        self.output.flush()

    def collect(self):
        try:
            while not self.stop_event.wait(1):
                self.sample()
        except Exception as error:
            self.error = str(error)

    def start(self):
        self.initial_counters = self.counters()
        self.initial_kernel_peak = self.kernel_peak()
        self.output = self.output_path.open("w", newline="")
        self.writer = csv.writer(self.output)
        self.writer.writerow(["timestamp", "memory_current_bytes", "anonymous_bytes", "limit_bytes"])
        self.sample()
        self.thread = threading.Thread(target=self.collect, daemon=True)
        self.thread.start()

    def finish(self):
        self.stop_event.set()
        if self.thread:
            self.thread.join()
        try:
            self.sample()
            final_counters = self.counters()
            result = {
                "counter_deltas": {key: value - self.initial_counters.get(key, 0) for key, value in final_counters.items()},
                "kernel_peak_after_bytes": self.kernel_peak(),
                "kernel_peak_before_bytes": self.initial_kernel_peak,
                "kernel_peak_scope": "cgroup lifetime; not reset",
                "limit_bytes": self.memory_limit,
                "sample_count": self.samples,
                "sampled_peak_anonymous_bytes": self.peak_anon,
                "sampled_peak_current_bytes": self.peak_current,
                "sampling_interval_seconds": 1,
            }
            if self.error:
                raise RuntimeError(self.error)
            return result
        finally:
            self.output.close()


def run_phase(name, command, environment, output_path):
    started = time.monotonic()
    print(f"Starting {name}: {command}", flush=True)
    with output_path.open("w") as output:
        try:
            exit_code = subprocess.run(command, env=environment, stdout=output, stderr=subprocess.STDOUT, check=False).returncode
        except OSError as error:
            output.write(str(error) + "\n")
            exit_code = 127
    finished = time.monotonic()
    print(f"Completed {name}: exit={exit_code}, elapsed={finished - started:.3f}s", flush=True)
    return {
        "command": command,
        "elapsed_seconds": finished - started,
        "exit_code": exit_code,
        "finished_monotonic": finished,
        "started_monotonic": started,
    }


def run_trial(mode, commands, environments, output_path, cgroup_path):
    output_path.mkdir(parents=True, exist_ok=True)
    summary = {"compile_inclusive": True, "mode": mode, "phases": {}}
    sampler = None
    started = time.monotonic()
    try:
        sampler = CgroupMemorySampler(cgroup_path, output_path / "memory.csv")
        sampler.start()
        if mode == "sequential":
            for name in ["resync", "normal"]:
                summary["phases"][name] = run_phase(name, commands[name], environments[name], output_path / f"{name}.log")
        else:
            with concurrent.futures.ThreadPoolExecutor(max_workers=2) as executor:
                futures = {
                    name: executor.submit(run_phase, name, commands[name], environments[name], output_path / f"{name}.log")
                    for name in ["resync", "normal"]
                }
                for name, future in futures.items():
                    summary["phases"][name] = future.result()
    except Exception as error:
        summary["error"] = str(error)
    finally:
        summary["elapsed_seconds"] = time.monotonic() - started
        if sampler and sampler.thread:
            try:
                summary["memory"] = sampler.finish()
            except Exception as error:
                summary["measurement_error"] = str(error)
        if len(summary["phases"]) == 2:
            phases = list(summary["phases"].values())
            summary["overlap_seconds"] = max(0, min(phase["finished_monotonic"] for phase in phases) - max(phase["started_monotonic"] for phase in phases))
        summary["success"] = (
            "error" not in summary
            and "measurement_error" not in summary
            and "memory" in summary
            and not any(value > 0 for key, value in summary.get("memory", {}).get("counter_deltas", {}).items() if key in {"allocation_failures", "oom", "oom_kill", "oom_group_kill"})
            and len(summary["phases"]) == 2
            and all(phase["exit_code"] == 0 for phase in summary["phases"].values())
        )
        (output_path / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
    return summary


def count_tests(path):
    counts = {"fail": 0, "pass": 0, "run": 0, "skip": 0}
    names = set()
    with path.open() as events:
        for line in events:
            event = json.loads(line)
            if event.get("Test") and "/" not in event["Test"] and event.get("Action") in counts:
                counts[event["Action"]] += 1
                if event["Action"] in {"run", "skip"}:
                    names.add(event["Test"])
    return {"counts": counts, "names": sorted(names)}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("mode", choices=["sequential", "parallel", "parallel-bounded"])
    arguments = parser.parse_args()
    client = os.environ["DEVSTACK_L2CL_KIND"]
    if client not in {"kona-node", "op-node"}:
        raise RuntimeError(f"Unsupported consensus client: {client}")
    label = "kona" if client == "kona-node" else "opnode"
    selection = Path(f".circleci/benchmarks/safedb-{label}-tests.txt")
    names = selection.read_text().splitlines()
    if not names or len(set(names)) != len(names) or any(not re.fullmatch(r"Test[A-Za-z0-9_]+", name) for name in names):
        raise RuntimeError("Invalid frozen test selection")
    output = Path(f"op-acceptance-tests/logs/concurrency-{label}-{arguments.mode}")
    output.mkdir(parents=True, exist_ok=True)
    results = Path("op-acceptance-tests/results")
    results.mkdir(parents=True, exist_ok=True)
    normal_workers = 7 if arguments.mode == "parallel-bounded" else 8
    build_cache = tempfile.mkdtemp(prefix="safedb-benchmark-go-cache-")
    commands = {}
    environments = {}
    for phase in ["normal", "resync"]:
        (output / phase).mkdir(parents=True, exist_ok=True)
        environments[phase] = dict(os.environ, GOCACHE=build_cache, LOG_LEVEL="info")
        command = [
            "bash", "ops/scripts/gotestsum-split.sh", "--format", "testname",
            "--jsonfile", str(output / phase / "events.json"),
            "--junitfile", str(results / f"benchmark-{label}-{arguments.mode}-{phase}.xml"),
            "--", "-count=1", f"-p={normal_workers if phase == 'normal' else 1}",
            "-parallel=1", f"-timeout={'30m' if phase == 'normal' else '10m'}",
        ]
        if phase == "normal":
            command += ["-run=^(" + "|".join(names) + ")$", "./op-acceptance-tests/tests/..."]
        else:
            environments[phase]["DEVSTACK_L2EL_KIND"] = "op-geth"
            command += ["./op-acceptance-tests/tests/safeheaddb_elsync", "./op-acceptance-tests/tests/safeheaddb_clsync"]
        commands[phase] = command
    cgroup_path = Path("/sys/fs/cgroup")
    if not (cgroup_path / "memory.current").exists():
        cgroup_path /= "memory"
    summary = run_trial(arguments.mode, commands, environments, output, cgroup_path)
    summary["build_cache_policy"] = "initially empty private cache, shared only between phases in this trial"
    summary["client"] = client
    summary["normal_workers"] = normal_workers
    summary["resource_class"] = os.environ["BENCHMARK_RESOURCE_CLASS"]
    summary["source_commit"] = subprocess.check_output(["git", "rev-parse", "HEAD"], text=True).strip()
    summary["test_selection_sha256"] = hashlib.sha256(selection.read_bytes()).hexdigest()
    summary["selected_test_names"] = len(names)
    for phase in summary["phases"]:
        try:
            counts = count_tests(output / phase / "events.json")
            summary["phases"][phase]["test_counts"] = counts["counts"]
            summary["phases"][phase]["observed_test_names"] = counts["names"]
            summary["success"] = summary["success"] and counts["counts"]["run"] > 0
            if phase == "normal":
                missing = sorted(set(names) - set(counts["names"]))
                unexpected = sorted(set(counts["names"]) - set(names))
                summary["selection_validation"] = {"missing": missing, "unexpected": unexpected}
                summary["success"] = summary["success"] and not missing and not unexpected
        except Exception as error:
            summary["error"] = str(error)
            summary["success"] = False
    (output / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
    print(json.dumps(summary, indent=2), flush=True)
    return 0 if summary["success"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
