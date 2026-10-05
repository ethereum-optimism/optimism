# Isolated selector uploader

The complete native selector uploader now passes strict same-SHA original
Circle/native parity at `63844aed622385299088d716152d8105d5f88b66`. This closes
one Main occurrence. The [retained comparison](rwx-selector-upload-evidence/parity.json)
binds every original compiler, HTTP, database and process report.

The optional `optimism-selector-upload-shadow` follows shared `run-main` routing.
It executes the original `just update-selectors` recipe, whose command remains
`forge selectors up --all`, in the default Foundry profile. Its only destination
is an isolated official Sourcify 4byte service and fresh PostgreSQL database.
Circle's normal publisher retains its original command and destination.

The unmodified service is built from Sourcify release `sourcify-4byte@1.1.16`,
source `9282528d8c210f22287761c26c4232c15f66585d`, using its pinned Node Dockerfile
and lockfile. The test database uses a pinned PostgreSQL image and the original
committed SQL schema. Its upload role can select and insert signatures; it has no
administrative privileges. Neither container exposes a host port. The client
joins their otherwise unroutable network namespace, verifies blocked IPv4/IPv6
egress and loopback DNS, and runs without root or new privileges. A private TLS
proxy retains complete original POSTs and API lookups while forwarding them to
the actual service. Private certificate keys are removed rather than exported.

The reusable image producer retains Docker/BuildKit data using RWX
`docker: preserve-data`. ABI preparation runs on 16 CPUs / 32 GiB separately
from the fresh uploader on 8 CPUs / 16 GiB. Preparation retains every compiled
source, declaration, ABI, compiler unit, profile and exclusion, including empty
libraries and contracts outside the upload source path. Every actual solc binary,
tool, input byte, executable mode, symlink and recursive submodule is bound to
its revision and hashes. The upload rejects stale preparation or restored image
identities, validates all actual imported signatures independently with Cast,
and reads every signature back through the real API. Test verdicts and uploads
are never cached. Protected `develop` warming targets only images and ABI
preparation and executes zero tests or uploads.

The ABI oracle enables `FOUNDRY_BUILD_INFO` only while retaining original
compiler inputs and outputs; the actual publisher uses its original setting.
Foundry's artifact source IDs can come from another compilation profile of the
same source/version. Validation preserves these original IDs while binding each
artifact to the exact build unit's own source map. The pinned compiler
[implementation](https://github.com/foundry-rs/foundry-core/blob/327739f0d2b848725b462271276f27aecd8b8ff9/crates/compilers/crates/compilers/src/artifact_output/mod.rs#L905)
selects artifact IDs without considering their profile. Complete real two-job
compiler originals cover this behavior; corrupt unit mappings still fail.

The graph requires solc 0.8.30 in addition to Circle's four installed versions.
Preparation now ensures all five are available before compiler auto detection,
so a cold worker chooses the same versions as a restored SVM cache. Both complete
discovery passes remain retained. Validation derives the used compiler set from
every original unit, separately from the complete installed inventory, and
rejects changed compiler bytes. The passing prototype compiled 601 sources,
retained 2,173 declarations initially and selected 167 declarations after
resolution stabilized. The actual publisher imported all 979 function/error
and 141 event signatures; the fresh database and complete API readback contained
exactly 1,120 unique signatures.

Prototype run
[`aec697d0`](https://cloud.rwx.com/optimism/runs/aec697d0cb08440d88f49ac515b19729)
executed complete preparation and upload freshly in 24 and 9 seconds of task
execution, respectively. These are prototype observations, not a provider speed
comparison. It also executed real import, duplicate import, denied write and
client timeout fixtures. Earlier failures remain retained: an incomplete client
PATH, an unused registry gitlink, profile-local source IDs, and compiler
resolution/status-stream validation. The prototype injected uncommitted helpers
into source `4ae28fd9`, and therefore adds zero same-SHA coverage.
The [preflight index](rwx-selector-upload-evidence/preflight.json) binds all 4,531
original files through their retained sealed manifest and archive hashes.

Circle's `c-selector_upload_replay` parameter defaults to false. An explicit API
run on `codex/rwx-ci-pilot`, with `main_dispatch=false`, selects only the original
uploader in private test mode. Other branches and normal pushes cannot select
this replay. Complete compiler, HTTP, database, service and process originals are
stored even after failures. The next validation is to run that replay and the
native definition on the same pushed revision, compare all originals, then
verify unchanged-input cache reuse, compiler-only warming and terminal checks.

Both hosted providers executed the complete uploader successfully at
`86f4ba931a0a8a4ef304767b4a802125253d8f7f` (native run `786f16f0`, Circle
pipeline 135611/job 5637452). Circle's directory artifact uploader omitted 49
empty original files. The replay now also archives the complete report tree;
coverage remains pending until a fresh same-SHA comparison validates that archive.

The compiler fixtures were recaptured in native run `af521751` with function
argument names that follow the repository's Solidity style rules. Both profiles'
exact unit mappings and their shared artifact ID remain covered regardless of
compilation completion order. The earlier captures and initial Semgrep findings
remain retained with the prototype evidence.

The first native execution at `e3fab86e` restored an earlier preparation report
through its incremental tool cache and correctly rejected that directory before
uploading. Source/submodule preparation now runs separately. The ABI tool cache
has a new identity and retains only compiler outputs and solc installations;
reports travel as explicit sealed artifacts. The old failure remains retained,
and cache validation includes a different-revision replay as well as unchanged
inputs. The comparator independently reconstructs both complete compiler
catalogues, original POSTs, database insertion and every API readback. Real
provider process captures cover Circle's UID 1001 and RWX's UID 1000; both must
remain unprivileged and isolated.

At `01150e71`, fresh native preparation and upload succeeded, and Circle retained
all 4,531 report files including empty files. The complete compiler, protocol and
database comparisons agree. Circle's API nevertheless truncated the preparation
and directory-upload console logs at 400 kB, so strict hosted parity remains
pending. The replay now stores the complete preparation stream inside its archive
and uploads two single archives instead of thousands of duplicated directory
artifacts. The first truncated originals remain retained.

The next complete comparison at `b472a22e` retained untruncated Circle logs but
caught a real compiler-cache difference: native preparation already had 0.8.30,
while Circle's first pass installed it during discovery. Their initial compiler
catalogues differed. Both complete originals remain retained and add no coverage.
The pinned prerequisite above removes that cold/warm difference before a new
same-SHA replay; it does not discard or normalize either compiler catalogue.

The unchanged-input CLI warm run `c5f99254` reused tools, contract tools, source,
registry images and ABI preparation through actual native cache hits. Its exact
source init remains `01150e71`; the CLI run's root commit metadata is null, so
this adds no hosted same-SHA coverage. Only routing executed freshly. No helper,
test or uploader task existed in the targeted run. The
[warm index](rwx-selector-upload-evidence/compiler-warm.json) binds its complete
original run metadata and source identities. Cached tasks' displayed execution
durations belong to their original executions and are not warm-run timings.


The corrected automatic native run
[`bb3a5d4a`](https://cloud.rwx.com/optimism/runs/bb3a5d4a02c945cd9cd1177ba37fc94e)
and Circle pipeline 135624/original [job 5637952](https://circleci.com/gh/ethereum-optimism/optimism/5637952)
pass complete parity at `63844aed`. Preparation installed all five pinned
compilers before discovery. Both complete initial and stable compiler catalogues
agree, with 601 compiled sources, 167 selected declarations, 979 function/error
and 141 event signatures. The one actual POST, all 1,120 fresh database rows and
every API readback agree. Native preparation/upload executed freshly in 23/9
seconds; these are task observations, not a provider speed claim. Circle retains
all 4,520 original report files and native all 4,517, including provider-specific
process captures and empty files; all original Circle API logs are untruncated.
The prior failed and truncated comparisons remain retained.

Normal pilot pushes now select this same private replay alongside the full PR
workflow. This permits one verification batch for the final selector, L2 fork
and Contracts gate changes. Other branches and publishers retain their routing.
