"""Decode original Go JSON and JUnit reports without provider parity policy.

Callers supply the package scope. Decoding retains retries, skips and package
failures; callers decide whether coverage and verdict histories are acceptable.
"""
import json
import math
from pathlib import Path
import re
import xml.etree.ElementTree as ET

OUTCOMES = {"pass", "fail", "error", "skip"}


RESULTS = {
    "success": "pass", "passed": "pass", "pass": "pass", "system-out": "pass",
    "failure": "fail", "failed": "fail", "fail": "fail",
    "error": "error", "skipped": "skip", "skip": "skip",
}


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
