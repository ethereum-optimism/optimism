#!/usr/bin/env python3
"""CI-only, bounded SMTChecker feasibility experiment on instrumented production source."""

import argparse
import ast
import gzip
import hashlib
import json
import os
from pathlib import Path
import posixpath
import re
import subprocess
import time

PORTAL = "src/L1/OptimismPortal2.sol"
GUARD = "        checkWithdrawal(withdrawalHash, _proofSubmitter);"
POINT = "        finalizedWithdrawals[withdrawalHash] = true;"
PROPERTY = "!finalizedWithdrawals[withdrawalHash] && provenWithdrawals[withdrawalHash][_proofSubmitter].timestamp != 0"
VARIANTS = ("guards", "without-check", "wrong-submitter", "abstract-reachability")


def replace_once(source, old, new):
    assert source.count(old) == 1, f"Production source anchor changed: {old}"
    return source.replace(old, new, 1)


def instrument(original, variant):
    # The nondeterministic internal summary overapproximates all starting storage. It does
    # not assert valid proofs or rely on the implementation constructor as proxy setup.
    helper = "\n    /// @custom:smtchecker abstract-function-nondet\n    function verificationHavoc() internal {}\n"
    source = replace_once(original, GUARD, GUARD)
    edits = []

    def edit(old, new):
        nonlocal source
        source = replace_once(source, old, new)
        edits.append((new, old))

    edit("\n}", helper + "\n}")
    edit("        // Cannot finalize withdrawal transactions while the system is paused.",
         "        verificationHavoc();\n        // Cannot finalize withdrawal transactions while the system is paused.")
    # Unrelated proving/trie traversal is outside this guard experiment. Its abstract
    # summary admits arbitrary state, and no provenance result is claimed.
    edit("    function proveWithdrawalTransaction(",
         "    /// @custom:smtchecker abstract-function-nondet\n    function proveWithdrawalTransaction(")
    if variant == "without-check":
        edit(GUARD, "        // Diagnostic: omitted eligibility check.")
    if variant == "wrong-submitter":
        edit(GUARD, "        checkWithdrawal(withdrawalHash, address(0));")
    assertion = "assert(" + ("false" if variant == "abstract-reachability" else PROPERTY) + ");"
    edit(POINT, "        " + assertion + "\n" + POINT)
    restored = source
    for new, old in reversed(edits):
        restored = replace_once(restored, new, old)
    assert restored == original, "Verification instrumentation changed production text"
    return source, len(source[:source.index(assertion)].encode())


def inputs(variant):
    remappings = ast.literal_eval(re.search(r"remappings\s*=\s*(\[.*?\])", Path("foundry.toml").read_text(), re.S)[1])
    remappings = [r.split("=", 1)[0] + "=" + r.split("=", 1)[1].rstrip("/") + "/" for r in remappings]
    sources, pending = {}, [PORTAL]
    while pending:
        unit = pending.pop()
        if unit in sources:
            continue
        content = Path(unit).read_text()
        sources[unit] = {"content": content}
        for statement in re.findall(r"\bimport\s+[^;]+;", content):
            name = re.search(r"[\"']([^\"']+)[\"']", statement)[1]
            if name.startswith("."):
                name = posixpath.normpath(posixpath.join(posixpath.dirname(unit), name))
            else:
                for mapping in sorted(remappings, key=len, reverse=True):
                    prefix, target = mapping.split("=", 1)
                    if name.startswith(prefix):
                        name = target + name[len(prefix):]
                        break
            pending.append(name)
    original = sources[PORTAL]["content"]
    source, offset = instrument(original, variant)
    sources[PORTAL]["content"] = source
    return {"language": "Solidity", "sources": sources, "settings": {
        "remappings": remappings, "outputSelection": {"*": {"": ["ast"]}},
        "modelChecker": {"engine": "chc", "solvers": ["smtlib2"], "targets": ["assert"],
                         "contracts": {PORTAL: ["OptimismPortal2"]}, "showUnproved": True, "timeout": 120000},
    }}, offset, hashlib.sha256(original.encode()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--solc", required=True)
    parser.add_argument("--image")
    parser.add_argument("--compile-only", action="store_true")
    args = parser.parse_args()
    assert args.compile_only or os.environ.get("CI") == "true", "Proof execution is CI-only"
    root = Path("test/kontrol/logs/finalizer-smt")
    root.mkdir(parents=True, exist_ok=True)
    deadline = time.monotonic() + 1800

    def run(command, data=None, timeout=180):
        remaining = min(timeout, deadline - time.monotonic())
        assert remaining > 0, "Experiment exhausted its 30-minute budget"
        return subprocess.run(command, input=data, text=True, capture_output=True, check=True, timeout=remaining)

    version = run([args.solc, "--version"]).stdout
    assert "0.8.15+commit.e14f2714" in version, version
    (root / "solc-version.txt").write_text(version)
    if not args.compile_only:
        assert args.image
        (root / "z3-version.txt").write_text(run(["docker", "run", "--rm", "--entrypoint", "z3", args.image, "--version"]).stdout)
    verdicts = []
    for variant in VARIANTS:
        start = time.monotonic()
        print(variant + ": starting", flush=True)
        directory = root / variant
        directory.mkdir(exist_ok=True)
        request, offset, source_hash = inputs(variant)
        if args.compile_only:
            request["settings"]["modelChecker"] = {"engine": "none"}

        def retain(name, text):
            # Compress evidence, never normalize or rewrite the solver's input.
            (directory / (name + ".gz")).write_bytes(gzip.compress(text.encode(), mtime=0))

        def compile_request(name):
            raw = json.dumps(request)
            retain(name + ".input.json", raw)
            result = run([args.solc, "--standard-json"], raw)
            retain(name + ".output.json", result.stdout)
            (directory / (name + ".stderr.txt")).write_text(result.stderr)
            output = json.loads(result.stdout)
            errors = [e for e in output.get("errors", []) if e["severity"] == "error"]
            assert not errors, json.dumps(errors)
            return output

        first = compile_request("export")
        if args.compile_only:
            print(variant + ": COMPILE_ONLY (SMT disabled)", flush=True)
            continue

        def target_messages(output):
            return [e["message"] for e in output.get("errors", [])
                    if e.get("errorCode") == "6328" and e.get("sourceLocation", {}).get("file") == PORTAL
                    and e["sourceLocation"]["start"] == offset]

        assert target_messages(first) == ["CHC: Assertion violation might happen here."], "Target was not exported as unresolved"
        queries = first.get("auxiliaryInputRequested", {}).get("smtlib2queries", {})
        assert queries, "No actual solver queries exported"
        responses = {}
        # Solc 0.8.15 can reorder conjunctions between compiler processes. A new
        # query hash requires a new actual solve, never rekeying an old response.
        for round_number in range(1, 9):
            for key, query in queries.items():
                assert re.fullmatch(r"0x[0-9a-fA-F]{64}", key)
                assert key not in responses, "Compiler did not consume a supplied response"
                retain(key + ".smt2", query)
                name = "withdrawal-smt-" + key[2:18]
                print(variant + ": solving " + key, flush=True)
                try:
                    solved = run(["docker", "run", "--rm", "--name", name, "-i", "--entrypoint", "z3", args.image, "-in"], query, 150)
                finally:
                    subprocess.run(["docker", "rm", "-f", name], capture_output=True, timeout=30)
                (directory / (key + ".response.txt")).write_text(solved.stdout)
                (directory / (key + ".stderr.txt")).write_text(solved.stderr)
                print(variant + ": solver returned " + solved.stdout.strip(), flush=True)
                assert solved.stdout.strip() in ("sat", "unsat"), "Unknown or malformed solver response"
                responses[key] = solved.stdout
            request["auxiliaryInput"] = {"smtlib2responses": responses}
            final = compile_request("checked-" + str(round_number))
            queries = final.get("auxiliaryInputRequested", {}).get("smtlib2queries", {})
            if not queries:
                break
        assert not queries, "Unanswered solver query after eight response rounds"
        messages = target_messages(final)
        expected = [] if variant == "guards" else ["CHC: Assertion violation happens here."]
        verdict = {"variant": variant, "matchedExpected": messages == expected,
                   "targetOffset": offset, "targetMessages": messages, "queryCount": len(responses),
                   "sourceSHA256": source_hash, "seconds": time.monotonic() - start,
                   "fullSecurityMilestone": False}
        verdicts.append(verdict)
        (root / "results.json").write_text(json.dumps(verdicts, indent=2) + "\n")
        print(json.dumps(verdict), flush=True)
    assert args.compile_only or all(v["matchedExpected"] for v in verdicts), "Feasibility requirements not met"


if __name__ == "__main__":
    main()
