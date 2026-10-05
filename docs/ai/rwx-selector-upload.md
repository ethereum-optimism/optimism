# Isolated selector uploader

The native selector uploader is implemented and its complete prototype passes.
Same-SHA hosted Circle/native comparison is pending, so this workload remains
unchecked in the parity checklist and adds zero verified coverage.

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

The first complete discovery can install another solc version needed by a
dependency. That changes which installed compatible compiler auto detection
chooses on the following invocation. Preparation retains the complete initial
pass, then repeats discovery with those compilers available and rejects further
changes to their set or bytes. The passing prototype compiled 601 sources,
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
