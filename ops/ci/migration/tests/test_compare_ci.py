#!/usr/bin/env python3
"""Real reporter fixtures plus synthetic corruption/retry tests, never runtime claims."""

import copy
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
import xml.etree.ElementTree as ET


SCRIPTS = Path(__file__).resolve().parents[1]
SPEC = importlib.util.spec_from_file_location("compare_ci", Path(__file__).resolve().parents[2] / 'migration' / 'compare-ci.py')
CI = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(CI)
SHA = "cb48622213e9aa8408de9904baafac784441ecc6"
PREFIX = "github.com/ethereum-optimism/optimism/op-node/rollup"
FEATURES = ["main", "CUSTOM_GAS_TOKEN", "OPTIMISM_PORTAL_INTEROP", "ZK_DISPUTE_GAME"]


class ComparisonTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.metadata = {
            "provider": "circleci", "sha": SHA, "branch": "codex/rwx-contracts-shadow",
            "routing_context": {"kind": "pr-branch", "base_branch": "develop"},
            "trigger": {"type": "webhook", "evidence": "synthetic unit fixture"},
            "workload": "go-rollup", "profile": "ci", "features": ["main"],
            "package_prefix": PREFIX, "test_config": {"tags": ["ci"], "short": False, "filter": None},
        }
        self.write_json("job.json", {"sha": SHA, "branch": self.metadata["branch"], "status": "success"})

    def write_json(self, name, data):
        path = self.root / name
        path.write_text(json.dumps(data) + "\n")
        return path

    def source(self, name="cases.json", format="circleci-tests", **fields):
        return {"id": name, "path": name, "format": format, "feature": "main", "role": "verdict",
                "metadata_path": "job.json", **fields}

    def rows(self):
        return [{"classname": PREFIX, "name": "TestSynthetic", "result": "success", "run_time": 1}]

    def collection(self, sources=None, **fields):
        if sources is None:
            self.write_json("cases.json", {"tests": self.rows()})
            sources = [self.source()]
        return {"version": 1, "metadata": copy.deepcopy(self.metadata), "sources": sources,
                "discovery": {"packages": [PREFIX], "complete": True, "provenance": "synthetic complete unit fixture"}, **fields}

    def normalized(self):
        return CI.normalize(self.collection(), self.root)

    def pair(self):
        left = self.normalized()
        right = copy.deepcopy(left)
        right["metadata"]["provider"] = "rwx"
        right["metadata"]["trigger"] = {"type": "push", "evidence": "synthetic unit fixture"}
        return left, right

    def partition_collection(self, *, empty=False):
        names = ['TestOne', 'TestTwo']
        groups = [names, []] if empty else [[name] for name in names]
        sources = []
        for index, assigned in enumerate(groups):
            events = [{'Action': 'start', 'Package': PREFIX}]
            for name in assigned:
                events.extend({'Action': action, 'Package': PREFIX, 'Test': name} for action in ('run', 'pass'))
            events.append({'Action': 'pass', 'Package': PREFIX})
            path = self.root / f'go-{index}.json'
            path.write_text('\n'.join(json.dumps(event) for event in events))
            sources.append(self.source(path.name, 'go-json', shard_index=index, shard_total=2))
        discovery = {'packages': [PREFIX], 'complete': True, 'provenance': 'synthetic exhaustive test listing',
                     'partition': 'test', 'total': 2, 'tests': [PREFIX + '::' + name for name in names],
                     'test_shards': [[PREFIX + '::' + name for name in group] for group in groups]}
        return self.collection(sources, discovery=discovery)

    def test_test_partitions_allow_shared_packages_but_preserve_exact_case_ownership(self):
        collection = self.partition_collection()
        normalized = CI.normalize(collection, self.root)
        self.assertEqual(len(normalized['cases']), 2)
        CI.validate_normalized(normalized)
        collection['discovery'] = {'packages': [PREFIX], 'complete': True, 'provenance': 'fixture'}
        with self.assertRaisesRegex(ValueError, 'duplicated package'):
            CI.normalize(collection, self.root)

    def test_test_partition_rejects_missing_duplicate_or_wrong_source_assignments(self):
        original = self.partition_collection()
        for kind in ('missing', 'duplicate', 'wrong_source'):
            collection = copy.deepcopy(original)
            groups = collection['discovery']['test_shards']
            if kind == 'missing': groups[0] = []
            elif kind == 'duplicate': groups[1] += groups[0]
            else: groups.reverse()
            with self.assertRaises(ValueError): CI.normalize(collection, self.root)

    def test_empty_test_partition_keeps_terminal_package_evidence(self):
        normalized = CI.normalize(self.partition_collection(empty=True), self.root)
        self.assertEqual(len(normalized['cases']), 2)
        CI.validate_normalized(normalized)
        normalized['sources'][1]['observed_packages'] = []
        with self.assertRaises(ValueError): CI.validate_normalized(normalized)

    def test_different_discovered_test_sets_are_compared(self):
        left = CI.normalize(self.partition_collection(), self.root)
        right = copy.deepcopy(left)
        right['metadata']['provider'] = 'rwx'
        right['cases'][1]['name'] = 'TestThree'
        right['discovery']['tests'][1] = PREFIX + '::TestThree'
        right['discovery']['test_shards'][1] = [PREFIX + '::TestThree']
        report = CI.compare(left, right)
        self.assertEqual(report['status'], 'different')
        self.assertIn('tests', report['discovery_differences'])

    def test_real_circleci_go_fixture_preserves_full_subtest_identity_and_skip_reasons(self):
        path = Path(__file__).parent / "fixtures/ci-comparison/circleci-go-cases.json"
        fixture = CI.read_json(path)
        packages = sorted({row["classname"] for row in fixture["cases"]})
        collection = self.collection([self.source(str(path))], discovery={"packages": packages})
        normalized = CI.normalize(collection, self.root)
        self.assertEqual(len(normalized["cases"]), 3)
        skips = [case for case in normalized["cases"] if case["outcome"] == "skip"]
        self.assertEqual({case["skip_reason"] for case in skips}, {"only applicable to span batches"})
        self.assertTrue(all("/" in case["name"] for case in skips))
        self.assertTrue(all(case["attempts"] is None for case in normalized["cases"]))

    def test_real_foundry_api_fixture_matches_junit_without_gas_decoration(self):
        path = Path(__file__).parent / "fixtures/ci-comparison/circleci-contract-cases.json"
        fixture = CI.read_json(path)
        self.metadata.update(workload="contracts-standard", profile="liteci", package_prefix=None)
        root = ET.Element("testsuites")
        suite = ET.SubElement(root, "testsuite")
        for row in fixture["cases"]:
            case = ET.SubElement(suite, "testcase", classname=row["classname"], name=row["name"], time="0")
            if row["result"] == "skipped":
                ET.SubElement(case, "skipped").text = row["message"]
        ET.ElementTree(root).write(self.root / "results.xml", encoding="unicode")
        left = CI.normalize(self.collection([self.source(str(path))], discovery=None), self.root)
        right = CI.normalize(self.collection([self.source("results.xml", "junit")], discovery=None), self.root)
        right["metadata"]["provider"] = "rwx"
        report = CI.compare(left, right)
        self.assertEqual(report["status"], "incomplete")  # Fixture is deliberately not a complete test-file inventory.
        self.assertFalse(report["different_cases"])
        self.assertFalse(report["missing_cases"])
        self.assertTrue(all("[SKIP:" not in case["skip_reason"] for case in left["cases"] if case["outcome"] == "skip"))

    def test_real_foundry_suite_identity_and_generic_or_prefixed_skips(self):
        # These are original reports from different SHAs, compared only at the
        # reporter-conversion boundary; no same-revision runtime claim is made.
        directory = Path(__file__).parent / "fixtures/ci-comparison"
        metadata = {**self.metadata, "package_prefix": None}
        source = self.source()
        api = CI.circleci_cases(directory / "circleci-foundry-skips.json", source, metadata)
        junit = CI.junit_cases(directory / "rwx-foundry-skips.xml", source, metadata)
        normalized_api = {CI.identity(case): case["skip_reason"] for case in api}
        normalized_junit = {CI.identity(case): case["skip_reason"] for case in junit}
        self.assertEqual(normalized_api, normalized_junit)
        self.assertEqual(len(normalized_junit), 4)
        self.assertEqual(sum(reason is None for reason in normalized_junit.values()), 2)
        self.assertIn("Skipping: standard configs incompatible with SUPER_ROOT_GAMES_MIGRATION", normalized_junit.values())
        self.assertTrue(all(suite.startswith("test/") and ":" in suite for _, suite, _ in normalized_junit))

    def test_canonical_fail_status_is_accepted_from_original_job_metadata(self):
        self.write_json("job.json", {"sha": SHA, "branch": self.metadata["branch"], "status": "fail"})
        normalized = CI.normalize(self.collection(), self.root)
        self.assertEqual(normalized["sources"][0]["status"], "fail")

    def test_same_count_can_hide_missing_and_extra_case(self):
        left, right = self.pair()
        right["cases"][0]["name"] = "TestSubstitute"
        report = CI.compare(left, right)
        self.assertEqual(report["status"], "different")
        self.assertEqual(report["missing_cases"][0]["name"], "TestSynthetic")
        self.assertEqual(report["extra_cases"][0]["name"], "TestSubstitute")

    def test_sha_profile_scope_and_routing_mismatch_are_incomparable(self):
        for field, value in (("sha", "a" * 40), ("profile", "liteci"), ("workload", "another-workload"),
                             ("routing_context", {"kind": "merge-group"})):
            left, right = self.pair()
            right["metadata"][field] = value
            with self.subTest(field=field):
                self.assertEqual(CI.compare(left, right)["status"], "incomparable")

    def test_different_raw_triggers_can_share_verified_routing_context(self):
        left, right = self.pair()
        report = CI.compare(left, right)
        self.assertEqual(report["status"], "equivalent")
        self.assertEqual(report["baseline"]["trigger"]["type"], "webhook")
        self.assertEqual(report["candidate"]["trigger"]["type"], "push")

    def test_missing_routing_context_discovery_or_revision_binding_is_incomplete(self):
        for missing in ("routing_context", "discovery", "revision_bound", "status"):
            left, right = self.pair()
            if missing == "routing_context":
                left["metadata"][missing] = right["metadata"][missing] = None
            elif missing == "discovery":
                right[missing] = None
            else:
                right["sources"][0][missing] = False if missing == "revision_bound" else None
            with self.subTest(missing=missing):
                self.assertEqual(CI.compare(left, right)["status"], "incomplete")

    def test_failed_or_canceled_job_cannot_hide_behind_passing_cases(self):
        for status in ("fail", "canceled"):
            left, right = self.pair()
            left["sources"][0]["status"] = right["sources"][0]["status"] = status
            report = CI.compare(left, right)
            self.assertEqual(report["status"], "different")
            self.assertEqual(len(report["unhealthy_sources"]), 2)

    def test_source_job_sha_branch_and_status_must_match(self):
        for change in ({"sha": "b" * 40}, {"branch": "develop"}, {"status": "failure"}):
            job = {"sha": SHA, "branch": self.metadata["branch"], "status": "success", **change}
            self.write_json("job.json", job)
            collection = self.collection()
            collection["sources"][0]["status"] = "success"
            with self.subTest(change=change), self.assertRaises(ValueError):
                CI.normalize(collection, self.root)

    def go_stream(self, name, events):
        (self.root / name).write_text("".join(json.dumps(event) + "\n" for event in events))
        return self.source(name, "go-json")

    def test_fail_then_pass_retry_is_separate_from_final_verdict(self):
        # Synthetic event stream exercises gotestsum retry behavior, not an observed flake.
        events = [{"Action": action, "Package": PREFIX, "Test": "TestSynthetic", "Elapsed": 1}
                  for action in ("run", "fail", "run", "pass")]
        events.insert(2, {"Action": "fail", "Package": PREFIX})
        events.append({"Action": "pass", "Package": PREFIX})
        data = CI.normalize(self.collection([self.go_stream("go.json", events)]), self.root)
        self.assertEqual(data["cases"][0]["outcome"], "pass")
        self.assertEqual([a["outcome"] for a in data["cases"][0]["attempts"]], ["fail", "pass"])
        self.assertFalse(data["package_failures"])
        left, _ = self.pair()
        data["metadata"]["provider"] = "rwx"
        report = CI.compare(left, data)
        self.assertEqual(report["status"], "equivalent")
        self.assertEqual(report["retries"]["candidate"]["observed_retry_attempts"], 1)
        self.assertEqual(report["retries"]["candidate"]["observed_package_retry_attempts"], 1)
        self.assertEqual(report["retries"]["candidate"]["retried_packages"][0]["outcomes"], ["fail", "pass"])
        self.assertEqual(report["retries"]["baseline"]["cases_without_attempt_evidence"], 1)

    def test_package_failure_outside_passing_case_is_unhealthy(self):
        events = [{"Action": "run", "Package": PREFIX, "Test": "TestSynthetic"},
                  {"Action": "pass", "Package": PREFIX, "Test": "TestSynthetic"},
                  {"Action": "fail", "Package": PREFIX}]
        right = CI.normalize(self.collection([self.go_stream("go.json", events)]), self.root)
        right["metadata"]["provider"] = "rwx"
        left, _ = self.pair()
        self.assertEqual(CI.compare(left, right)["status"], "different")

    def test_truncated_or_ambiguous_go_events_are_rejected(self):
        streams = [[{"Action": "run", "Package": PREFIX, "Test": "TestSynthetic"}],
                   [{"Action": "pass", "Package": PREFIX, "Test": "TestSynthetic"}],
                   [{"Action": "run", "Package": PREFIX, "Test": "TestSynthetic"}] * 2,
                   [{"Action": "run", "Package": PREFIX, "Test": "TestSynthetic"},
                    {"Action": "pass", "Package": PREFIX, "Test": "TestSynthetic"}]]
        for events in streams:
            with self.subTest(events=events), self.assertRaises(ValueError):
                CI.normalize(self.collection([self.go_stream("go.json", events)]), self.root)

    def test_complete_manifest_requires_terminal_no_test_packages_too(self):
        events = [{"Action": "run", "Package": PREFIX, "Test": "TestSynthetic"},
                  {"Action": "pass", "Package": PREFIX, "Test": "TestSynthetic"},
                  {"Action": "pass", "Package": PREFIX}]
        source = self.go_stream("go.json", events)
        discovery = {"packages": [PREFIX, PREFIX + "/no-tests"], "complete": True, "provenance": "synthetic manifest"}
        with self.assertRaisesRegex(ValueError, "do not match complete discovery"):
            CI.normalize(self.collection([source], discovery=discovery), self.root)
        events.append({"Action": "skip", "Package": PREFIX + "/no-tests"})
        source = self.go_stream("go.json", events)
        normalized = CI.normalize(self.collection([source], discovery=discovery), self.root)
        self.assertEqual(len(normalized["cases"]), 1)
        self.assertEqual(normalized["sources"][0]["observed_packages"], discovery["packages"])

    def test_empty_discovery_and_repeated_or_omitted_shards_are_rejected(self):
        collection = self.collection()
        collection["discovery"]["packages"] = []
        with self.assertRaises(ValueError):
            CI.normalize(collection, self.root)
        collection = self.collection()
        collection["sources"][0].update(shard_index=0, shard_total=2)
        with self.assertRaises(ValueError):
            CI.normalize(collection, self.root)
        self.write_json("second.json", {"tests": self.rows()})
        collection["sources"].append(self.source("second.json", shard_index=0, shard_total=2))
        with self.assertRaisesRegex(ValueError, "duplicated shard"):
            CI.normalize(collection, self.root)

    def test_duplicate_source_path_id_or_case_cannot_double_count_shards(self):
        for duplicate in (self.source(), self.source(id="another-id"), self.source("second.json")):
            self.write_json("second.json", {"tests": self.rows()})
            collection = self.collection()
            collection["sources"].append(duplicate)
            with self.subTest(duplicate=duplicate), self.assertRaises(ValueError):
                CI.normalize(collection, self.root)

    def test_diagnostic_shard_cannot_substitute_for_missing_verdict_shard(self):
        self.write_json("cases.json", {"tests": self.rows()})
        self.write_json("diagnostic.json", {"tests": self.rows()})
        collection = self.collection([self.source(shard_index=0, shard_total=2),
                                      self.source("diagnostic.json", role="diagnostic", shard_index=1, shard_total=2)])
        with self.assertRaisesRegex(ValueError, "verdict shard"):
            CI.normalize(collection, self.root)

    def test_legitimate_empty_component_share_needs_real_outside_scope_cases(self):
        self.write_json("outside.json", {"tests": [{**self.rows()[0], "classname": "example.org/another-package"}]})
        collection = self.collection([self.source(shard_index=0, shard_total=2),
                                      self.source("outside.json", shard_index=1, shard_total=2)])
        self.write_json("cases.json", {"tests": self.rows()})
        self.assertEqual(len(CI.normalize(collection, self.root)["cases"]), 1)
        self.write_json("outside.json", {"tests": []})
        with self.assertRaises(ValueError):
            CI.normalize(collection, self.root)

    def test_all_declared_contract_features_must_have_executed_cases(self):
        self.metadata.update(features=FEATURES, workload="contracts-standard", package_prefix=None)
        sources = []
        for feature in FEATURES:
            path = feature + ".json"
            self.write_json(path, {"tests": self.rows()})
            sources.append(self.source(path, feature=feature))
        collection = self.collection(sources)
        self.assertEqual(len(CI.normalize(collection, self.root)["cases"]), 4)
        collection["sources"].pop()
        with self.assertRaisesRegex(ValueError, "each declared feature"):
            CI.normalize(collection, self.root)

    def test_foundry_diagnostic_rerun_does_not_replace_failure(self):
        self.write_json("job.json", {"sha": SHA, "branch": self.metadata["branch"], "status": "failure"})
        self.write_json("failure.json", {"tests": [{**self.rows()[0], "result": "failure"}]})
        self.write_json("rerun.json", {"tests": self.rows()})
        data = CI.normalize(self.collection([self.source("failure.json"), self.source("rerun.json", role="diagnostic")]), self.root)
        self.assertEqual(len(data["cases"]), 1)
        self.assertEqual(data["cases"][0]["outcome"], "fail")

    def test_unknown_or_different_skip_reason_does_not_claim_parity(self):
        left, right = self.pair()
        for data in (left, right):
            data["cases"].append({**data["cases"][0], "name": "TestSkipped", "outcome": "skip", "skip_reason": None})
        self.assertEqual(CI.compare(left, right)["status"], "incomplete")
        left["cases"][1]["skip_reason"] = "feature disabled"
        right["cases"][1]["skip_reason"] = "wrong reason"
        self.assertEqual(CI.compare(left, right)["status"], "different")

    def test_malformed_xml_json_outcomes_pagination_and_durations_fail_closed(self):
        malformed = [{"tests": []}, {"tests": [{**self.rows()[0], "result": "mystery"}]},
                     {"tests": self.rows(), "next_page_token": "still-more"},
                     {"tests": [{**self.rows()[0], "run_time": -1}]},
                     {"tests": self.rows(), "sha": "a" * 40}]
        for payload in malformed:
            self.write_json("cases.json", payload)
            with self.subTest(payload=payload), self.assertRaises(ValueError):
                CI.normalize(self.collection([self.source()]), self.root)
        (self.root / "invalid.json").write_text('{"tests":[],"tests":[]}')
        with self.assertRaises(ValueError):
            CI.read_json(self.root / "invalid.json")
        for xml in ('<other/>', '<testsuite><testcase name="MissingClass"/></testsuite>',
                    '<!DOCTYPE testsuite><testsuite/>',
                    f'<testsuite><testcase classname="{PREFIX}" name="test"><failure/><skipped/></testcase></testsuite>'):
            (self.root / "results.xml").write_text(xml)
            with self.subTest(xml=xml), self.assertRaises(ValueError):
                CI.normalize(self.collection([self.source("results.xml", "junit")]), self.root)

    def test_duration_cache_resources_and_cost_remain_separate_unknowns(self):
        left, right = self.pair()
        right["measurements"] = {"wall_seconds": 10, "resources": [{"cpus": 8}],
                                 "cache": {"state": "warm", "historical_execution_seconds": 90}}
        report = CI.compare(left, right)
        supplied = report["measurements"]["candidate"]
        self.assertEqual(supplied["wall_seconds"], 10)
        self.assertIsNone(supplied["cpu_seconds"])
        self.assertIsNone(supplied["billed_seconds"])
        self.assertIsNone(supplied["cost"])
        self.assertEqual(supplied["cache"]["historical_execution_seconds"], 90)

    def test_cli_emits_machine_report_and_honest_nonzero_incomplete_or_incomparable(self):
        left, right = self.pair()
        baseline = self.write_json("baseline.json", left)
        candidate = self.write_json("candidate.json", right)
        command = [sys.executable, str(Path(__file__).resolve().parents[2] / 'migration' / 'compare-ci.py'), "compare", "--baseline", str(baseline),
                   "--candidate", str(candidate), "--output", str(self.root / "report.json"),
                   "--markdown", str(self.root / "report.md")]
        for expected, status in ((0, "equivalent"), (1, "incomplete"), (2, "incomparable")):
            if expected == 1:
                right["discovery"] = None
            if expected == 2:
                right["metadata"]["sha"] = "a" * 40
            self.write_json("candidate.json", right)
            result = subprocess.run(command, capture_output=True, text=True)
            self.assertEqual(result.returncode, expected, result.stderr)
            self.assertEqual(CI.read_json(self.root / "report.json")["status"], status)
            self.assertIn(status, (self.root / "report.md").read_text())


if __name__ == "__main__":
    unittest.main()
