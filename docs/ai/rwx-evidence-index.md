# RWX evidence retrieval and comparison

PR #23151 covers all 86 baseline CircleCI PR job occurrences. Its generated
reports are archived outside Git. The three maintained documents are this guide, the [operator guide](rwx-migration.md)
and [86-job checklist](rwx-parity-todos.md). The small
[summary checksum index](rwx-evidence-summaries.sha256) remains integrity data.
Historical human-written closeouts and experiments are archived below.

## Archive identity and custody

The archive captures source revision
`a1aa49aaf3713a8172f3f615f094488fd8e39c3d` on October 5, 2026.
It contains 216,558 report paths, including complete provider originals,
preparation and test logs, selections, settings, retries, failure fixtures,
provider run/check observations, and the 92 previously committed summaries.
Deduplication and Zstandard compression reduce 38,238,271,163 logical bytes
to 1,347,568,690 bytes of report objects. The manifest and source bundle bring
the complete archive to about 2.4 GB.

Two private copies are retained:

- Operator workstation:
  `/Users/edward/Workspace/op/rwx-ci-pilot-evidence/2026-10-05-a1aa49aa/`
- Existing CI validation host, accessed through the operator's `hetzner` SSH
  alias: `/home/admin/.local/share/optimism-ci-evidence/2026-10-05-a1aa49aa/`

Both archive directories are private to their owner. They are not public
downloads or managed artifact storage. An operator with access to either copy
must supply the archive to a reviewer. Keep both copies until the team assigns
an evidence storage location and retention policy.

| File | SHA-256 |
| --- | --- |
| `manifest.json` | `450aaa5e07daa581c258feb3e94b713215d382f559cfd66186f8fa3efa09b8b3` |
| `source.bundle` | `9a548a480c0b2c8636d1074f3e2ed3a5d8ba2853353be83f23f8cc2919a87c3a` |
| `restore.py` | `bd419f52c0f7d5a9e707ac4d984334e769f1c386d591c1658e079e7eeb84e320` |

The manifest binds each report path, byte count, mode, original SHA-256 and
compressed object parts. It retains original collection seals and labels the
historical exceptions below. `SHA256SUMS` in the archive covers every stored
file. `verification-record.json` records restoration and comparison results.
The self-contained Git bundle preserves the pilot branch's source history
through the captured revision. No archive, bundle or compressed object is added
to the repository.

## Retrieve and restore

Copy the whole archive directory, including `objects/`, outside a checkout.
For the existing private host:

```bash
rsync -a hetzner:/home/admin/.local/share/optimism-ci-evidence/2026-10-05-a1aa49aa/ ./rwx-evidence-archive/
```

Use Python 3.11 or newer and the `zstd` command. Check the three identity hashes
against the table above before running the archived helper. From the archive
directory, verify all stored files and restore into a new directory:

```bash
sha256sum -c SHA256SUMS
python3 restore.py \
  --manifest-sha256 450aaa5e07daa581c258feb3e94b713215d382f559cfd66186f8fa3efa09b8b3 \
  --destination ../rwx-evidence-restored
```

On macOS, use `shasum -a 256 -c SHA256SUMS`. Omit `--destination` to verify
every original without retaining decompressed files. Add repeatable `--prefix`
arguments to restore selected families, for example
`--prefix .ci/rwx-l2-evidence/f821/ --prefix captures/`.
The helper rejects a changed manifest, missing or corrupt compressed parts,
changed original bytes and an existing destination. Identical restored files
can share hard links; treat the restored originals as read-only.

Use `git clone source.bundle ../rwx-pilot-source` to recover the recorded source
and run the comparers at that archived revision (historically `ops/ci/compare-*.py`; currently `ops/ci/migration/compare-*.py`) against restored `.ci/` reports.
Provider run/check captures reside under `captures/` or alongside their report
families. They retain original source revisions and run identities.

Archived workload closeouts link historical summaries at the immutable
[captured revision](https://github.com/ethereum-optimism/optimism/tree/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai).
Their exact bytes are also recoverable from the archive and checked by the
92-line summary checksum file. Deleting them from the PR's final tree does not
remove commits already pushed to the pilot branch. A future squash merge can
land the reviewed final tree without importing those earlier report blobs.

## Verified coverage and historical exceptions

Archive validation restored every report path on the workstation and independently
verified every original on the private CI host. All 23,141 unique objects passed
their compressed and decompressed hashes. The existing L2 fork, Contracts gate,
selector and flaky-report comparers passed against restored originals. Empty-file
restoration passed; changed manifests, missing/corrupt parts and an existing
restore destination were rejected. All 92 summary hashes and 97 historical report
links were verified before deleting the generated files from the final tree.
The source bundle passed a clean clone to its recorded revision and branch,
followed by `git fsck`. An initial bundle inherited shallow history and failed
that clone rehearsal; its original and failure log remain under `validation/`.

The authoritative last three workload comparisons use
`f821983dd56cbd7e488ab903d1ac386a330de7f6`: complete L2 fork reports, the 21
Contracts prerequisites, and selector publication/readback. Final readiness at
`a1aa49aa` passed all four required Circle gates, dependency review and all 23
optional RWX checks. These are historical observations; a later PR head needs
its own terminal checks. Earlier workload revisions and complete failure
evidence remain in the archive.

The archive labels these older collection limitations explicitly:

- 113 unavailable files belong only to the incomplete preliminary selector
  collection `.ci/rwx-selector-evidence/86f`. They are not part of the passing
  `6384` or `f821` selector comparisons.
- 4,368 missing local paths were recovered from other collected copies with
  the exact SHA-256 required by their original seals.
- 188 files declared empty by original manifests are retained as empty files.
- Ten regenerated derived files under `.ci/rwx-rust-stage-evidence/6a/derived`
  differ from their older seals. The observed bytes and original expected
  hashes are both recorded. The authoritative `68ad` Rust originals are intact.

These exceptions receive no new parity credit. The archive preserves them for
audit rather than treating every historical collection as complete.

## Documentation and experiment archive

The October 6 cleanup captures the pre-cleanup source
`57094f2aeece2b08ec7c2cba8ae19609b2859ae3`. Its private archive contains all
28 earlier RWX/comparison Markdown files, three retired package-probe definitions,
pre-cleanup Circle replay/routing files, exact non-CI and Circle audit diffs, and
the successful pre-cleanup check snapshot. These are historical instructions;
use the maintained guide for current operations. The audit diffs compare against
the PR merge base `c8e4ba855d79ca56463909ef5a2c5830a1189401`; the October 5
upstream observation is recorded separately in `index.json`.

Two owner-private copies retain 40 hashed files (about 6.3 MB):

- `/Users/edward/Workspace/op/rwx-ci-pilot-evidence/2026-10-06-maintenance-57094f2aee/`
- `hetzner:/home/admin/.local/share/optimism-ci-evidence/2026-10-06-maintenance-57094f2aee/`

`SHA256SUMS` has SHA-256
`d46f4a3b14c0cef29a5e68b71c8edcfdcd931b463d6ec3de0db3e355fe1baea9`.
Both copies were verified before removing the old documentation and machinery.
Files under `source/` preserve repository-relative paths. Retrieval requires no
archive script or decompressor:

```bash
rsync -a hetzner:/home/admin/.local/share/optimism-ci-evidence/2026-10-06-maintenance-57094f2aee/ ./rwx-maintenance-history/
cd rwx-maintenance-history
shasum -a 256 SHA256SUMS  # compare with the independently recorded value above
shasum -a 256 -c SHA256SUMS
# Historical package probe:
# source/.rwx/local-package-fixture.yml and source/.rwx/packages/fixture-*.yml
```

The [pre-cleanup source](https://github.com/ethereum-optimism/optimism/tree/57094f2aeece2b08ec7c2cba8ae19609b2859ae3)
also preserves its tracked history. Keep archive data outside Git. No report,
compressed object or source bundle is added to the final PR tree.

Later local-package/helper evidence is stored separately under the operator's
same `rwx-ci-pilot-evidence` root: `2026-10-05-local-packages-af0d89bfb2`,
`2026-10-05-helper-refactor-8ec3e62ce9` and
`2026-10-05-helper-refactor-57094f2aee`. The first has a private host mirror;
the helper archives are workstation collections. Their own indices record
validation scope. The initial helper archive preserves the missing-SP1-import
failure; the corrected source adds the filtered library and verifies both hosted
tasks. The full pre-cleanup terminal check snapshot is in the maintenance archive.
Do not infer two-copy custody for archives that only have one recorded copy.

## Collect new evidence

Use a clean pushed SHA with an open PR for Circle's PR-only project setting.
Verify each provider's full SHA, branch, effective settings, selection and final
job/task status before downloading originals. Fetch every API page and every
expected shard. Keep run/job URLs, provider responses and checks alongside the
reports. Avoid secret values and expiring signed download URLs in maintained docs.

Retain complete selection, shard assignments, source/tool/settings hashes,
compiler inventories, initial Go JSON/JUnit, stdout/stderr, per-test logs,
retry/diagnostic histories and job outcomes. Final JUnit or Circle's case API
alone cannot prove zero retries. A diagnostic pass never replaces the initial
failure. Store derived comparison outputs separately from sealed originals.
Report collection must run after failures and cancellations too.

For full Go, select packages from `just list-test-packages`, not a rollup prefix.
Circle's `c-go_fresh_tests=true` enables `-count=1`; RWX verdicts are always fresh.
Acceptance retains each variant's test listing and package manifest. Contracts
retain every feature/profile's complete test files, compiler-bound signatures,
initial reports and diagnostic reruns. Coverage needs Circle's explicit
`c-contract_coverage_replay=true` for identical SHA/feature-derived Solidity and
Go FFI inputs; unseeded hit counts are not an exact replay. Full historical NUT
regeneration uses `c-nut_provenance_full=true` in a normal pipeline, not a separate
replay workflow. L2 comparisons retain the pinned block, state reads and sealed
relay observations. Different upstream request counts alone do not establish
parity or a speed difference; native Foundry caches can serve reads locally.

Use the suite's `compare-*.py --help` and archived invocation/collection index.
For example, standard contract reports use directory inputs and a full SHA:

```bash
python3 ops/ci/migration/compare-contract-suites.py \
  --circle /path/to/circle-report --rwx /path/to/rwx-report \
  --sha FULL_COMMIT_SHA --suite standard --feature main \
  --output /path/to/derived/parity.json
```

The generic `compare-ci.py` remains available for Go JSON/JUnit/case collections,
including rollup CLI compatibility. Its version-1 collection records `metadata`
(provider, SHA, branch, workload, profile, features, routing context, effective
`test_config`), `sources` (format, path, provider `metadata_path`, feature, role,
shard index/total), complete `discovery` with provenance, and measurements. Paths
are relative to the collection JSON. Source metadata must record actual
`sha`, `branch`, terminal `status`. Set completeness only from retained discovery;
matching case reports alone do not prove selection.

```bash
python3 ops/ci/migration/compare-ci.py normalize --input circle.json --output circle.normalized.json
python3 ops/ci/migration/compare-ci.py normalize --input rwx.json --output rwx.normalized.json
python3 ops/ci/migration/compare-ci.py compare --baseline circle.normalized.json --candidate rwx.normalized.json --output parity.json
```

Exit 0 means supplied evidence is equivalent; exit 1 means different or incomplete;
exit 2 means incomparable or rejected input. These tools validate supplied bytes,
not collector identity or provider authentication. Review discovered gaps, skips
and retry histories explicitly; equivalence does not transfer gate ownership.

Measure push/run creation through the final verdict, including queueing, setup,
compilation, transfers and tests. Label unavailable push timestamps, warm/cold
compiler state, resource allocation and every sample. Cached task timestamps can
belong to a prior run: do not count them as current compute time. Archived
benchmark experiments establish observations, not an RWX speed claim. Preserve
all samples outside Git; the chosen 12-shard configuration stays in the operator
guide until a separately justified change.

For the Rust incremental trial, retain both `settings.json` profile overrides and
`cache-source.json` restored/refreshed file counts, sccache statistics and original
feature/test reports. Use the same SHA, partition and runner resources for the
`cargo-incremental=0` baseline and `cargo-incremental=1` candidate. Distinguish
the first incremental-state population from subsequent warm reuse. Measure cache
restoration/output time as well as the command stages; the source timestamp map's
version upgrade requires one conservative refresh. Do not interpret local crate
sccache bypasses as lost registry cache coverage.

## Runtime separation evidence

New native receipts bind `native_policy_sha256` and manifest version 4. Circle
workflow mappings live in `ops/ci/migration/circle-gates.json`; the native manifest
contains no Circle configuration references. Retain the mapping, policy,
definitions and source revision with each comparison. Historical evidence remains
immutable and uses the archived comparer/source revision, including its original
helper paths and receipt schema.

Run permanent and migration tests separately. Retain scenario results and skips,
layer code sizes, routing/adapter output, configuration lint/validation and exact
hosted check identities. Empty originals may be recovered only offline from their
manifest-declared empty hash; retain the audit records. Runtime verification never
writes missing originals. No rerun may replace original failure evidence.

The NUT provenance comparer was previously an inline `--compare` mode of the
runtime runner. It now runs as
`python3 ops/ci/migration/compare-nut-provenance.py --compare CIRCLE RWX --output comparison.json`.
Other moved comparers retain their basenames and argument interfaces.

Pilot-readiness evidence (2026-10-06) is retained outside Git in
`/Users/edward/Workspace/op/rwx-ci-pilot-evidence/2026-10-06-pilot-ready` and its
checksum-verified Hetzner mirror at
`/home/admin/rwx-ci-pilot-evidence/2026-10-06-pilot-ready`. The retrieval bundle
contains source, exact experiment definitions, complete sample/stage JSON, original
stdout/stderr archives, current vault/ruleset metadata, denied credential/cache
probe results, cancellation before/action/terminal observations, combined local
validation and final-head hosted observations. `SHA256SUMS` seals the collection;
verify it before using evidence. No secret values are included.

Changed-crate benchmark run: `294e0a96fdf44c288e2c0d2abd747b45`, source
`3df2ffe9191e710834a567254cbb92461f51f29c`. Both cases used three samples per
mode, alternating order, the same source edits and commands, isolated targets,
pinned sccache, and the permanent per-file timestamp helper. Each original is
manifest-hash verified. Seeds start with empty Cargo targets and are excluded. Rust compiler hits were zero; the second seed can reuse shared C/C++/assembler sccache entries. Archive/restore
measure local uncompressed tar round trips; RWX cache-network transfer and whole
pipeline cost were not measured. The fixture method added an isolated probe
function only in the worker and restored the source afterward; it is not in Git.

RPC denial: `30e1d38ecac44f80b08f06bd9be5a8b8`; protected cache-write probe:
`0505b0c81496457d96bd731fd52b8d76`; cancellation:
`9af37c3158a1494e865d12e7c7c0621d`. These CLI runs do not establish automatic
fork/merge-queue GitHub check association.

The independent cache reader `a9d19a4233c741fc95d6f57a788c9005` uses the same
protected cache name and confirms the CLI writer's harmless marker was not
restored. Retain its definition, terminal result and
`cache-denial-reader-observation.json`; a task's empty list of restored tool
caches alone does not prove that it published no cache version.

The first pilot-readiness push (`d3bfc043fb`) compiled the full Rust test workload
successfully but exceeded the filesystem-layer cap (64.9 GiB inherited + 54.2 GiB
added). Its failed producer, original archive and terminal engine diagnosis are
retained as `rust-build-layer-limit-d3bfc043fb.tar` and the matching coordinator
observation. The corrective target-cache namespace and final-head hosted result
are separate evidence; no original failure was overwritten.

The corrected producer at `6fcfba3e43` published successfully. Its first target
layer uploaded 57,315 MiB in 292 seconds. This is a cold-namespace publication
observation, not a warm-run benchmark. Retain its task log and engine result;
cache transfer remains a performance follow-up.

The merged `develop` revision includes `27766c0270`, which removes historical
proofs v1 and changes the op-reth test recipe's validator selector to `op-reth`.
The first cache-namespace correction still supplied `op-reth-proof-v1`; its nine
op-reth E2E cases failed before node startup. Retain that original report as
`rust-e2e-op-reth-6fcfba3e43-failure.tar`. The shared runner now matches the owning
recipe for both providers. A permanent test compares its sequencer, validator
and proof-history environment with that recipe so future selector changes fail
locally before a hosted run.

At `d256ae4611`, both providers pass all nine corrected op-reth cases. The
same-SHA verdict comparison in `final-d256ae4611/op-reth-verdict-parity.json`
verifies 21 Circle and 24 RWX originals, identical selection/runtime settings,
parallelism eight and no empty-file recovery. This is an op-reth verdict
comparison, not a new complete Rust E2E producer/dependency comparison.

The same revision's Rust test producer returned `ESTALE` on the same cached
metadata in attempts one and two. Each nine-file original manifest is verified;
all 1,344 source timestamps were restored, with zero refreshed files. The
breakpoint observation records readable, shared-inode metadata and probes that
did not reproduce the error. That attempt touched cached files, compiled and
archived successfully, then exceeded the layer cap (64.6 + 45.3 GiB). Retain its
original report, terminal engine message and
`rust-tests-build-d256ae4611-estale-debug-observations.json`. Keep subsequent
unmodified cold/warm validation distinct from this diagnostic attempt. Retain
`cache-output.json` and `cache-publication.json` with the corrective validation.

At `75f51b1427`, the cold producer and fresh verdict passed, followed by an
untouched warm producer retry. The warm verdict did not execute: RWX rejected
its 100 GB disk because dependency layers occupied 88.37 GB and required another
20 GB of scratch space. Retain both attempt engine records and original reports;
compare the subsequent 150 GB validation separately. Successful warm compilation
does not establish a passing warm verdict.
