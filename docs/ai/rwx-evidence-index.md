# RWX pilot evidence archive

PR #23151 covers all 86 baseline CircleCI PR job occurrences. Its generated
reports are archived outside Git. The repository retains this index,
[summary checksums](rwx-evidence-summaries.sha256), and human-written workload
closeouts linked from [the coverage checklist](rwx-parity-todos.md).

## Archive identity and custody

The archive captures source revision
`a1aa49aaf3713a8172f3f615f094488fd8e39c3d` on October 5, 2026.
It contains 216,558 report paths, including complete provider originals,
preparation and test logs, selections, settings, retries, failure fixtures,
provider run/check observations, and the 92 previously committed summaries.
Deduplication and Zstandard compression reduce 38,238,271,163 logical bytes
to 1,347,568,690 bytes of report objects. The manifest and source bundle bring
the complete archive to about 1.5 GB.

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
| `source.bundle` | `49041721241f7f6cb23c1471056125d44ccc13360b4453c12b0c254a61e527a7` |
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
and run the existing `ops/ci/compare-*.py` tools against restored `.ci/` reports.
Provider run/check captures reside under `captures/` or alongside their report
families. They retain original source revisions and run identities.

Historical summary links in the workload closeouts use the immutable
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
