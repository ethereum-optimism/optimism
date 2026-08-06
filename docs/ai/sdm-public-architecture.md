# SDM in the public monorepo

Sequencer-Defined Metering (SDM) lets a block producer grant per-transaction gas rebates and commit
them in a synthetic PostExec transaction (type `0x7D`). This document describes what part of that
lives in this repository and what deliberately does not.

Read this before changing anything under `rust/alloy-op-evm/src/post_exec/`,
`rust/op-reth/crates/post-exec-replay/`, `rust/op-reth/crates/sdm-fixture-node/`, or the SDM
acceptance tests.

## The split: mechanism is public, policy is not

The public tree implements the **mechanism** and none of the **policy**.

1. **Stock `op-reth` produces nothing.** `OpEvmFactory<Tx, R>` takes the refund policy as a type
   parameter defaulting to `NullRefundPolicy`
   (`rust/alloy-op-evm/src/post_exec/null.rs`). Lagoon activation and the operator opt-in
   (`admin_setOperatorSdmOptIn`) still drive the shared Produce-mode machinery, but an empty policy
   emits no `SDMGasEntry`, so payload assembly appends no `0x7D`. Inertness is a property of the
   default type, not of configuration — no operator can misconfigure a released binary into issuing
   refunds. `TestSDMOptInIsInertOnStockOpReth` pins this.

2. **No refund policy ships here.** There is no block-warming implementation, and no warming
   constants, in this repository. `PostExecRefundInspector`
   (`rust/alloy-op-evm/src/post_exec/refund.rs`) is the seam a producer implements: the refund
   lifecycle (`begin_tx` / `finish_tx` / `snapshot` / `restore`) plus context-generic observer hooks
   (`inspect_step`, `inspect_call`, `inspect_create`, `inspect_selfdestruct`, `note_account_touch`).
   Keep those hooks even though nothing public observes anything — deleting them deletes the
   extension point. The observer hooks are methods on this trait rather than a `revm::Inspector`
   bound because `R` is a factory type parameter and Rust cannot express
   `for<DB> Inspector<OpEvmContext<DB>>`.

3. **Everything that consumes a `0x7D` block stays public and fully covered.** Verification,
   derivation (op-node and kona-node), receipt settlement and `opGasRefund` exposure, RPC, and the
   Cannon/kona fault-proof path all handle externally produced, consensus-valid SDM blocks. So do
   the malformed-payload rejection tests. This is the consensus-critical half and it belongs here.

A refund policy must never be added to a released public binary. Adding one to `alloy-op-evm`,
`op-reth`, or any published crate is the one change this architecture exists to prevent.

## The test-only fixture producer

Public acceptance tests need real `0x7D` blocks, which needs a producer. That is
`rust/op-reth/crates/sdm-fixture-node/` — the non-published `op-reth-sdm-fixture` binary.

Its `FixedRefundPolicy` refunds **exactly one gas per committed normal transaction**, and zero for
deposits and the `0x7D` transaction itself. It is stateless, observes nothing, and has no
configurable amount. That makes the Go-side oracle trivially exact (`entry.GasRefund == 1`, receipt
`opGasRefund == 1`) and independent of any real policy, so a policy change cannot churn public tests.
It is deliberately **not** an example of a production policy.

The fixture is selected only for acceptance-test sequencers, via
`OpRethWithBinary("op-reth-sdm-fixture")`. Verifiers and every proof consumer stay stock.
`NewFixtureSingleChainFaultProofSystem` verifies the running sequencer's `web3_clientVersion`, so
selection failing silently (preset option accepted but never applied to the sequencer) fails the test
rather than passing vacuously.

### Release exclusion is enforced, not assumed

The fixture must be `publish = false`, absent from workspace `default-members`, and absent from
`docker-bake.hcl`, `.github/images.json`, `.github/images.apko.json`, `apko/`, `melange/`, `ops/`,
`k8s/`, and `devnets/`. There is no fixture image.
`.circleci/scripts/check-sdm-fixture-release-exclusion.sh` enforces all of that, and
`check-sdm-fixture-test-names.sh` enforces that the required SDM acceptance test names still exist.
Both run in the `sdm-fixture-guards` job and are required by `ci-gate`.

Note what the name guard cannot do: it checks that test *names* exist, not that they still assert
anything. Weakening an SDM test's assertions passes CI.

## `debug_replaySDMBlock` is structural only

`rust/op-reth/crates/post-exec-replay/` re-executes a block's non-`0x7D` transactions with
`PostExecMode::Disabled`, so receipts report each transaction's **raw** (pre-rebate) gas regardless
of which policy produced the block. It reports that raw gas next to the block's claimed refunds and
flags structurally invalid claims: duplicate or out-of-range payload index, a refund targeting a
deposit or the `0x7D` transaction, or a refund exceeding the transaction's raw gas.

It does **not** recompute what refunds should have been, and must not start: recomputation requires a
policy. There is therefore no synthesized payload and no per-refund attribution breakdown in the
public response. A producer running a real policy owns the policy-aware replay that reports
embedded-versus-recomputed agreement.

Replay is counterfactual by construction — it re-executes without the rebates the canonical block
applied, so balance-dependent execution can diverge from canonical. That is inherent to the tool.

## Proof-history launch wiring is duplicated on purpose

`reth_optimism_node::proof_history::launch_node` and the fixture's `launch_fixture_node` are
near-identical: storage-version dispatch, the proofs-history ExEx, and the `eth_getProof` /
`debug_executePayload` RPC overrides.

They are not shared, and the reason is worth recording so nobody repeats the attempt. reth
parameterizes add-ons and the entire RPC stack by a node's component set, so a launcher generic over
the node type has to restate reth's whole `EthApi` / `RpcNodeCore` bound chain (~25 bounds) in
op-reth production code — a worse artifact than the duplication. What can be shared is shared:
`spawn_proofs_db_metrics` and `OpNode::payload_builder`. The fixture launches without reth's debug
capabilities because `FixtureOpNode` does not implement `DebugNode`; acceptance tests never use them.

**Any change to stock proof-history wiring must be mirrored in the fixture.** This has already
bitten once: a fixture without proof history made `kona-host` stall until the SDM proof test timed
out after ~18 minutes, starving its whole CI shard. Adding a `ProofsStorageVersion` variant is caught
by the compiler; changing *what gets installed* is not.

## Vendored builders remain public and distributed

`rust/op-rbuilder/` and `rust/rollup-boost/` remain vendored workspaces. Their CI checks, release
builds, images, apko/melange definitions, devstack launchers, operator documentation, and the
`SingleChainWithFlashblocks` preset remain supported. Generic Flashblocks stream and transfer tests
continue to exercise both binaries.

Public op-rbuilder uses the same `NullRefundPolicy` as stock op-reth. Lagoon activation and the
operator gate remain wired, but opting in cannot emit a `0x7D` or refund gas. The builder still
carries the generic opaque policy snapshot across flashblocks so downstream policy integrations can
preserve block-scoped state without reintroducing warming policy into this repository. Rollup-boost
continues to proxy, select, seal, and forward payloads without owning SDM economics.

Also preserved: op-conductor's rollup-boost client, health, and WebSocket integrations;
`reth-optimism-flashblocks`; op-alloy Flashblocks wire types; `op-service` consumers; and all
packaging and operator documentation for the vendored binaries.

## Coverage that intentionally left the public tree

These assertions were removed here because they test policy, and belong to whoever owns the policy:
exact warming constants and exclusions; per-block slot/account uniqueness and first-warmer
attribution; the storage-refund breakdown (same-slot versus many-slot totals); phantom-warming
rollback against a real policy; and SDM-specific live stream-to-canonical materialization. Generic
Flashblocks stream and transfer coverage remains public and unchanged.

Public coverage that replaces them, and must not regress:

| Property | Public test |
|---|---|
| Fixture payload, receipt, and gas accounting agree; verifier matches producer | `TestSDMFixturePayloadReceiptAndAccounting` |
| Active Lagoon plus opt-in still emits nothing on stock `op-reth` | `TestSDMOptInIsInertOnStockOpReth` |
| Operator gate controls production | `TestSDMFixtureOperatorOptInControlsProduction` |
| Span and singular batch derivation of a non-empty `0x7D` block | `TestSDMPostExecBlockDerivesAndChainProgresses` |
| Isolated stock verifier reproduces the target | `TestSDMPostExecBlockDerivesOnIsolatedVerifier` |
| Real pre/post-Lagoon activation boundary | `TestSDMActivatesAtLagoonBoundary` |
| L1 derivation through kona/Cannon over an actual non-empty SDM target | `TestInteropSingleChainFaultProofsWithSDM` |
| Candidate rollback cannot leak block-scoped policy state | scripted-policy unit tests in `alloy-op-evm` |

The last row is why `PostExecRefundInspector::snapshot`/`restore` and the scripted test policies
exist: they cover the rollback *mechanism* without needing a real policy.

## Working rules

- Interop infrastructure and Policy Engine contracts are orthogonal to SDM. Do not modify their
  production, preset, runtime, contract, or snapshot files for an SDM change unless root-cause
  analysis proves no SDM-owned alternative exists. Never regenerate Policy Engine artifacts for an
  SDM-only change.
- Retry unrelated flaky interop or Policy Engine CI failures before changing code.
- `TestInteropSingleChainFaultProofsWithSDM` cannot run locally without MIPS prestate artifacts in
  `op-program/bin`; it needs CI.
