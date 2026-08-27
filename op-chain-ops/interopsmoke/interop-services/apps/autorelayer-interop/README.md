# AutoRelayer Interop

> **WIP — Naive design, not production-ready.**
>
> This relayer is under active development. Several design choices are intentionally
> simplified for the v0 interop testnet and will need revisiting before any
> production deployment:
>
> - **Deposit budget tracking is naive.** Gas consumption is estimated with a
>   fixed constant (`200_000 wei`), not measured from actual receipts.
>   Budget state is persisted to a local JSON file — not a database, not
>   replicated, and lost if the file is deleted.
>
> - **"Who is the sender?" is an open question.** The relayer currently gates on
>   `message.txOrigin` (the EOA that initiated the transaction) for
>   deposit lookups. `txOrigin` is not the right key for deposit attribution
>   in many cases.
>
> - **No retry or circuit breaker logic.** A failed relay is skipped and not
>   retried until the next polling cycle (if ponder still reports it as pending).
>
> - **Single-process, single-threaded.** Messages are processed sequentially
>   within each module.

## Overview

An autorelayer for OP Stack interoperable networks. It polls a [ponder-interop](../ponder-interop/README.md) indexer for pending cross-chain messages and submits relay transactions on destination chains.

The relayer is built around a **module system**:

- **EthBridgeModule** — Relays `SendETH` messages originating from the `SuperchainETHBridge` predeploy. Only relays for users who have deposited ETH to the `MostBasicRelayDeposit` contract (deposit-gated). Tracks per-depositor gas budget.
- **GeneralRelayModule** — Catch-all for L2ToL2CDM messages not handled by a specialized module (e.g., promise/callback cross-chain calls).
- **PromiseModule** — Resolves cross-chain promises by polling `canResolve()` on resolver contracts and calling `resolve()` when ready.
- **CallbackShareModule** — Shares resolved promises to chains whose callbacks are still waiting on them.

Modules are registered in [`src/app.ts`](src/app.ts) and run sequentially on each polling tick.

## Getting Started

```bash
# From monorepo root
pnpm install

# Copy env and configure
cp apps/autorelayer-interop/.env.example apps/autorelayer-interop/.env

# Start the ponder indexer first (required dependency)
pnpm --filter @eth-optimism/ponder-interop dev:v0

# Run the relayer
pnpm --filter autorelayer-interop dev
```

## Configuration

All configuration is via environment variables (or CLI flags). See [`.env.example`](.env.example) for the full reference.

### Required

| Variable                 | Description                           | Default                  |
| ------------------------ | ------------------------------------- | ------------------------ |
| `PONDER_INTEROP_API_URL` | URL of the ponder-interop indexer API | `http://127.0.0.1:42069` |

### Transaction Signing (pick one)

| Variable                 | Description                                                                |
| ------------------------ | -------------------------------------------------------------------------- |
| `SENDER_PRIVATE_KEY`     | Hex-encoded private key for local signing                                  |
| `SPONSORED_ENDPOINT_URL` | URL of a sponsored transaction endpoint (default: `http://127.0.0.1:3000`) |

If `SENDER_PRIVATE_KEY` is set, the relayer signs transactions locally on every chain. Otherwise, it forwards unsigned transactions to `<SPONSORED_ENDPOINT_URL>/<chainId>`. See the [sponsored-sender README](../sponsored-sender/README.md).

### Optional

| Variable                   | Description                                                    | Default |
| -------------------------- | -------------------------------------------------------------- | ------- |
| `LOOP_INTERVAL_MS`         | Polling interval in milliseconds                               | `500`   |
| `CHAIN_ENDPOINT_OVERRIDES` | Comma-separated RPC URLs to override ponder-reported endpoints | —       |
| `GAS_TANK_ADDRESS`         | GasTank contract address for gas accounting                    | —       |
| `PROMISE_CONTRACT_ADDRESS` | Promise contract address for cross-chain promise resolution    | —       |
| `METRICS_ENABLED`          | Expose Prometheus `/metrics` endpoint                          | `false` |
| `METRICS_PORT`             | Port for the metrics HTTP server                               | `7300`  |

## Metrics

The relayer ships a Prometheus-format `/metrics` endpoint (disabled by default).
Set `METRICS_ENABLED=true` (or `--metrics-enabled`) and scrape
`http://<host>:7300/metrics`.

### Top-level

| Metric                                    | Type      | Labels                                                                   |
| ----------------------------------------- | --------- | ------------------------------------------------------------------------ |
| `relayer_cycles_total`                    | counter   | —                                                                        |
| `relayer_cycle_duration_seconds`          | histogram | —                                                                        |
| `relayer_messages_from_indexer`           | gauge     | `src`, `dst`                                                             |
| `relayer_ponder_request_duration_seconds` | histogram | `endpoint`                                                               |
| `relayer_ponder_errors_total`             | counter   | `endpoint`, `kind` (http_4xx \| http_5xx \| parse \| timeout \| network) |
| `relayer_ponder_last_success_timestamp`   | gauge     | `endpoint`                                                               |

### Per-module funnel

Metric names use three nouns: **message** (pre-attempt decisions), **relay_attempt** (the full try), **relay_tx** (the on-chain transaction). See the glossary at the top of `src/metrics.ts` for details.

All per-module metrics labeled with `module`, `src`, `dst`, `relayer_eoa` unless otherwise noted.

| Metric                                            | Type      | Extra labels                                                                              |
| ------------------------------------------------- | --------- | ----------------------------------------------------------------------------------------- |
| `relayer_module_message_backlog`                  | gauge     | —                                                                                         |
| `relayer_module_message_skipped_total`            | counter   | `reason` (no_client \| no_account \| no_deposit \| promise_not_ready \| no_resolver_code) |
| `relayer_module_message_config_error`             | gauge     | `reason`                                                                                  |
| `relayer_module_relay_attempt_failed_total`       | counter   | `stage` (simulation \| broadcast \| execution), `reason`                                  |
| `relayer_module_relay_attempt_duration_seconds`   | histogram | `outcome` (executed \| failed \| broadcast \| skipped)                                    |
| `relayer_module_relay_attempt_retry_total`        | counter   | —                                                                                         |
| `relayer_module_relay_budget_warning_total`       | counter   | eth-bridge only                                                                           |
| `relayer_module_relay_tx_broadcast_total`         | counter   | —                                                                                         |
| `relayer_module_relay_tx_executed_total`          | counter   | —                                                                                         |
| `relayer_module_relay_tx_last_executed_timestamp` | gauge     | —                                                                                         |
| `relayer_module_relay_tx_in_flight`               | gauge     | —                                                                                         |
| `relayer_module_relay_tx_in_flight_stale`         | gauge     | —                                                                                         |

### Journeys

See [`docs/relay-journeys.md`](docs/relay-journeys.md) for a walk-through of
every path a message can take through the funnel, with the exact counter
increments at each step.

### Logs

Failure log lines carry `stage` and `reason` fields that match the metric
labels one-to-one. Pivot from a metric alert to the raw events with a query
like `module="general-relay" AND reason="already_relayed"`. Every broadcast
also logs `tx_hash`, `relayer_eoa`, `target_contract`, and `function_selector`
for easy join against on-chain data.

## Architecture

```
RelayerApp (src/app.ts)
  ├── ClientManager         — per-chain public + wallet clients
  ├── BudgetTracker         — per-depositor gas budget with JSON checkpoint
  ├── RelayerMetrics        — typed prom-client facade (see Metrics)
  ├── EthBridgeModule       — SendETH filtering, deposit gate, budget tracking
  ├── GeneralRelayModule    — catch-all for L2ToL2CDM messages
  ├── PromiseModule         — cross-chain promise resolution
  ├── CallbackShareModule   — shares resolved promises across chains
  └── Relayer               — orchestrator; fetches Ponder lists once per cycle,
                               runs modules in sequence
```

Modules extend [`BaseRelayModule`](src/strategies/baseRelayModule.ts) which owns
session-dedup and retry-tracking primitives and the fire-and-forget receipt
confirmation loop.

### Relay Flow (EthBridgeModule)

1. Fetch pending messages from ponder
2. Cache deposit balances (one lookup per unique `txOrigin` per tick)
3. For each message:
   - Filter: is `sender` the SuperchainETHBridge predeploy?
   - Gate: does `txOrigin` have a deposit balance > 0?
   - Budget: does the depositor have enough remaining budget? (disabled for now)
   - Simulate relay transaction (prevents double-relay)
   - Submit relay transaction
   - Record gas consumption against depositor budget

## Testing

```bash
# Run all tests
pnpm --filter autorelayer-interop vitest --run

# Run a specific test file
pnpm --filter autorelayer-interop vitest --run __tests__/strategies/ethBridgeModule.test.ts
```

Tests are in `__tests__/`, mirroring the `src/` directory structure.
