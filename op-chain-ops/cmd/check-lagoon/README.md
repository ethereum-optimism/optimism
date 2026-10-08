# check-lagoon

Smoke tests for interop cross-chain messaging, the op-interop-filter failsafe,
and SDM PostExec conformance.

## Commands

### all

Runs the `interop-smoke all` suite and one SDM workload per chain concurrently.
It derives an SDM account from `--account` (or uses `--sdm-account`) and tops
it up to `--sdm-l2-fund-amount` on both chains.

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
