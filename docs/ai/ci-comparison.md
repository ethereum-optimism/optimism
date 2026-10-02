# Compare CircleCI and RWX shadow reports

`ops/ci/compare-ci.py` compares retained evidence for one workload on one commit.
It runs locally without network access or credentials. Keep raw downloads outside
Git (for example, under `.ci/comparison/`); retain provider run/job URLs alongside
the collection so another operator can audit it.

## Collect one revision

Open a PR for the comparison branch: CircleCI currently has `build-prs-only`
enabled. Wait for both providers to finish, and verify their full commit SHA and
branch before downloading reports. A previous commit with the same source tree
is a historical baseline, not a same-revision comparison.

For Go rollup, download every aggregate CircleCI Go JSON shard and both RWX Go
JSON shards. Select `github.com/ethereum-optimism/optimism/op-node/rollup` with
`metadata.package_prefix`. Retain RWX's package manifest and the CircleCI package
selection evidence, including packages without test files. Go JSON preserves
attempt history; CircleCI's final test API or JUnit cannot establish zero retries.

For standard contracts, collect CircleCI's complete test API response and RWX's
original JUnit for each of `main`, `CUSTOM_GAS_TOKEN`, `OPTIMISM_PORTAL_INTEROP`
and `ZK_DISPUTE_GAME`. Mark trace/rerun reports as `diagnostic`: a successful
rerun must not replace the first verdict. Retain the test-file inventory and
configuration evidence. Distinguish runtime configuration dumps from settings
that are only declared in source. The standard branch profile is `liteci`;
`develop` uses `ci`. Neither profile covers the other contract jobs.

Do not collect secret values. Fetch every page of test APIs and every expected
shard. Record the original job/task terminal outcome even if its test cases all
passed: later build or convention-check failures still matter.

## Describe the evidence

A version-1 collection lists local report sources. Paths are relative to the
collection JSON. This abbreviated Go example is illustrative; replace all
metadata and paths with verified evidence:

```json
{
  "version": 1,
  "metadata": {
    "provider": "rwx",
    "sha": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "branch": "codex/comparison-example",
    "workload": "go-rollup",
    "profile": "ci",
    "features": ["main"],
    "package_prefix": "github.com/ethereum-optimism/optimism/op-node/rollup",
    "routing_context": {
      "kind": "branch",
      "base_branch": "develop",
      "run_main": true,
      "run_contracts_feature_tests": true
    },
    "trigger": {"type": "github.push", "evidence": "provider-run.json"},
    "test_config": {"tags": ["ci"], "short": false, "timeout": "40m"}
  },
  "sources": [
    {
      "id": "rollup-0",
      "format": "go-json",
      "path": "shard-0/log.json",
      "metadata_path": "shard-0/job-metadata.json",
      "feature": "main",
      "role": "verdict",
      "shard_index": 0,
      "shard_total": 1
    }
  ],
  "discovery": {
    "packages": ["github.com/ethereum-optimism/optimism/op-node/rollup"],
    "complete": true,
    "provenance": "package-discovery.json"
  },
  "measurements": {"wall_seconds": null, "billed_seconds": null, "cost": null}
}
```

Use `circleci-tests`, `go-json`, or `junit` for each source's `format`. CircleCI
case files accept `tests`, `items`, or `cases` lists; a nonempty pagination token
is rejected. A source's `metadata_path` points to a retained JSON object with
`sha`, `branch`, and `status` (`success`, `fail`, or `canceled`). Extract those
values from the provider response and retain the original response for audit.
CircleCI case envelopes containing `sha` can bind that revision directly, with
an explicit source `status`. The tool checks consistency with these files; it
does not authenticate the collector or attest a provider response.

Populate `routing_context` from verified shared routing output, using the same
field names and values for both providers. Preserve raw trigger provenance in
`trigger`; an RWX push and a CircleCI PR webhook may select the same context.
Do not infer a context from a trigger name. Include effective test settings in
`test_config`, with configuration provenance alongside the collection. Metadata
mismatches make the comparison incomparable.

For Go discovery, use `packages` and optionally `total`/`shards` from the package
manifest. For contracts, use `test_files`. Set `complete: true` only when retained
selection evidence proves the full workload was discovered; it requires a
nonempty `provenance` reference. Matching case reports alone do not prove this.

## Normalize and compare

```bash
python3 ops/ci/compare-ci.py normalize --input .ci/comparison/circleci.json --output .ci/comparison/circleci.normalized.json
python3 ops/ci/compare-ci.py normalize --input .ci/comparison/rwx.json --output .ci/comparison/rwx.normalized.json
python3 ops/ci/compare-ci.py compare --baseline .ci/comparison/circleci.normalized.json --candidate .ci/comparison/rwx.normalized.json --output .ci/comparison/report.json --markdown .ci/comparison/report.md
python3 ops/ci/test_compare_ci.py
```

The JSON report includes missing/extra identities, outcome and skip-reason
changes, observed retries, evidence gaps and retained measurements. `equivalent`
means the supplied workload evidence meets the comparison checks; it does not
establish aggregate gate coverage or authorize changing required checks.
`different` identifies a mismatch or unhealthy source. `incomplete` retains
case comparisons but lacks required discovery, revision, outcome, routing or
skip-reason evidence. `incomparable` identifies incompatible metadata. Exit
codes are 0 for equivalent, 1 for different/incomplete, and 2 for incomparable
or rejected input.

Equivalence compares final case verdicts; it can coexist with observed retries.
Review retry history separately before judging reliability. Unavailable attempt
history is reported as unknown rather than as zero retries.

For the pilot, the primary metric is push-to-final-verdict wall time for the same
selected workload and coverage. Include queueing, setup and transfers; label
run-start timings when earlier timestamps are unavailable. Report setup, longest
shard, summed task time, CPU time, cache state and actual resources separately with
explicit units and scope. Runner sizes may differ when optimizing wall time.
Cost and billed-usage analysis are deferred; leave those values null until needed.
RWX cached tasks can retain historical execution fields; exclude them from current
compute totals. An aggregate
CircleCI Go job's duration includes other packages and dependency builds, so it
cannot measure only the rollup slice. Separate warm-cache observations from cold
runs, and compare multiple samples before drawing performance conclusions.
