# Rust E2E shadow parity

All nine `rust-e2e-ci` inventory occurrences are verified at
`7d2f568b693f3b86b5b5178ed2730cd3d8c6a2f3` in draft PR #23151.
[RWX run 74e65d49](https://cloud.rwx.com/optimism/runs/74e65d4904e9468abdd8d3339186077e)
and [Circle pipeline 135555](https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/135555)
executed the complete workload successfully. Circle retains the required gates.

| Workload | Complete observed selection | Results |
| --- | --- | --- |
| Workspace release | 76 packages, 88 units (83 workspace, five host build-dependency), 16 binaries | Complete on both providers |
| Contracts | `ci`, `--skip test`, script preparation and embedded deployer archive | All runtime inputs agree |
| Cannon prestates | Both configured variants, eight output files | Every file byte-identical |
| Kona proof | 37 top-level tests, eight shards, 368 case identities | 365 passes, three skips |
| Node restart | Three top-level tests, three shards | Three passes |
| Simple Kona | 15 top-level tests, one shard | 14 passes, one skip |
| Simple Kona sequencer | 15 top-level tests, one shard | 14 passes, one skip |
| op-reth sysgo | Nine top-level tests, one shard | Nine passes |
| Aggregate | All producers, compilers and fourteen verdicts | Passed |

The comparison verifies 671 original Circle files and 699 original RWX files,
complete selection, exactly-once initial assignments, settings, original JSON,
JUnit and retry histories. All 410 identities agree: 405 passes, five skips and
zero retries. No manifest-declared empty logs had to be restored.

The three proof skip reasons match verbatim. The two `TestL2FinalizedSync` skips
include logger timestamps. The comparator accepts only that exact common-package
test, after checking the benchmark source's unconditional skip, the complete
severity/message/scope sequence and valid date/time fields. Both original reasons
are retained. Changed messages, severity, scope, source and unrelated skips fail
fixtures. This is a resolved timestamp difference, not blanket log normalization.

Proof tests keep their explicit `-parallel=8`; node/reth tests preserve Go's
default. Both providers observed effective parallelism eight and `GOMAXPROCS=8`.
`runtime.NumCPU` reports 32 on Circle and eight on RWX; that host-visibility
difference is retained in the evidence. Original package working directories,
source, fixtures, Git metadata, pinned tools and runtime Go/Forge builds remain
available. Compilation uses 16 CPU / 32 GiB; fresh verdicts use 8 CPU / 16 GiB.
Reports and verdict results are excluded from reusable compiler outputs.

## Artifact and failure evidence

All 1,079 contract artifacts agree in ABI, runtime/deployment bytecode and other
runtime fields. Eight compiler source graphs contain the same files, and 2,688
embedded source files/links agree. Outer archive bytes differ: only workspace-path
metadata and compiler-generated source/node/type identifiers differ after full
structural comparison; source-span offsets and lengths agree. Only the compiler identifier fields may differ; this comparison does not claim
a globally consistent identifier mapping across cached compilation units. RWX embeds an extra `.gitcommit` containing the exact benchmark SHA
for its fallback; that difference is retained. The deployer archive's complete
compile outputs are checked against the producer's full file hashes.

Both prestate hashes match Circle:
`0x03e384aad91052e86a9912ad763cd32027586d99b5e8e2024115c606f35b6afa`
and `0x03d56f7fd7d39b381efc142e127f41109709812d788c436ebad8f6de0633bc39`.
All binary, metadata and proof file hashes are retained.

[First failures](rwx-rust-e2e-evidence/first-failures.json) retain the JUnit option
parser defect, the validator's incorrect rejection of valid dual-role Cargo units,
and the reversed working-directory arguments that selected the root contract
build for op-reth. RWX explicitly reported memory exhaustion on that wrong
workload. The corrected adapter executes the nested fixture recipe and passed
at the original 8 CPU / 16 GiB allocation. A real Forge/Go fixture verifies it.

[Negative probes](rwx-rust-e2e-evidence/negative-probes.json) retain actual fresh
failure and process-group cancellation reports, including original partial
JSON and exit 143 completeness rejection. A separate full-release probe verifies
the actual aggregate rejects failed/canceled verdicts. These were CLI rehearsals
against isolated fixture commits, without GitHub status publishing; they do not
establish RWX-engine abort behavior or production credential availability.

## Cache and timing observations

The benchmark reused 71/88 Cargo units and recorded 13 Rust sccache hits.
Release execution took 245 seconds. A subsequent
[compile-only warm run 675d1256](https://cloud.rwx.com/optimism/runs/675d125651e04cf2a05d48b061bf0960)
reused 69/88 units and recorded 14 Rust sccache hits, with the same source
fingerprint and 1,343 restored source files. Its contract and prestate producers
completed in 11 and two seconds. It selected only preparation/producer/compiler
targets: zero suite verdicts, helper tests or gates ran. The actual protected
`develop` cache-rebuild event remains to be observed after merge.

For this single benchmark, RWX run start through its final aggregate was 734.150
seconds. Circle pipeline creation through its final E2E gate was 1,231.590 seconds
(workflow creation boundary: 1,180.791 seconds). Task timing fields and the
provider's separately defined runtime counters are retained without substituting
them for elapsed timestamps. Circle rebuilt all 88 workspace/host units; its
shared release report did not retain sccache statistics, so dependency/compiler
cache warmth is unknown. These are single observations with different cache
states, not a warm median or a demonstrated speed win.

At the benchmark SHA, all four required Circle gates, dependency review and all
existing optional RWX shadows reached successful terminal states. GitHub reported
144 successes and one neutral check. The PR stayed draft and mergeable.

## Reproduction and remaining limits

[Parity index](rwx-rust-e2e-evidence/parity.json) retains all original report hashes,
selections, commands, settings, task/job IDs and terminal gates. Larger embedded
file inventories are bound by canonical JSON digests. The complete 2.1 MB
comparison and complete provider archives remain under
`.ci/rwx-e2e-stage-evidence/7d2/` and immutable provider artifacts. Reproduce with:

```sh
python3 ops/ci/compare-rust-e2e.py \
  --root .ci/rwx-e2e-stage-evidence/7d2 \
  --sha 7d2f568b693f3b86b5b5178ed2730cd3d8c6a2f3 \
  --output .ci/rwx-e2e-stage-evidence/7d2/comparison.json
```

[Cache/warming index](rwx-rust-e2e-evidence/cache-and-warming.json) retains complete
release coverage, original statistics and compile-only task evidence. Broader
cache invalidation, engine cancellation, privileged/fork routing, protected warming
and required-gate ownership still have separate todos. This stage adds nine
verified jobs, taking overall inventory coverage to **47/86 (55%)**.

At `6245472e`, every optional native shadow and three required Circle gates
passed, but Circle job 5635823 failed `TestResyncing`. The restarted validator
fatally exited after `UnexpectedStaticFileTxNumber(Transactions, 0, 1)`; the
test then hit its five-minute resync deadline. Rust and sysgo sources are
unchanged from the preceding fully green `fd426d87` head. The
[complete original failure index](rwx-rust-e2e-evidence/6245-circle-sysgo-failure.json)
retains every report seal and terminal check counts. Diagnosis remains open;
no retry or test relaxation has been added. Earlier verified same-SHA parity
remains bound to its documented revisions, and this head is not fully green.
