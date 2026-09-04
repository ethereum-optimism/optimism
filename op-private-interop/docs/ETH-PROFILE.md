# Private ETH profile

This profile patches the existing private interop implementation. Native currency and L1 deposits
use ordinary ETH semantics. Private messaging requires positive transaction gas price, excluding
deposit-originated sends, resends and messenger relays throughout nested application calls.
Sequenced application interop retains the existing messenger interface.

`SuperchainETHBridge` has independent `allowedSendChain` and `allowedRelayChain` mappings. Both
default empty. The existing L2 ProxyAdmin owner calls
`setChainPermissions(uint256 chainId, bool allowSend, bool allowRelay)` on the bridge predeploy.
Sending checks the destination before burning; relaying checks the authenticated messenger source
before releasing liquidity and retains the remote bridge identity check. Approval records a
governance decision about shared backing; the contract does not discover L1 lockbox membership.
For A → B, A must allow sending to B and B must allow relaying from A. Disable sends and drain
pending transfers before revoking relays; immediate revocation can strand transfers.

Keep both native route lists empty for this private profile: the existing renderer/replay contract
rejects native bridge messages. Allowlisting a peer alone does not add private native bridge support.
Generic application messaging and native bridge permissions are independent.

## Build and generate artifacts

From `packages/contracts-bedrock`:

```sh
mise x -- just build-source
```

From the repository root:

```sh
python3 op-private-interop/genesis/export-bytecode.py
go run ./op-private-interop/cmd/genesis \
  --genesis /path/to/source-genesis.json \
  --rollup /path/to/source-rollup.json \
  --out /path/to/new-artifact-directory \
  --artifact-base-url https://artifact-host/immutable-version
```

Use matching ETH deployment artifacts with interop active at genesis. The command refuses CGT
sources, unknown messenger/bridge releases, preconfigured policy storage, mismatched source hashes,
and existing output directories. It preserves unrelated allocations and ordinary deposit machinery.
It installs the two pinned implementations, sets the messenger policy slot, recomputes the private
genesis hash and derives the public projection. Generated private and projection rollup files each
reference their corresponding genesis. `report.json` records hashes, bytecode commitments and file
checksums. L1 backing validation is explicitly outside this offline transformation.

The messenger policy is stored at `keccak256("privateinterop.requirePaidMessages")` on its proxy.
There is no public setter: it is a genesis/upgrade policy. The default zero word preserves ordinary
messaging on other chains. Unlike the messenger's optional policy, the new bridge implementation
always requires explicit route permissions; installing it on another chain also starts with no
routes unless they are configured. Coordinate any existing bridge upgrade accordingly.

## NetChef integration

The output includes fragments for the current runtime-projection topology:

- Merge `netchef-chain-values.json` into the private chain inventory entry's `values`.
- Merge `netchef-batcher-service-values.json` into its batcher service's `spec.values`.
- Merge `netchef-supernode-service-values.json` into the shared supernode service's `spec.values`.

Keep existing service values and topology. Private ELs, the projection EL, batcher and supernode all
receive the same `private-genesis.json` source. The projection EL retains `--rollup.private`; the
supernode retains its private-chain flags and derives the projected rollup config itself. The
chain-level supernode rollup override is intentional: NetChef merges chain values after service
values for the supernode. All consumers must run images built from this patch, including op-reth's
updated source-code hash validator. No new NetChef schema is required.

The materialized `projection-genesis.json` and `projection-rollup.json` are inspection/parity
artifacts in this mode. Do not feed an already projected genesis to `--rollup.private`.

Upload files under an immutable path and verify downloaded checksums before rollout. NetChef
inspection can regenerate the source files, so keep this transformation as an explicit subsequent
artifact step. Changing the source deployment inputs requires regenerating all outputs together.

A changed genesis requires a fresh chain database or a deliberate devnet reset. A new download URL
or `RETH_GENESIS_REDOWNLOAD` does not migrate an initialized database. The overlay also cannot change
the existing L1 portal's asset mode or backing. Verify isolated backing on L1 before funding the
private chain; a dedicated lockbox must have the intended portal authorization set.

## Validation and remaining integration work

Contract tests cover route directions, authorization, relay authentication, liquidity protection and
private messenger policy. Go tests check source validation and artifact transformation; Go and Rust
share golden private/projection hashes. The private devstack preset applies this profile to an ETH
source before starting its nodes. Acceptance tests cover deposit funding and forced-message rejection,
alongside the existing bidirectional application messaging test.

The full optimized Solidity regression run passed 2,334 tests and found four deployment gas-margin
failures. After raising the messenger and bridge deployment budgets, all four failed cases passed
in a focused rerun. Go package tests and all six Rust projection tests passed. A clean optimized
source build passed the full interface check and regenerated ABI, storage and semver snapshots;
the embedded runtime bytecode matches that build. Formatting, contract validation and Semgrep also
passed.

The following acceptance tests passed locally on September 4 with the rebuilt op-reth binary:

- `TestPrivateETHDepositCannotSendInterop`: L1 ETH funds the private chain; a forced messenger send
  reverts with the configured policy error and emits no logs.
- `TestPrivateInteropMessengerBothDirections`: ordinary application interop works into and out of
  the private chain with the existing message-position resolver.
- `TestPrivateOutageDoesNotBlockPublicProgress`: after stopping the private CL, EL and batcher, the
  projection and counterparty advance beyond the last private timestamp. The test uses a ten-L1-block
  sequencing window, verifies a forced public inbox call cannot execute a message at zero base fee,
  and checks that a fallback block contains only system/deposit transactions and no logs. Production
  keeps the normal sequencing window.

The generated overrides were also rendered with NetChef's existing node chart 0.46.1: the genesis
download URL and decoded rollup ConfigMap refer to the generated private artifacts. This is local
validation, not a live devnet rollout.

`RUST_JIT_BUILD=1 mise x -- go run ./op-up --private-interop --smoke` also passed. It runs the
chain-ops `interopsmoke` suite in the process that registered the private message-position resolver.
Native ETH bridging is explicitly skipped for private pairs. A standalone remote private-pair smoke
remains unsupported because it does not have that resolver; this run does not verify a live NetChef
deployment.

The renderer currently computes log positions from filtered private receipts; deposits that produce
different logs in the projection need an explicit resolution rule. Simple EOA deposits produce no
logs, but this is not a general claim of arbitrary deposit-call compatibility.

V1 remains operator-attested. This patch does not implement private-state proofs or private withdrawal
settlement; the existing proof-bytes extension remains available for later verification rules.
