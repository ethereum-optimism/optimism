#!/usr/bin/env python3
"""Normalize retained CI reports and compare exact, same-revision test evidence.

No network, credentials or provider SDKs are needed. Collection paths are relative
to their JSON file. This compares reported cases; it does not prove that omitted
tests were discovered or that two providers billed the same amount of work.
"""

import argparse
from collections import Counter
import json
import math
from pathlib import Path
import re
import sys
import xml.etree.ElementTree as ET


OUTCOMES = {"pass", "fail", "error", "skip"}
RESULTS = {
    "success": "pass", "passed": "pass", "pass": "pass", "system-out": "pass",
    "failure": "fail", "failed": "fail", "fail": "fail",
    "error": "error", "skipped": "skip", "skip": "skip",
}
COMPARABLE = ("sha", "routing_context", "branch", "workload", "profile", "features", "test_config", "package_prefix")
MEASUREMENT_DEFAULTS = {
    "wall_seconds": None, "setup_seconds": None, "longest_shard_seconds": None,
    "summed_task_seconds": None, "cpu_seconds": None, "billed_seconds": None,
    "cost": None, "cache": None, "resources": None,
}


def terminal_status(value):
    if value is None or value in ("running", "pending", "queued", "unknown"):
        return None
    if value in ("success", "succeeded", "passed"):
        return "success"
    if value in ("fail", "failure", "failed", "error", "timedout", "timed_out"):
        return "fail"
    if value in ("canceled", "cancelled", "terminated"):
        return "canceled"
    raise ValueError(f"unsupported source status {value!r}")


def required_text(value, label):
    if not isinstance(value, str) or not value.strip() or value != value.strip():
        raise ValueError(f"{label} must be a nonempty string without surrounding whitespace")
    return value


def number(value, label):
    if value is None:
        return None
    if isinstance(value, bool) or not isinstance(value, (int, float)) or not math.isfinite(value) or value < 0:
        raise ValueError(f"{label} must be a finite nonnegative number or null")
    return value


def unique_list(values, label, *, allow_empty=False):
    if not isinstance(values, list) or (not values and not allow_empty):
        raise ValueError(f"{label} must be a nonempty list")
    for value in values:
        required_text(value, label)
    if len(values) != len(set(values)):
        raise ValueError(f"{label} contains duplicates")
    return sorted(values)


def json_object(value, label):
    if not isinstance(value, dict):
        raise ValueError(f"{label} must be an object")
    return value


def no_duplicate_keys(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError(f"duplicate JSON key: {key}")
        result[key] = value
    return result


def read_json(path):
    return json.loads(Path(path).read_text(encoding="utf-8"), object_pairs_hook=no_duplicate_keys,
                      parse_constant=lambda value: (_ for _ in ()).throw(ValueError(f"invalid JSON number: {value}")))


def validate_metadata(value):
    result = dict(json_object(value, "metadata"))
    if result.get("provider") not in ("circleci", "rwx"):
        raise ValueError("metadata.provider must be circleci or rwx")
    if not isinstance(result.get("sha"), str) or not re.fullmatch(r"[0-9a-f]{40}", result["sha"]):
        raise ValueError("metadata.sha must be a full lowercase commit SHA")
    for key in ("branch", "workload", "profile"):
        required_text(result.get(key), f"metadata.{key}")
    context = result.get("routing_context")
    if context is not None:
        json_object(context, "metadata.routing_context")
        required_text(context.get("kind"), "routing_context.kind")
    result["routing_context"] = context
    # Raw trigger provenance is intentionally separate from comparable routing.
    # A native push and a PR webhook can share a verified routing context. We
    # never infer that context from a webhook name or rewrite trigger evidence.
    trigger = result.get("trigger")
    if trigger is not None:
        json_object(trigger, "metadata.trigger")
        if trigger.get("type") is not None:
            required_text(trigger["type"], "metadata.trigger.type")
    result["trigger"] = trigger
    result["features"] = unique_list(result.get("features"), "metadata.features")
    if not json_object(result.get("test_config"), "metadata.test_config"):
        raise ValueError("metadata.test_config must describe the effective test settings")
    prefix = result.get("package_prefix")
    if prefix is not None:
        required_text(prefix, "metadata.package_prefix")
    result["package_prefix"] = prefix
    return result


def skip_reason(message):
    """Remove reporter decoration, preserving the reason rather than gas/timing."""
    if message is None or message == "":
        return None
    if not isinstance(message, str):
        raise ValueError("skip message must be a string or null")
    if message.lstrip().startswith("[SKIP]"):
        return None
    if "[SKIP:" in message:
        message = message.split("[SKIP:", 1)[0]
    message = re.sub(r"^\s*skipped:\s*", "", message)
    if message.strip() == "skipped":
        return None
    lines = []
    for line in message.splitlines():
        line = line.strip()
        if not line or re.match(r"(?:=== (?:RUN|PAUSE|CONT)|--- (?:SKIP|PASS|FAIL))\b", line):
            continue
        line = re.sub(r"^\S+\.go:\d+:\s*", "", line)
        lines.append(line)
    return " ".join(" ".join(lines).split()) or None


def identity(case):
    return case["feature"], case["suite"], case["name"]


def in_scope(suite, metadata):
    prefix = metadata.get("package_prefix")
    return prefix is None or suite == prefix or suite.startswith(prefix + "/")


def make_case(suite, name, outcome, duration, reason, source, attempts=None):
    if outcome not in OUTCOMES:
        raise ValueError(f"unsupported outcome {outcome!r}")
    return {
        "feature": source["feature"], "suite": required_text(suite, "case suite"),
        "name": required_text(name, "case name"), "outcome": outcome,
        "skip_reason": reason if outcome == "skip" else None,
        "duration_seconds": number(duration, "case duration_seconds"),
        "attempts": attempts, "source_id": source["id"],
    }


def circleci_cases(path, source, metadata):
    data = read_json(path)
    if isinstance(data, dict):
        if data.get("sha") is not None and data["sha"] != metadata["sha"]:
            raise ValueError("CircleCI case file SHA differs from collection SHA")
        if data.get("next_page_token"):
            raise ValueError("CircleCI test response is incomplete; collect every page first")
        fields = [key for key in ("items", "tests", "cases") if key in data]
        if len(fields) != 1:
            raise ValueError("CircleCI case JSON must have exactly one items/tests/cases list")
        data = data[fields[0]]
    if not isinstance(data, list):
        raise ValueError("CircleCI cases must be a list")
    cases = []
    for row in data:
        json_object(row, "CircleCI case")
        suite = required_text(row.get("classname"), "CircleCI classname")
        if not in_scope(suite, metadata):
            continue
        outcome = RESULTS.get(row.get("result"))
        if outcome is None:
            raise ValueError(f"unknown CircleCI case result {row.get('result')!r}")
        cases.append(make_case(suite, row.get("name"), outcome, row.get("run_time"),
                               skip_reason(row.get("message")) if outcome == "skip" else None, source))
    return cases


def junit_cases(path, source, metadata):
    # No entity expansion/DTDs: reports do not need external or custom entities.
    content = path.read_text(encoding="utf-8")
    if re.search(r"<!\s*(?:DOCTYPE|ENTITY)\b", content, re.IGNORECASE):
        raise ValueError("JUnit DTDs/entities are unsupported")
    root = ET.fromstring(content)
    if root.tag not in ("testsuite", "testsuites"):
        raise ValueError("JUnit root must be testsuite or testsuites")
    cases = []

    def walk(element, suite_name=None):
        if element.tag == "testsuite":
            suite_name = element.get("name")
        if element.tag == "testcase":
            yield element, suite_name
        for child in element:
            yield from walk(child, suite_name)

    for case, parent_suite in walk(root):
        # Foundry emits its path:contract identity on testsuite.name, whereas
        # gotestsum emits classname on each case. Never substitute a case name
        # or a file basename when neither identity is available.
        suite = required_text(case.get("classname") or parent_suite, "JUnit testcase classname or parent suite name")
        if not in_scope(suite, metadata):
            continue
        verdicts = [child for child in case if child.tag in ("failure", "error", "skipped")]
        if len(verdicts) > 1:
            raise ValueError("ambiguous JUnit testcase verdict")
        outcome = RESULTS[verdicts[0].tag] if verdicts else "pass"
        reason = None
        if outcome == "skip":
            child = verdicts[0]
            reason = skip_reason(child.get("message") or child.text)
        duration = case.get("time")
        cases.append(make_case(suite, case.get("name"), outcome, float(duration) if duration is not None else None,
                               reason, source))
    return cases


def go_cases(path, source, metadata):
    attempts, active, output = {}, {}, {}
    package_terminals = []
    observed_packages = set()
    for line_number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        if not line.strip():
            continue
        try:
            event = json.loads(line, object_pairs_hook=no_duplicate_keys)
        except ValueError as error:
            raise ValueError(f"Go JSON line {line_number}: {error}") from error
        json_object(event, "Go event")
        suite = required_text(event.get("Package"), "Go event Package")
        if not in_scope(suite, metadata):
            continue
        observed_packages.add(suite)
        action = event.get("Action")
        if action not in ("start", "run", "pause", "cont", "output", "pass", "fail", "skip", "build-output", "build-fail"):
            raise ValueError(f"unknown Go event action {action!r}")
        name = event.get("Test")
        if not name:
            if action in ("pass", "fail", "skip", "build-fail"):
                package_terminals.append((suite, action))
            continue
        required_text(name, "Go event Test")
        key = (suite, name)
        if action == "run":
            if key in active:
                raise ValueError(f"Go test started twice without a verdict: {key}")
            active[key] = True
            output[key] = []
        elif action == "output":
            value = event.get("Output")
            if not isinstance(value, str):
                raise ValueError("Go output event must contain string Output")
            if key in active:
                output[key].append(value)
        elif action in ("pass", "fail", "skip"):
            if key not in active:
                raise ValueError(f"Go verdict without matching run event: {key}")
            attempt = {"outcome": action, "duration_seconds": number(event.get("Elapsed"), "Go Elapsed"),
                       "skip_reason": skip_reason("".join(output[key])) if action == "skip" else None}
            attempts.setdefault(key, []).append(attempt)
            del active[key]
    if active:
        raise ValueError("Go JSON has tests without terminal verdicts")
    # A build/TestMain failure can coexist with passing case records. Preserve it
    # outside the case set so case parity cannot turn a failed run green.
    final_packages = dict(package_terminals)
    if observed_packages != set(final_packages):
        raise ValueError("Go JSON has packages without terminal verdicts")
    source["observed_packages"] = sorted(observed_packages)
    failures = [suite for suite, action in final_packages.items() if action in ("fail", "build-fail")]
    package_history = {}
    for suite, action in package_terminals:
        package_history.setdefault(suite, []).append(action)
    source["package_attempts"] = [{"suite": suite, "outcomes": history} for suite, history in sorted(package_history.items())]
    cases = []
    for (suite, name), history in attempts.items():
        final = history[-1]
        cases.append(make_case(suite, name, final["outcome"], final["duration_seconds"], final["skip_reason"], source, history))
    return cases, sorted(set(failures))


def validate_discovery(value):
    if value is None:
        return None
    result = dict(json_object(value, "discovery"))
    keys = [key for key in ("packages", "test_files") if key in result]
    if not keys:
        raise ValueError("discovery must contain packages or test_files")
    for key in keys:
        result[key] = unique_list(result[key], f"discovery.{key}")
    if result.get("complete") is not None and type(result["complete"]) is not bool:
        raise ValueError("discovery.complete must be boolean or null")
    if result.get("complete") is True:
        required_text(result.get("provenance"), "complete discovery provenance")
    partition = result.get('partition', 'package')
    if partition not in ('package', 'test'):
        raise ValueError('unsupported discovery partition')
    if partition == 'test':
        tests = unique_list(result.get('tests'), 'discovery.tests')
        for identity in tests:
            parts = identity.split('::')
            if len(parts) != 2 or parts[0] not in result.get('packages', []) or not re.fullmatch(r'Test\w*', parts[1]):
                raise ValueError('invalid discovered Go test identity')
        result['tests'] = tests
        total, groups = result.get('total'), result.get('test_shards')
        if type(total) is not int or total < 1 or not isinstance(groups, list) or len(groups) != total:
            raise ValueError('test discovery shards do not match total')
        assigned = []
        for group in groups:
            assigned.extend(unique_list(group, 'test shard', allow_empty=True))
        if sorted(assigned) != tests or len(assigned) != len(set(assigned)):
            raise ValueError('test discovery shards must cover every identity exactly once')
    if "shards" in result:
        total = result.get("total")
        if type(total) is not int or total < 1 or not isinstance(result["shards"], list) or len(result["shards"]) != total:
            raise ValueError("discovery shards do not match total")
        flattened = []
        for shard in result["shards"]:
            if not isinstance(shard, list) or any(not isinstance(value, str) for value in shard):
                raise ValueError("discovery shard must list package strings")
            flattened.extend(shard)
        if sorted(flattened) != result.get("packages") or len(flattened) != len(set(flattened)):
            raise ValueError("discovery shards must cover every package exactly once")
    return result


def measurements(value):
    result = {**MEASUREMENT_DEFAULTS, **json_object({} if value is None else value, "measurements")}
    for key in MEASUREMENT_DEFAULTS:
        if key.endswith("_seconds"):
            number(result[key], f"measurements.{key}")
    # Preserve supplied observations without computing CPU or billed time from
    # wall time. Units/source/scope belong beside any externally supplied price.
    return result


def validate_package_coverage(sources, discovery, cases=None):
    go_sources = [source for source in sources if source["role"] == "verdict" and source.get("format") == "go-json"]
    observed = []
    for source in go_sources:
        packages = source.get("observed_packages")
        if not isinstance(packages, list) or any(not isinstance(package, str) for package in packages):
            raise ValueError("Go source must retain observed terminal packages")
        if len(packages) != len(set(packages)):
            raise ValueError("duplicated observed package within a source")
        observed.extend(packages)
    if discovery and discovery.get('partition') == 'test':
        # Test-level sharding deliberately invokes every package on each node.
        # Validate exact case ownership instead of weakening package partitions.
        if not go_sources or cases is None or any(source.get('format') != 'go-json'
                                                 for source in sources if source['role'] == 'verdict'):
            raise ValueError('test partition coverage requires original Go events')
        for source in go_sources:
            if sorted(source['observed_packages']) != discovery['packages']:
                raise ValueError('test shard does not retain every selected terminal package')
            index = source.get('shard_index')
            if type(index) is not int or source.get('shard_total') != discovery['total']:
                raise ValueError('test source lacks its declared shard identity')
            expected = set(discovery['test_shards'][index])
            actual = {case['suite'] + '::' + case['name'] for case in cases
                      if case['source_id'] == source['id'] and '/' not in case['name']}
            if actual != expected:
                raise ValueError('test source verdicts differ from its assigned top-level identities')
            if any(case['suite'] + '::' + case['name'].split('/')[0] not in expected
                   for case in cases if case['source_id'] == source['id']):
                raise ValueError('subtest belongs to an unassigned top-level identity')
        return
    if len(observed) != len(set(observed)):
        raise ValueError("duplicated package across Go shard sources")
    if go_sources and discovery and discovery.get("complete") is True and "packages" in discovery:
        if sorted(observed) != discovery["packages"]:
            raise ValueError("observed terminal Go packages do not match complete discovery")


def validate_shards(sources, metadata):
    shards = set()
    for source in sources:
        if "shard_index" not in source and "shard_total" not in source:
            continue
        index, total = source.get("shard_index"), source.get("shard_total")
        if type(index) is not int or type(total) is not int or not 0 <= index < total:
            raise ValueError("invalid source shard index/total")
        if source["role"] != "verdict":
            continue
        shard = (source["feature"], index, total)
        if shard in shards:
            raise ValueError("duplicated shard input")
        shards.add(shard)
    for feature in metadata["features"]:
        totals = {total for f, _, total in shards if f == feature}
        if len(totals) > 1:
            raise ValueError("inconsistent source shard totals")
        for total in totals:
            if {index for f, index, t in shards if f == feature and t == total} != set(range(total)):
                raise ValueError("shard inputs do not cover every declared verdict shard")


def validate_cases(cases, metadata, sources):
    if not isinstance(cases, list) or not cases:
        raise ValueError("no cases in selected workload")
    seen, executed = set(), Counter()
    source_ids = {source["id"]: source for source in sources if source["role"] == "verdict"}
    for case in cases:
        json_object(case, "normalized case")
        for key in ("feature", "suite", "name", "source_id"):
            required_text(case.get(key), f"case.{key}")
        key = identity(case)
        if key in seen:
            raise ValueError(f"duplicate verdict case across sources: {key}")
        seen.add(key)
        if case["feature"] not in metadata["features"] or case["source_id"] not in source_ids:
            raise ValueError("case feature/source is outside collection")
        if case["feature"] != source_ids[case["source_id"]]["feature"]:
            raise ValueError("case feature does not match its source")
        if not in_scope(case["suite"], metadata) or case.get("outcome") not in OUTCOMES:
            raise ValueError("case suite/outcome is outside collection")
        if source_ids[case["source_id"]].get("status") == "success" and case["outcome"] in ("fail", "error"):
            raise ValueError("successful source contains failing case verdicts")
        number(case.get("duration_seconds"), "case.duration_seconds")
        reason = case.get("skip_reason")
        if reason is not None:
            required_text(reason, "case.skip_reason")
        if case["outcome"] != "skip" and reason is not None:
            raise ValueError("only skipped cases can have a skip reason")
        history = case.get("attempts")
        if history is not None:
            if not isinstance(history, list) or not history:
                raise ValueError("case attempts must be a nonempty list or null")
            for attempt in history:
                json_object(attempt, "attempt")
                if attempt.get("outcome") not in OUTCOMES:
                    raise ValueError("invalid attempt outcome")
                number(attempt.get("duration_seconds"), "attempt.duration_seconds")
            if history[-1].get("outcome") != case["outcome"] or history[-1].get("skip_reason") != reason:
                raise ValueError("final attempt does not match case verdict")
        if case["outcome"] != "skip":
            executed[case["feature"]] += 1
    if any(not executed[feature] for feature in metadata["features"]):
        raise ValueError("each declared feature must contain executed cases")


def normalize(collection, base_path):
    if collection.get("version") != 1 or type(collection["version"]) is not int:
        raise ValueError("unsupported collection version")
    metadata = validate_metadata(collection.get("metadata"))
    raw_sources = collection.get("sources")
    if not isinstance(raw_sources, list) or not raw_sources:
        raise ValueError("sources must be a nonempty list")
    sources, cases, failures = [], [], []
    ids, paths = set(), set()
    for raw in raw_sources:
        source = dict(json_object(raw, "source"))
        for key in ("id", "path", "feature"):
            required_text(source.get(key), f"source.{key}")
        source.setdefault("role", "verdict")
        if source["role"] not in ("verdict", "diagnostic") or source["feature"] not in metadata["features"]:
            raise ValueError("unknown source role/feature")
        path = (base_path / source["path"]).resolve()
        if source["id"] in ids or (path, source["feature"]) in paths:
            raise ValueError("duplicate source id or report path for a feature")
        ids.add(source["id"])
        paths.add((path, source["feature"]))
        if source.get("sha", metadata["sha"]) != metadata["sha"]:
            raise ValueError("source SHA differs from collection SHA")
        source["revision_bound"] = False
        source_status = terminal_status(source.get("status"))
        if source.get("metadata_path"):
            evidence = read_json(base_path / source["metadata_path"])
            if evidence.get("sha") != metadata["sha"] or evidence.get("branch") != metadata["branch"]:
                raise ValueError("source job metadata does not match collection SHA/branch")
            source["revision_bound"] = True
            evidence_status = terminal_status(evidence.get("status"))
            if source_status is not None and source_status != evidence_status:
                raise ValueError("source status differs from original job metadata")
            source_status = evidence_status
        elif source.get("format") == "circleci-tests":
            evidence = read_json(path)
            if isinstance(evidence, dict) and evidence.get("sha") == metadata["sha"]:
                source["revision_bound"] = True
        source["status"] = source_status
        sources.append(source)
        if source["format"] == "circleci-tests":
            parsed = circleci_cases(path, source, metadata)
        elif source["format"] == "junit":
            parsed = junit_cases(path, source, metadata)
        elif source["format"] == "go-json":
            parsed, package_failures = go_cases(path, source, metadata)
            if source["role"] == "verdict":
                failures.extend({"source_id": source["id"], "suite": suite} for suite in package_failures)
        else:
            raise ValueError(f"unknown source format {source['format']!r}")
        # Aggregate CircleCI Go shards may contain no cases in the selected
        # component. Require explicit shard metadata and valid out-of-scope
        # evidence before accepting that empty share; the collection itself
        # must still have executed cases for every feature.
        if not parsed and source['format'] == 'go-json' and (collection.get('discovery') or {}).get('partition') == 'test':
            # Exact identity/package checks below reject missing nonempty shares.
            pass
        elif not parsed and metadata.get("package_prefix") and "shard_index" in source:
            unfiltered = {**metadata, "package_prefix": None}
            if source["format"] == "go-json":
                outside, _ = go_cases(path, dict(source), unfiltered)
            elif source["format"] == "circleci-tests":
                outside = circleci_cases(path, source, unfiltered)
            else:
                outside = junit_cases(path, source, unfiltered)
            if not outside:
                raise ValueError(f"empty report, including outside selected scope: {source['id']}")
        elif not parsed:
            raise ValueError(f"source contains no cases in selected scope: {source['id']}")
        if source["role"] == "verdict":
            if source["status"] == "success" and any(case["outcome"] in ("fail", "error") for case in parsed):
                raise ValueError("successful source contains failing case verdicts")
            cases.extend(parsed)
    validate_shards(sources, metadata)
    validate_cases(cases, metadata, sources)
    discovery = validate_discovery(collection.get("discovery"))
    if discovery and "packages" in discovery:
        if any(case["suite"] not in discovery["packages"] for case in cases):
            raise ValueError("reported case package is absent from discovery")
    validate_package_coverage(sources, discovery, cases)
    return {"version": 1, "metadata": metadata, "sources": sources,
            "cases": sorted(cases, key=identity), "package_failures": failures,
            "discovery": discovery,
            "measurements": measurements(collection.get("measurements"))}


def validate_normalized(data):
    json_object(data, "normalized input")
    if data.get("version") != 1 or type(data["version"]) is not int:
        raise ValueError("unsupported normalized version")
    metadata = validate_metadata(data.get("metadata"))
    sources = data.get("sources")
    if not isinstance(sources, list) or not sources:
        raise ValueError("normalized sources must be a nonempty list")
    ids = []
    for source in sources:
        json_object(source, "source")
        ids.append(required_text(source.get("id"), "source.id"))
        if source.get("role") not in ("verdict", "diagnostic"):
            raise ValueError("invalid normalized source role")
        if source.get("feature") not in metadata["features"]:
            raise ValueError("invalid normalized source feature")
        if type(source.get("revision_bound")) is not bool:
            raise ValueError("normalized source revision_bound must be boolean")
        if source.get("status") not in (None, "success", "fail", "canceled"):
            raise ValueError("invalid normalized source terminal status")
        for history in source.get("package_attempts", []):
            required_text(history.get("suite"), "package attempt suite")
            outcomes = history.get("outcomes")
            if not isinstance(outcomes, list) or not outcomes or any(outcome not in ("pass", "fail", "skip", "build-fail") for outcome in outcomes):
                raise ValueError("invalid package attempt history")
    if len(ids) != len(set(ids)):
        raise ValueError("duplicate normalized sources")
    validate_shards(sources, metadata)
    validate_cases(data.get("cases"), metadata, sources)
    validate_discovery(data.get("discovery"))
    validate_package_coverage(sources, data.get("discovery"), data['cases'])
    measurements(data.get("measurements"))
    if not isinstance(data.get("package_failures"), list):
        raise ValueError("normalized package_failures must be a list")
    return metadata


def case_label(key):
    return {"feature": key[0], "suite": key[1], "name": key[2]}


def retry_summary(data):
    known = [case for case in data["cases"] if case.get("attempts") is not None]
    retried = [{**case_label(identity(case)), "attempts": case["attempts"]} for case in known if len(case["attempts"]) > 1]
    package_retries = [{"source_id": source["id"], **history} for source in data["sources"] if source["role"] == "verdict"
                       for history in source.get("package_attempts", []) if len(history["outcomes"]) > 1]
    return {"cases_with_attempt_evidence": len(known), "cases_without_attempt_evidence": len(data["cases"]) - len(known),
            "observed_retry_attempts": sum(len(case["attempts"]) - 1 for case in known), "retried_cases": retried,
            "observed_package_retry_attempts": sum(len(history["outcomes"]) - 1 for history in package_retries),
            "retried_packages": package_retries}


def compare(baseline, candidate):
    left_metadata, right_metadata = validate_normalized(baseline), validate_normalized(candidate)
    metadata_differences = [{"field": key, "baseline": left_metadata.get(key), "candidate": right_metadata.get(key)}
                            for key in COMPARABLE if left_metadata.get(key) != right_metadata.get(key)]
    if left_metadata["provider"] == right_metadata["provider"]:
        metadata_differences.append({"field": "provider", "baseline": left_metadata["provider"],
                                     "candidate": right_metadata["provider"], "reason": "expected different providers"})
    report = {"version": 1, "status": "incomparable" if metadata_differences else "equivalent",
              "baseline": left_metadata, "candidate": right_metadata, "metadata_differences": metadata_differences,
              "missing_cases": [], "extra_cases": [], "different_cases": [], "unknown_skip_reasons": [],
              "incomplete_evidence": [],
              "unhealthy_sources": [],
              "retries": {"baseline": retry_summary(baseline), "candidate": retry_summary(candidate)},
              "measurements": {"baseline": measurements(baseline.get("measurements")),
                               "candidate": measurements(candidate.get("measurements"))},
              "package_failures": {"baseline": baseline["package_failures"], "candidate": candidate["package_failures"]},
              "discovery": {"baseline": baseline.get("discovery"), "candidate": candidate.get("discovery")},
              "limitations": ["Case parity compares retained reports, not unreported discovery or execution.",
                              "Wall time, summed task time, CPU time and billed time are separate observations; unknown values are null."]}
    if metadata_differences:
        return report
    for side, data in (("baseline", baseline), ("candidate", candidate)):
        if data["metadata"].get("routing_context") is None:
            report["incomplete_evidence"].append({"side": side, "reason": "verified shared routing context unavailable"})
        discovery = data.get("discovery")
        if discovery is None or discovery.get("complete") is not True:
            report["incomplete_evidence"].append({"side": side, "reason": "authoritative complete discovery unavailable"})
        for source in data["sources"]:
            if source["role"] == "verdict" and not source["revision_bound"]:
                report["incomplete_evidence"].append({"side": side, "source_id": source["id"], "reason": "report lacks revision-binding evidence"})
            if source["role"] == "verdict" and source.get("status") is None:
                report["incomplete_evidence"].append({"side": side, "source_id": source["id"], "reason": "terminal job/task outcome unavailable"})
            elif source["role"] == "verdict" and source["status"] != "success":
                report["unhealthy_sources"].append({"side": side, "source_id": source["id"], "status": source["status"]})
    left, right = {identity(case): case for case in baseline["cases"]}, {identity(case): case for case in candidate["cases"]}
    report["missing_cases"] = [case_label(key) for key in sorted(left.keys() - right.keys())]
    report["extra_cases"] = [case_label(key) for key in sorted(right.keys() - left.keys())]
    for key in sorted(left.keys() & right.keys()):
        before, after = left[key], right[key]
        if before["outcome"] != after["outcome"] or before["skip_reason"] != after["skip_reason"]:
            report["different_cases"].append({**case_label(key), "baseline": {k: before[k] for k in ("outcome", "skip_reason")},
                                              "candidate": {k: after[k] for k in ("outcome", "skip_reason")}})
        if before["outcome"] == "skip" or after["outcome"] == "skip":
            if before["skip_reason"] is None or after["skip_reason"] is None:
                report["unknown_skip_reasons"].append(case_label(key))
    report["counts"] = {"baseline": dict(Counter(case["outcome"] for case in left.values())),
                        "candidate": dict(Counter(case["outcome"] for case in right.values()))}
    if any(report[key] for key in ("missing_cases", "extra_cases", "different_cases")):
        report["status"] = "different"
    elif report["unknown_skip_reasons"] or report["incomplete_evidence"]:
        report["status"] = "incomplete"
    if baseline["package_failures"] or candidate["package_failures"] or report["unhealthy_sources"]:
        report["status"] = "different"
    if baseline.get("discovery") is not None and candidate.get("discovery") is not None:
        # Compare discovered sets, not provider-specific shard packing.
        for field in ("packages", "test_files", "tests"):
            if baseline["discovery"].get(field) != candidate["discovery"].get(field):
                report.setdefault("discovery_differences", []).append(field)
                report["status"] = "different"
    return report


def markdown(report):
    def safe(value):
        return str(value).replace("`", "'").replace("\n", " ")
    lines = [f"CI comparison: **{report['status']}**", "",
             f"Revision: `{safe(report['baseline']['sha'])}`", "",
             f"Workload: {safe(report['baseline']['workload'])}; profile: {safe(report['baseline']['profile'])}.", "",
             f"Missing cases: {len(report['missing_cases'])}; extra cases: {len(report['extra_cases'])}; "
             f"different outcomes/reasons: {len(report['different_cases'])}.", ""]
    if report["metadata_differences"]:
        lines.append("Incomparable metadata: " + ", ".join(safe(item["field"]) for item in report["metadata_differences"]) + ".")
    for name, side in report["retries"].items():
        lines.append(f"{name}: {side['observed_retry_attempts']} observed retry attempts; "
                     f"{side['cases_without_attempt_evidence']} cases without attempt evidence.")
    if report["unknown_skip_reasons"]:
        lines.append(f"Skip reasons unavailable for {len(report['unknown_skip_reasons'])} matching cases.")
    if report["incomplete_evidence"]:
        lines.append(f"Incomplete provenance/discovery evidence: {len(report['incomplete_evidence'])} observations.")
    if report["unhealthy_sources"]:
        lines.append(f"Failed/canceled verdict sources: {len(report['unhealthy_sources'])}.")
    lines.extend(["", "Durations, resources, caches and cost are retained as separate observations in the JSON report.", ""])
    lines.extend(report["limitations"])
    return "\n".join(lines) + "\n"


def write_json(path, data):
    path.write_text(json.dumps(data, indent=2, sort_keys=True, allow_nan=False) + "\n", encoding="utf-8")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    normalize_parser = commands.add_parser("normalize", help="convert local retained reports")
    normalize_parser.add_argument("--input", type=Path, required=True)
    normalize_parser.add_argument("--output", type=Path, required=True)
    compare_parser = commands.add_parser("compare", help="compare two normalized providers")
    compare_parser.add_argument("--baseline", type=Path, required=True)
    compare_parser.add_argument("--candidate", type=Path, required=True)
    compare_parser.add_argument("--output", type=Path, required=True)
    compare_parser.add_argument("--markdown", type=Path)
    args = parser.parse_args()
    try:
        if args.command == "normalize":
            write_json(args.output, normalize(read_json(args.input), args.input.resolve().parent))
            return 0
        report = compare(read_json(args.baseline), read_json(args.candidate))
        write_json(args.output, report)
        if args.markdown:
            args.markdown.write_text(markdown(report), encoding="utf-8")
        print(f"CI comparison: {report['status']}")
        return 0 if report["status"] == "equivalent" else 2 if report["status"] == "incomparable" else 1
    except (KeyError, TypeError, ValueError, OSError, ET.ParseError) as error:
        print(f"CI comparison rejected input: {error}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
