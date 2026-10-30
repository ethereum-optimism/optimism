# check-lagoon

Smoke tests for interop cross-chain messaging, the op-interop-filter failsafe,
SDM PostExec conformance, and canonical RPC replay settlement.

## Commands

### all

Runs the `interop-smoke all` suite and one SDM workload per chain concurrently.
It derives an SDM account from `--account` (or uses `--sdm-account`) and tops
it up to `--sdm-l2-fund-amount` on both chains. Add `--sdm.check-rpc` to require
the canonical RPC replay checks below on both workload blocks. This is opt-in:
a passing default `all` result does **not** establish generic tracing correctness.

```bash
go run . all \
  --l2-a http://localhost:9545 --l2-b http://localhost:9546 \
  --account <key> \
  --sdm-rollup-rpc-a http://localhost:7545/901 \
  --sdm-rollup-rpc-b http://localhost:7545/902
```

### roundtrip

Bridges ETH A→B and B→A via `SuperchainETHBridge`, relaying each message, for N iterations.

```bash
go run . roundtrip --config <config.toml>
```

### failsafe

Full failsafe lifecycle test:
1. Bridge A↔B (expect success)
2. Enable failsafe on all configured filter instances
3. Attempt relay in both directions (expect rejection, 3 attempts each)
4. Disable failsafe
5. Bridge A↔B again (expect success)

```bash
go run . failsafe --config <config.toml>
```

### sdm all | block | verifier

`sdm all` deploys StateBloat, submits a dense workload, and validates the block
that includes most of it. `sdm block` validates an existing block;
`sdm verifier` also requires `--sdm-l2-verifier` agreement.

```bash
go run . sdm all --sdm-l2 http://localhost:9545 --sdm-account <key>
go run . sdm block --sdm-l2 http://localhost:9545 --sdm-block 12345
```

The producer needs the `admin_` and `debug_` namespaces. `--sdm-rollup-rpc`
adds Lagoon-activation and safe-head checks; the safe-head wait is bounded only
by the command timeout (10 minutes for `sdm`, 25 minutes for `all`). The checks fail if the producer
has not opted in to SDM; `--sdm-opt-in` enables it for the duration of the
check and switches it back off afterwards.

Checked for every SDM block:

- one trailing type-`0x7D` transaction with canonical encoding, hash, and zero envelope fields;
- a version-1 payload anchored to the block, with strictly increasing, non-zero refund entries that target neither deposits nor the PostExec transaction;
- receipt `opGasRefund` values matching the payload, and zero gas and L1 fee fields on the PostExec receipt;
- block-scoped L1 fee parameters and per-receipt DA footprints summing to the header;
- `debug_replaySDMBlock` gas accounting;
- optionally, verifier agreement, Lagoon activation, and safe-head progression.

The replay runs with post-exec accounting disabled, so replayed balances never
include refunds from earlier transactions in the block. A transaction whose
execution reads such a balance (a refunded sender or a fee vault) replays with
different gas and fails the replay gas checks even though the block is valid.
The StateBloat workload never reads balances, but a block from `sdm block` or
`sdm verifier`, or one shared with third-party transactions, can.

### sdm rpc — canonical RPC replay settlement

Read-only regression gate for [protocol-team#485](https://github.com/ethereum-optimism/protocol-team/issues/485).
Correct receipts do not establish correct traces: generic replay can omit SDM
settlement even when consensus, receipts and stored state are correct.

```bash
# No private key, funding, transactions, or operator opt-in changes.
go run . sdm rpc --sdm.l2 http://localhost:9545 --sdm.block 12345 --sdm.json

# Check both producer and independent verifier, preserving both failure reports.
go run . sdm rpc --sdm.l2 http://localhost:9545 \
  --sdm.l2-verifier http://localhost:9645 --sdm.block 12345 --sdm.json

# Alternatively add the gate after the existing conformance/workload checks:
go run . sdm all --sdm.l2 http://localhost:9545 --sdm.account <key> --sdm.check-rpc
go run . sdm block --sdm.l2 http://localhost:9545 --sdm.block 12345 --sdm.check-rpc
```

The selected block must contain actual SDM refunds. Checks cover:

- `debug_traceTransaction` default and call tracers: each ordinary transaction's
  gas and success/failure agree with its **individual** canonical receipt, not
  cumulative block gas. Includes unrefunded transactions and later targets.
- `debug_traceTransaction` prestate tracer in diff mode: balance changes agree
  with `debug_accountAt` immediately before/after that transaction. Checks the
  sender, coinbase, base/L1/operator fee vaults and all accounts in the diff.
  Reused senders are checked at their transaction boundaries, never against
  block-final balances. Storage/nonce/code diffs are not covered.
- `debug_traceBlockByHash` and `debug_traceBlockByNumber` call tracers: trace
  count/order, available transaction hashes, per-transaction gas and status,
  including deposits and the trailing PostExec transaction.
- Indexed `debug_traceCall`: after every refunded transaction, an ephemeral
  code override reads sender/fee-vault balances. Its output must match the
  settled canonical prefix. This tests a synthetic call on settled state,
  not whether a recorded refund is assigned to that synthetic call.
- The block must remain canonical throughout the check.

`debug_accountAt` is a required **executor-backed oracle** on op-reth; a missing
method is a failure, not a skipped check. The gate does not use
`debug_replaySDMBlock` as its oracle: that diagnostic endpoint intentionally
omits settlement and may fail for refund-dependent transactions. Consequently
standalone `sdm rpc` can check those blocks without first passing diagnostic
replay conformance. Its optional `--sdm.rollup-rpc` checks Lagoon activation only,
not safe-head progression. `--sdm.check-rpc` on other commands still runs their
normal conformance and configured safe-head checks first.

On failure the command exits nonzero; `--sdm.json` still emits per-check results
under `rpc_replay` and, when selected, `verifier_rpc_replay`. Top-level `all`
uses `--json` and preserves these within each chain's result. Unsupported methods
and RPC failures are explicit errors. The normal command timeout bounds the
check. Trace failures do not cause new workload transactions to be submitted.
The work scales with block size; prefer small, purpose-built SDM blocks.

This is **not** the complete #485 acceptance matrix. It does not cover parity
`trace_*`, `eth_callMany`/`debug_traceCallMany`, OTS, JS tracers, intermediate
roots, or raw/bad-block diagnostics. It does not generate a refund-dependent
insufficient-balance workload or independently prove `debug_accountAt` correct.
Do not equate passing this gate with completing the issue or fleet rollout.

## Config

Copy `config.example.toml` to a local file, fill in your values, and pass with
`--config`. The config holds live secrets (the `account` private key and filter
`jwt-secret`s), so every `*.toml` in this directory is gitignored except
`config.example.toml` — your copy can use any name and won't be committed:

```toml
l2-a = "https://your-chain-a-rpc"
l2-b = "https://your-chain-b-rpc"
account = "<hex-private-key-no-0x>"
relay-timeout = "2m"
iterations = 3          # roundtrip only
propagation-wait = "6s" # failsafe only

[filter]                # failsafe only
admin-rpc  = ["http://filter-host:8420"]
jwt-secret = ["0x<32-byte-hex-secret>"]

[sdm]                   # sdm and all; see config.example.toml
l2 = "https://producer-rpc"
account = "<hex-private-key-no-0x>"
```

The `account` key uses the foundry dev key on devnets with `fund_dev_accounts: true`:
```
ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
```

CLI flags and `CHECK_LAGOON_*` env vars override values in the config file.
