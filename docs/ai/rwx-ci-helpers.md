# RWX report helper ownership

Runtime runners do not import offline comparison scripts. Shared mechanics live
in two modules under `ops/ci`:

- `ci-report.py` reads and writes report JSON, hashes files, verifies manifest
  bytes, and captures one subprocess invocation. It owns signal forwarding and
  preservation of the original process exit code.
- `ci-test-results.py` decodes original Go JSON and JUnit. It retains case and
  package failures, skips and retry histories. It does not compare providers or
  decide whether a suite has complete coverage.

The modules use only the Python standard library. Existing file-based loaders
work both when a script runs directly and when a test loads it from another
working directory.

## Caller contracts

The suite chooses its package or case selection, command, working directory,
required report files, source/settings binding and verdict rules. Contract
upgrades and Rust E2E each check their own final status before passing the
original file manifest to the shared byte verifier. Freshness, cache provenance,
diagnostic reruns and retry ceilings remain in their existing suite owners.

The byte verifier rejects missing required entries, unsafe paths, invalid hashes
and corrupt bytes. Runtime callers do not authorize reconstruction of missing
files. Offline Circle collectors may explicitly supply an audit list to recover
only a manifest-declared empty file. Recovery records the report, path and hash.

The subprocess helper executes once and forwards cancellation to the process
group. The stage records the original negative signal status and returns its
shell exit code. Rust keeps its combined log and optional separate stdout.
Contracts keep separate stdout/stderr and their existing redactor. Contract
output is redacted before it is saved, echoed or hashed. The helper does not
retry or turn diagnostic output into a passing verdict.

Input seals and isolated runtime fixtures include the shared modules wherever
they supply execution behavior. Bootstrap tool inputs remain unchanged because
these modules are consumed from the source checkout, not tool installation.

## Regression coverage

`ci_test_fixtures.py` consolidates report writing, sealing and mutation for the
three contract comparison fixtures. A shared working-directory context also
restores the test process after the NUT CLI runner changes directories. This
prevents a deleted temporary checkout from affecting later fixture groups.
Each fixture still specifies its own
selection, profiles, commands, compiler signatures and original verdicts.

The coordinator runs the shared helper and existing report decoder regressions.
Tests import runtime runners from a directory with no comparer files, reject
corrupt and missing originals, require explicit empty-file recovery, and drain
stderr larger than a pipe buffer while preserving separate JSON stdout. Existing
suite tests retain real subprocess cancellation, stale compiler/cache rejection,
original failures, diagnostic reruns and Go retry-limit coverage.

Local verification on October 5, 2026 passed all 554 helper tests (512 passed,
42 opt-in/tool tests skipped). The final comparer/helper batch passed 57 tests;
RWX lint passed for the changed coordinator. Hosted status belongs to the PR's
exact source revision and is recorded separately from these local results.
