# check-lagoon

Post-Lagoon conformance checks for both features gated by the Lagoon fork:

- Interop cross-chain messaging and the op-interop-filter failsafe; and
- Sequencer-Defined Metering (SDM) PostExec production, RPC encoding, receipt
  fields, replay accounting, independent verifier agreement, and safe-head
  derivation.

## Complete Lagoon smoke test

### `all`

Runs the full `interop-smoke all` suite and one SDM workload on each chain
concurrently. This is the one-command end-to-end Lagoon check: chain identity,
transfers, cross-chain ETH bridging, valid executing messages, invalid-message
reorgs in both directions, PostExec production on both chains, and safe-head
progression are exercised together.

Only one pre-funded account is required. The command derives a stable,
domain-separated SDM test key from `--account`, funds it on both L2s, and then
starts the concurrent workloads with independent nonce domains:

```bash
go run . all \
  --l2-a http://localhost:9545 \
  --l2-b http://localhost:9546 \
  --account <funded-private-key-on-both-chains> \
  --sdm-rollup-rpc-a http://localhost:7545/901 \
  --sdm-rollup-rpc-b http://localhost:7545/902 \
  --sdm-contract-a 0x... \
  --sdm-contract-b 0x...
```

Use `--sdm-account` to supply a stable second key instead of generating one,
or `--sdm-fund-amount` to change its default minimum balance of 0.02 ETH. An
explicit SDM account is topped up only when its balance is below that minimum.
The derived private key remains in memory and is never printed or persisted;
the same funded address is deterministically reused on later runs.

The contract flags are optional; omitting them deploys StateBloat independently
on each chain. The two rollup routes are required because `all` verifies that
each SDM block becomes safe and remains canonical while the invalid-message
workload is causing its expected reorgs.

`all` deliberately does not run the filter-admin `failsafe` command, which
requires a separate op-interop-filter deployment and JWT credentials.

## Interop commands

### `roundtrip`

Bridges ETH A→B and B→A via `SuperchainETHBridge`, relaying each message, for N
iterations.

```bash
go run . roundtrip --config <config.toml>
```

### `failsafe`

Runs the full failsafe lifecycle:

1. Bridge A↔B and expect success.
2. Enable failsafe on every configured filter.
3. Attempt relays in both directions and expect rejection.
4. Disable failsafe.
5. Bridge A↔B again and expect success.

```bash
go run . failsafe --config <config.toml>
```

## SDM commands

### `sdm all`

Optionally calls `admin_setOperatorSdmOptIn(true)`, deploys the StateBloat
workload, submits a dense repeated-slot burst, finds a block with a non-empty
PostExec payload, and runs the checks below.

```bash
go run . sdm all \
  --l2 http://localhost:9545 \
  --account <funded-private-key> \
  --rollup-rpc http://localhost:7545 \
  --l2-verifier http://localhost:9546
```

`--l2-verifier` and `--rollup-rpc` are optional. Supplying them additionally
checks producer/verifier canonical agreement, Lagoon activation at the block
timestamp, and safe-head progression past the SDM block.

The producer RPC needs the `admin_` and `debug_` namespaces. The verifier needs
normal `eth_` receipt access. The rollup endpoint needs
`optimism_rollupConfig` and `optimism_syncStatus`.

The opt-in is intrusive and lives only in the execution client's memory. Use
`--opt-in=false` on a shared network whose operator has already opted in.

If the workload account is empty, it can be funded through the chain's portal:

```bash
go run . sdm all \
  --l2 http://localhost:9545 \
  --account <l2-private-key> \
  --fund-l2 \
  --l1 http://localhost:8545 \
  --l1-account <funded-l1-private-key> \
  --portal 0x...
```

### `sdm block`

Validates an existing block without changing opt-in state or submitting
transactions. No account key is required.

```bash
go run . sdm block \
  --l2 http://localhost:9545 \
  --block 12345 \
  --json
```

### `sdm verifier`

Runs the existing-block checks and requires independent verifier agreement.

```bash
go run . sdm verifier \
  --l2 http://localhost:9545 \
  --l2-verifier http://localhost:9546 \
  --block 12345
```

### SDM assertions

The SDM commands validate policy-independent protocol behavior:

- exactly one trailing type-`0x7D` transaction;
- canonical `0x7D || input` EIP-2718 bytes and `keccak256(0x7D || input)`
  transaction hash, resolvable through raw-transaction, transaction, and receipt RPCs;
- zero sender, gas, and value envelope fields, with zero-or-omitted nonce and recipient fields;
- version-1 payload anchored to the containing block;
- strictly increasing, unique refund indexes that target neither deposits nor
  the PostExec transaction;
- receipt `opGasRefund` values matching the payload;
- zero PostExec `gasUsed`, `effectiveGasPrice`, `blobGasUsed`, `l1Fee`, and
  `l1GasUsed`, with no `opGasRefund`;
- block-scoped L1 fee parameters matching regular receipts in the same block;
- per-receipt DA footprints summing to the block header, excluding deposits;
- `debug_replaySDMBlock` identity, payload, receipt gas, and raw/canonical gas
  accounting;
- optional producer/verifier block, transaction, and receipt agreement; and
- optional Lagoon activation and safe-head progression.

Policy-specific expectations, such as an exact refund amount, deliberately do
not belong in this tool. For compatibility with already-deployed clients, the
checker tolerates the legacy nonzero `gasPrice` field on PostExec transaction
responses. Newly built clients are still required by their serialization tests
to omit this inapplicable field.

## Config

Copy `config.example.toml`, fill in the relevant values, and pass it with
`--config`. CLI flags and `CHECK_LAGOON_*` environment variables override TOML
values.

Interop values remain at the root. SDM values live under `[sdm]`:

```toml
l2-a = "https://chain-a.example"
l2-b = "https://chain-b.example"
account = "<interop-private-key>"

[sdm]
l2 = "https://producer.example"
l2-verifier = "https://verifier.example"
rollup-rpc = "https://rollup.example"
# Used by top-level `all` with root l2-a/l2-b.
rollup-rpc-a = "https://rollup.example/901"
rollup-rpc-b = "https://rollup.example/902"
# Optional for top-level `all`; required by the focused `sdm all` command.
account = ""
l2-fund-amount = "20000000000000000"
opt-in = true
batch-size = 12
slot-count = 20
```

The config contains live private keys and JWT secrets. Every `*.toml` file in
this directory is gitignored except `config.example.toml`.

For devnets with `fund_dev_accounts: true`, the standard Foundry account-zero
key is:

```text
ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
```

## Compatibility

`op-chain-ops/cmd/sdm-devnet` remains available for existing wrappers. New
conformance and automation should use `check-lagoon sdm`; it performs the
strict protocol checks that the legacy workload driver does not.
