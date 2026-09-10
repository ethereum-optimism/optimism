import { Counter, Gauge, Histogram, type Registry } from 'prom-client'

/*
============================================================================
Metrics taxonomy
============================================================================

Metric names use a noun-first convention. There are three nouns that
describe the lifecycle of a cross-chain message in this service:

  message       — an item surfaced by the indexer (Ponder). The relayer
                  decides what to do with each one before any on-chain work
                  starts. Metrics on this noun capture those decisions:
                  what's in the backlog (would relay right now if we
                  could), what's been skipped and why (pre-attempt gating:
                  no_deposit, promise_not_ready), what's blocked by
                  configuration gaps (no wallet for dest).

  relay_attempt — the full work of trying to relay one message, from the
                  point the module commits to trying through a terminal
                  outcome. Spans pre-flight checks, simulation, broadcast,
                  and receipt wait. An attempt may never produce a tx
                  (e.g., simulation fails). Metrics on this noun capture:
                  attempts that failed (by stage and reason), attempt
                  duration (by outcome), and attempts retried after a
                  prior failure.

  relay_tx      — the actual on-chain Ethereum transaction we submitted.
                  Only exists if broadcast succeeded. Metrics on this noun
                  capture: txs broadcast, txs executed (mined with
                  status=success), txs still in-flight awaiting indexer
                  confirmation, and stale in-flight txs (signals reverts
                  or indexer lag).

Lifecycle diagram:

  message --(skipped)--------------------------> [terminal, no attempt]
  message --(attempt starts)----> relay_attempt
                                    ├──(sim fail)----------> [terminal, no tx]
                                    └──(broadcast) --> relay_tx
                                                          ├── executed
                                                          └── reverted

Reading a metric name:

  relayer_module_<noun>_<verb|state>[_total|_seconds]
    - relayer_module_message_backlog            — messages in backlog
    - relayer_module_relay_tx_broadcast_total   — count of txs broadcast
    - relayer_module_relay_attempt_failed_total — attempts that failed

Top-level (not per-module) metrics:

  relayer_build_info{version,sha,dirty,node} — build metadata, value=1
  relayer_messages_from_indexer{src,dst} — raw pending count from indexer
  relayer_cycles_total                   — relayer loop iterations
  relayer_cycle_duration_seconds         — loop iteration duration
  relayer_ponder_*                       — indexer health
============================================================================
*/

/**
 * Typed facade over prom-client metrics. Modules and the relayer call typed
 * `.inc()` / `.set()` / `.observe()` methods here rather than touching the
 * underlying registry directly — which keeps metric names, label shapes, and
 * help text defined in one place. See the taxonomy block comment above for
 * the three-noun mental model (message / relay_attempt / relay_tx).
 */
export class RelayerMetrics {
  // Build metadata (constant — set once at boot)
  readonly buildInfo: Gauge<'version' | 'sha' | 'dirty' | 'node'>

  // Top-level (cycle-scoped)
  readonly cyclesTotal: Counter<string>
  readonly cycleDurationSeconds: Histogram<string>
  readonly messagesFromIndexer: Gauge<'src' | 'dst'>
  readonly messagesUnowned: Gauge<'src' | 'dst'>

  // Indexer (Ponder) health
  readonly ponderRequestDurationSeconds: Histogram<'endpoint'>
  readonly ponderErrorsTotal: Counter<'endpoint' | 'kind'>
  readonly ponderLastSuccessTimestamp: Gauge<'endpoint'>

  // Relayer-side resources — per-EOA × per-chain native balance, snapshot each cycle
  readonly relayerEoaBalanceEth: Gauge<'relayer_eoa' | 'chain_id'>

  // Per-module: message-stage metrics (pre-attempt decisions)
  readonly moduleMessageBacklog: Gauge<'module' | 'src' | 'dst' | 'relayer_eoa'>
  readonly moduleMessageSkippedTotal: Counter<
    'module' | 'src' | 'dst' | 'relayer_eoa' | 'reason'
  >

  // Per-module: failure-registry observability (current state of the
  // permanent-failure table — what's in the penalty box right now)
  readonly moduleFailureRegistrySize: Gauge<
    'module' | 'src' | 'dst' | 'reason'
  >
  readonly moduleFailureRegistryOldestAgeSeconds: Gauge<
    'module' | 'src' | 'dst' | 'reason'
  >

  // Per-module: relay_attempt-stage metrics (the full try, tx or no tx)
  readonly moduleRelayAttemptFailedTotal: Counter<
    'module' | 'src' | 'dst' | 'relayer_eoa' | 'stage' | 'reason'
  >
  readonly moduleRelayAttemptDurationSeconds: Histogram<
    'module' | 'src' | 'dst' | 'relayer_eoa' | 'outcome'
  >
  readonly moduleRelayAttemptRetryTotal: Counter<
    'module' | 'src' | 'dst' | 'relayer_eoa'
  >

  // Per-module: relay_tx-stage metrics (the on-chain transaction)
  readonly moduleRelayTxBroadcastTotal: Counter<
    'module' | 'src' | 'dst' | 'relayer_eoa'
  >
  readonly moduleRelayTxExecutedTotal: Counter<
    'module' | 'src' | 'dst' | 'relayer_eoa'
  >
  readonly moduleRelayTxLastExecutedTimestamp: Gauge<
    'module' | 'src' | 'dst' | 'relayer_eoa'
  >
  readonly moduleRelayTxInFlight: Gauge<
    'module' | 'src' | 'dst' | 'relayer_eoa'
  >
  readonly moduleRelayTxInFlightAgeSeconds: Histogram<
    'module' | 'src' | 'dst' | 'relayer_eoa'
  >

  constructor(registry: Registry) {
    const registers = [registry]

    this.buildInfo = new Gauge({
      name: 'relayer_build_info',
      help: "Build metadata for the running relayer. Value is always 1; info lives in labels: version (package.json), sha (git rev-parse HEAD), dirty ('true'|'false'|'unknown'), node (process.version). Set once at boot by the app wiring.",
      labelNames: ['version', 'sha', 'dirty', 'node'],
      registers,
    })

    this.cyclesTotal = new Counter({
      name: 'relayer_cycles_total',
      help: 'Polling cycles executed',
      registers,
    })

    this.cycleDurationSeconds = new Histogram({
      name: 'relayer_cycle_duration_seconds',
      help: 'Duration of a full polling cycle, in seconds',
      buckets: [0.1, 0.5, 1, 2, 5, 10, 30, 60],
      registers,
    })

    this.messagesFromIndexer = new Gauge({
      name: 'relayer_messages_from_indexer',
      help: 'Pending L2ToL2CDM messages returned by the indexer (Ponder) this cycle, partitioned by source/destination chain. Raw upstream count; may include messages no module owns or messages in expected-failure states (e.g., no_deposit).',
      labelNames: ['src', 'dst'],
      registers,
    })

    this.messagesUnowned = new Gauge({
      name: 'relayer_messages_unowned',
      help: "Pending messages that NO enabled module owns this cycle — they will never be attempted under the current ENABLED_MODULES configuration. Always zero when a catch-all module (general-relay) is enabled. A non-zero value is a configuration alert: pending work exists that this deployment is not configured to relay (the '6/08 stalled fresh install' class of incident).",
      labelNames: ['src', 'dst'],
      registers,
    })

    this.ponderRequestDurationSeconds = new Histogram({
      name: 'relayer_ponder_request_duration_seconds',
      help: 'Duration of Ponder HTTP requests, in seconds',
      labelNames: ['endpoint'],
      buckets: [0.05, 0.1, 0.25, 0.5, 1, 2, 5],
      registers,
    })

    this.ponderErrorsTotal = new Counter({
      name: 'relayer_ponder_errors_total',
      help: 'Ponder request errors by endpoint and kind',
      labelNames: ['endpoint', 'kind'],
      registers,
    })

    this.ponderLastSuccessTimestamp = new Gauge({
      name: 'relayer_ponder_last_success_timestamp',
      help: 'Unix timestamp (seconds) of the last successful Ponder request, by endpoint',
      labelNames: ['endpoint'],
      registers,
    })

    this.relayerEoaBalanceEth = new Gauge({
      name: 'relayer_eoa_balance_eth',
      help: 'Per-EOA × per-chain native balance in ETH (string-formatted from wei via viem formatEther, then parsed to float for Prometheus). Snapshot read once per cycle; depletion vs top-up cadence drives funding alerts.',
      labelNames: ['relayer_eoa', 'chain_id'],
      registers,
    })

    this.moduleMessageBacklog = new Gauge({
      name: 'relayer_module_message_backlog',
      help: 'Messages this module believes it could successfully relay right now (passes module-local gating: has client, has deposit where required, not already in-flight). This is the stall signal -- combine with `time() - relayer_module_relay_tx_last_executed_timestamp` to detect "ready work not draining".',
      labelNames: ['module', 'src', 'dst', 'relayer_eoa'],
      registers,
    })

    this.moduleMessageSkippedTotal = new Counter({
      name: 'relayer_module_message_skipped_total',
      help: 'Messages this module declined to attempt, by reason (no_client, no_account, no_deposit, promise_not_ready, no_resolver_code). In-flight messages (broadcast but not yet indexed) are tracked separately via relayer_module_relay_tx_in_flight and are NOT counted here.',
      labelNames: ['module', 'src', 'dst', 'relayer_eoa', 'reason'],
      registers,
    })

    this.moduleFailureRegistrySize = new Gauge({
      name: 'relayer_module_failure_registry_size',
      help: 'Current count of entries in the relay failure registry that are permanently flagged as unrelayable (rpc_rejected, expired, or transient failures that hit MAX_FAILURES), bucketed by route × reason. Snapshot of "what is in the penalty box right now" — climbing buckets are alert-worthy.',
      labelNames: ['module', 'src', 'dst', 'reason'],
      registers,
    })

    this.moduleFailureRegistryOldestAgeSeconds = new Gauge({
      name: 'relayer_module_failure_registry_oldest_age_seconds',
      help: 'For each route × reason bucket in the failure registry, the age in seconds of the oldest entry (now - min(last_failed_at)). Pairs with relayer_module_failure_registry_size to answer "how long has this been stuck?"',
      labelNames: ['module', 'src', 'dst', 'reason'],
      registers,
    })

    this.moduleRelayAttemptFailedTotal = new Counter({
      name: 'relayer_module_relay_attempt_failed_total',
      help: 'Relay attempts that failed, by stage (simulation|broadcast|execution) and reason. An attempt may or may not have produced an on-chain tx (simulation failures produce no tx).',
      labelNames: ['module', 'src', 'dst', 'relayer_eoa', 'stage', 'reason'],
      registers,
    })

    this.moduleRelayAttemptDurationSeconds = new Histogram({
      name: 'relayer_module_relay_attempt_duration_seconds',
      help: 'Duration of a relay attempt from commit-to-try through terminal outcome, in seconds. outcome is one of: executed, failed, broadcast, skipped.',
      labelNames: ['module', 'src', 'dst', 'relayer_eoa', 'outcome'],
      buckets: [0.05, 0.1, 0.5, 1, 2, 5, 10, 30],
      registers,
    })

    this.moduleRelayAttemptRetryTotal = new Counter({
      name: 'relayer_module_relay_attempt_retry_total',
      help: 'Relay attempts on messages that failed in a prior attempt this process lifetime. Useful as a load signal: ratio over total attempts tells you how much work is repeat-failure churn.',
      labelNames: ['module', 'src', 'dst', 'relayer_eoa'],
      registers,
    })

    this.moduleRelayTxBroadcastTotal = new Counter({
      name: 'relayer_module_relay_tx_broadcast_total',
      help: 'Relay transactions broadcast (accepted by the node, tx hash returned). Does not guarantee on-chain execution — pair with relayer_module_relay_tx_executed_total.',
      labelNames: ['module', 'src', 'dst', 'relayer_eoa'],
      registers,
    })

    this.moduleRelayTxExecutedTotal = new Counter({
      name: 'relayer_module_relay_tx_executed_total',
      help: 'Relay transactions mined with status=success. Gap vs relayer_module_relay_tx_broadcast_total = reverts or unconfirmed.',
      labelNames: ['module', 'src', 'dst', 'relayer_eoa'],
      registers,
    })

    this.moduleRelayTxLastExecutedTimestamp = new Gauge({
      name: 'relayer_module_relay_tx_last_executed_timestamp',
      help: 'Unix timestamp (seconds) of the last successful relay tx for this module+route. Query as `time() - metric` for staleness; combine with relayer_module_message_backlog for stall detection.',
      labelNames: ['module', 'src', 'dst', 'relayer_eoa'],
      registers,
    })

    this.moduleRelayTxInFlight = new Gauge({
      name: 'relayer_module_relay_tx_in_flight',
      help: 'Relay txs this module has broadcast but the indexer has not yet reported a RelayedMessage for. Garbage-collected each cycle: once the indexer stops returning a message as pending, it is removed.',
      labelNames: ['module', 'src', 'dst', 'relayer_eoa'],
      registers,
    })

    this.moduleRelayTxInFlightAgeSeconds = new Histogram({
      name: 'relayer_module_relay_tx_in_flight_age_seconds',
      help: 'Snapshot distribution of in-flight relay tx ages (now - broadcast_timestamp), re-observed each cycle and reset between cycles. Lets the dashboard derive "stale" at any threshold in PromQL (e.g. _count - _bucket{le="30"} for >30s) instead of baking the threshold into emission code.',
      labelNames: ['module', 'src', 'dst', 'relayer_eoa'],
      buckets: [5, 10, 30, 60, 120, 300, 600],
      registers,
    })
  }
}
