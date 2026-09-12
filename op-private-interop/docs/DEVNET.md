# Private interop devnet validation

Use one source revision for op-node, op-batcher, op-supernode, op-reth and the custom contract
artifacts. The projection deposit execution rule changes consensus from the contract-guard
prototype: use fresh databases for that migration. This page describes preparation and validation;
it does not imply a particular remote environment has been deployed.

## Versioned local bundle

From a clean checkout, run `op-private-interop/scripts/build-devnet.sh`. It builds the Go
components, op-reth and contract artifacts into `.devnet-bin/private-interop-bundle/<revision>`
and writes a SHA-256 manifest with source revision, toolchain versions and Rust build profile.
The default Rust profile is `dev`, matching the local acceptance setup; use `RUST_PROFILE=release`
for an optimized deployment build. A bundle is host-specific and is not a published container
image. Build/deploy images from the same revision for the selected environment.

## Durable state and endpoints

| Component | Configuration / state |
| --- | --- |
| Private EL | Ordinary private-chain genesis and persistent execution database. |
| Projection EL | Same source genesis with `--rollup.private`; persistent execution database. |
| Supernode | Private chain/genesis configuration; persistent data directory including denial and interop transition databases. |
| Private LightCL | Its own L1 RPC, private EL engine endpoint, `--l2.follow.source=<supernode>/<id>/claimed`, and `--l2.follow.source.recovery-path=/data/private-recovery.db`. |
| Private batcher | Existing private-interop flag group, private EL/CL and public projection EL/ordinary CL endpoints. |

The recovery journal contains private header metadata and durable public/private replay checkpoints. Keep it on the private node's persistent
volume. Preserve it with the private EL database; recovery authenticates surviving prefixes using
this history before rebuilding deposit-only blocks from L1. On restart, LightCL revalidates saved replay
progress against both canonical chains before retaining newer valid private blocks. A changed recovery
prefix or public schedule still triggers rewind. Normal public LightCLs do not need the journal.
In-flight replacements are revalidated when a new recovery plan arrives and again when execution
completes, before advancing safety or resuming sequencing. A stale completion does not authorize
retaining its unsafe block, even when an earlier replay checkpoint remains canonical.

The `/claimed` endpoint reports private commitments and recovery schedules. The ordinary
`/<chain-id>` supernode route reports projection safety. Do not interchange them.

## Local verification

From the repository root, with the pinned mise toolchain and a built op-reth:

```sh
export RUST_JIT_BUILD=1
export RUST_BINARY_PATH_OP_RETH="$PWD/rust/target/debug/op-reth"
mise exec -- go test ./op-private-interop/... ./op-supernode/supernode/activity/claimfollow \
  ./op-supernode/supernode/chain_container ./op-node/rollup/driver ./op-chain-ops/interopsmoke
mise exec -- go test ./op-acceptance-tests/tests/interop/private-interop -count=1 -timeout=30m
```

The production-cadence soak is separate because it takes roughly twenty minutes. It accumulates
two 300-block ranges with publication stopped, then checks catch-up with a 3,600-L1-block sequencing
window and ordinary two-second private blocks:

```sh
PRIVATE_INTEROP_SOAK=1 mise exec -- go test ./op-acceptance-tests/tests/interop/private-interop \
  -run '^TestPrivatePublicationCatchesUpAtProductionCadence$' -count=1 -timeout=30m -v
```

The shorter outage/recovery and batcher-rotation fixtures use a ten-L1-block sequencing window.
They exercise expiry without waiting for the production window. Batcher rotation can expire
unpublished old-key history; the rotation test does not establish uninterrupted handover.

`TestPrivateRecoveryFollowsL1Reorg` removes the L1 origin of an already executed recovery block
while the rejected range is still incomplete. `TestPrivateRecoveryRestartDiscardsReorgedProgress`
repeats this with LightCL stopped across the reorg, reusing its execution database and journal.
Both require revocation of the old recovery snapshot, replacement of abandoned private history,
preservation of finalized ancestry, and fresh transactions/messages reaching canonical cross-safe
history. These fixtures use the short test window, not production-window recovery timing.

For an interactive temporary local devnet:

```sh
mise exec -- go run ./op-up --private-interop --dir "$PWD/.devnet/op-up"
```

`op-up` creates an in-memory devnet. Its directory is a cache/scratch location, not a persistent
chain deployment. Stop it with SIGTERM or Ctrl-C when finished. It prints direct, batch-capable
RPC endpoints and a publicly known local test account; that account is only for disposable devnets.

## Smoke against a deployed pair

Build the standalone tool:

```sh
mise exec -- go build -o .devnet-bin/interop-smoke ./op-chain-ops/cmd/interop-smoke
```

Supply a funded test account through `INTEROP_SMOKE_SMOKE_PRIVATE_KEY` using the environment's
secret mechanism. With the four RPC variables set to the target environment:

```sh
.devnet-bin/interop-smoke all --private-pair-b \
  --l2a-rpc "$PUBLIC_A_RPC" --l2b-rpc "$PRIVATE_B_RPC" \
  --projection-b-rpc "$PROJECTION_B_RPC" \
  --projection-b-rollup-rpc "$PROJECTION_B_ROLLUP_RPC" \
  --private-position-timeout 15m
```

This sends transactions and deliberately submits an invalid message to the public counterparty.
Use a disposable devnet. Native ETH bridging is reported as skipped for the private profile.
The default resolver uses the standard messenger/inbox export policy; deployments with extra
emitters must supply matching resolver configuration programmatically.

For each deployment, record the source revision, image digests, contract bundle hash, chain IDs,
genesis hashes, RPC routes and validation results. Verify a nonzero claimed checkpoint advances,
run the standalone smoke, then exercise restart using the same persistent volumes. Monitor
private unsafe/local-safe lag, projection local/cross-safe lag, recovery progress and batcher
publication errors. Alert on sustained lack of progress relative to the configured publication
cadence and sequencing window, rather than treating the intentional cadence lag as a failure.

### Production-cadence recovery and reorg soak

The opt-in recovery soak keeps two-second L2 blocks, 300-block private ranges,
and the full 3,600-L1-block sequencing window. It first catches up two unpublished
private ranges and establishes valid messaging in both directions. It then reorgs
L1 during partial private recovery twice, including a LightCL restart with the
same database and journal, followed by completed-recovery restart preservation,
a separate supernode restart, and fresh cross-safe traffic. Allow 90–120 minutes:

```sh
PRIVATE_INTEROP_REORG_SOAK=1 mise exec -- go test \
  ./op-acceptance-tests/tests/interop/private-interop \
  -run '^TestPrivateRecoveryAcrossL1ReorgAtProductionCadence$' \
  -count=1 -parallel=1 -timeout=150m -v
```

Use the matching native op-reth and default-profile contracts as above. Full hashes,
receipts, heads, recovery snapshots, and phase timing are logged as JSON evidence.
The L1 reorg revokes the rejected range before its window expires; this tests
recovery across a reorg at production cadence, **not** recovery through full-window
expiry. With this fixture's six-second L1 blocks, expiry alone takes about six
hours. After revocation, a new plan may permit ordinary private sequencing; the
interrupted recovery block itself is required to have deposit-only content.
The post-recovery restart in this soak happens after the reorg has revoked the
old range. The existing shortened-window
`TestCompletedPrivateRecoveryRestartKeepsNewPrivateBlocks` separately covers
restart while a completed prefix is still advertised by the source.
