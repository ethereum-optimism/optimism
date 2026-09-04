/* eslint-disable @typescript-eslint/member-ordering */
import type { SentMessage } from '@eth-optimism/ponder-interop/client'
import { contracts } from '@eth-optimism/viem'
import type { Logger } from 'pino'
import type { Hex, PublicClient } from 'viem'

import type { RelayerMetrics } from '@/metrics.js'
import type {
  RelayFailureRegistry,
  RelayStatus,
} from '@/relay/relayFailureRegistry.js'
import type { ClientManager } from '@/services/clientManager.js'

import type {
  PonderEndpoint,
  RelayModule,
  RunContext,
  RunResult,
} from './types.js'

export interface FunnelLabels {
  module: string
  src: string
  dst: string
  relayer_eoa: string
}

interface SubmittedRelay {
  timestamp: number
  relayerEoa: string
}

/**
 * Common base for pluggable relay modules. Owns shared helpers (logger scoping,
 * metric-label derivation) so concrete modules focus on their own filtering +
 * attempt logic. Subsequent commits lift more behavior (session dedup, funnel
 * counter emission) into this base.
 */
export abstract class BaseRelayModule implements RelayModule {
  /**
   * Broadcast timestamp (ms since epoch) keyed by message hash. Serves two
   * jobs: session dedup (while the entry exists, re-broadcast is suppressed),
   * and the in-flight metric (Date.now() minus the stored timestamp gives
   * age, observed into module_relay_tx_in_flight_age_seconds and counted in
   * module_relay_tx_in_flight).
   * Garbage-collected each cycle: entries whose hash is no longer returned
   * by Ponder as pending have been indexed (RelayedMessage landed on dest)
   * and are dropped. See pruneAndEmitInFlight().
   */
  private readonly submittedAt = new Map<string, SubmittedRelay>()

  /**
   * Durable per-message failure history backing the circuit breaker.
   * Replaces the in-process Set<string> previously held here — see
   * RelayFailureRegistry for the schema. Concrete subclasses pass it through
   * super() from app.ts wiring.
   */
  protected readonly failureRegistry: RelayFailureRegistry

  protected constructor(failureRegistry: RelayFailureRegistry) {
    this.failureRegistry = failureRegistry
  }

  abstract readonly name: string
  abstract readonly needs: readonly PonderEndpoint[]
  abstract run(ctx: RunContext): Promise<RunResult>

  protected moduleLog(ctx: RunContext): Logger {
    return ctx.log.child({ module: this.name })
  }

  /**
   * Whether this module already submitted a relay for `hash` during this
   * process lifetime. Used to suppress re-broadcasts during the window
   * between submission and Ponder indexing the relay event.
   */
  protected hasSubmitted(hash: string): boolean {
    return this.submittedAt.has(hash)
  }

  /**
   * Record a successful submission. Also clears any prior failure state for
   * this hash so subsequent re-emissions from Ponder are treated as
   * dedup-skips, not retries.
   */
  protected recordSubmitted(hash: string, labels?: FunnelLabels): void {
    this.submittedAt.set(hash, {
      timestamp: Date.now(),
      relayerEoa: labels?.relayer_eoa ?? '',
    })
    this.failureRegistry.clearFailure(hash)
  }

  /**
   * Called once per cycle, per module, with the pending messages this module
   * owns (already filtered by sender / type). Performs three jobs:
   *   1. garbage-collects submittedAt entries whose hash is no longer in
   *      pending (Ponder indexed the RelayedMessage on dest);
   *   2. emits module_relay_tx_in_flight (count) and observes each in-flight
   *      tx's age into module_relay_tx_in_flight_age_seconds (histogram), so
   *      the dashboard can derive "stale" at any threshold via PromQL
   *      (e.g. `_count - _bucket{le="30"}`);
   *   3. emits failure-registry observability metrics (size and oldest age
   *      per route × reason) from RelayFailureRegistry.getStats().
   * The relayer resets all of these before the module loop each cycle, so
   * modules only need to .set() / .observe() their own buckets.
   */
  protected pruneAndEmitInFlight(
    ctx: RunContext,
    ownedPending: ReadonlyArray<{
      messageHash: string
      source: number | bigint
      destination: number | bigint
    }>,
  ): void {
    // If the Ponder fetch backing this module's list failed this cycle, the
    // empty/partial pending set is not evidence of anything: skip pruning the
    // session dedup map so a Ponder blip can't trigger re-broadcasts (R12).
    // The shared failure registry is GC'd by the relayer core against the
    // union of all modules' pending keys (see Relayer.gcFailureRegistry) —
    // never here, where only this module's slice is visible (R8).
    const fetchFailed = this.needs.some((n) => ctx.failedEndpoints?.has(n))
    if (!fetchFailed) {
      const pendingHashes = new Set(ownedPending.map((m) => m.messageHash))
      for (const hash of this.submittedAt.keys()) {
        if (!pendingHashes.has(hash)) this.submittedAt.delete(hash)
      }
    } else {
      ctx.log.warn(
        { module: this.name },
        'ponder fetch failed this cycle -- skipping session-dedup prune',
      )
    }

    const byRoute = new Map<
      string,
      {
        src: string
        dst: string
        relayer_eoa: string
        total: number
      }
    >()
    const now = Date.now()
    for (const m of ownedPending) {
      const submitted = this.submittedAt.get(m.messageHash)
      if (submitted === undefined) continue
      const relayerEoa = submitted.relayerEoa
      const key = `${m.source}|${m.destination}|${relayerEoa}`
      const b = byRoute.get(key) ?? {
        src: String(m.source),
        dst: String(m.destination),
        relayer_eoa: relayerEoa,
        total: 0,
      }
      b.total++
      byRoute.set(key, b)

      ctx.metrics.moduleRelayTxInFlightAgeSeconds.observe(
        {
          module: this.name,
          src: String(m.source),
          dst: String(m.destination),
          relayer_eoa: relayerEoa,
        },
        (now - submitted.timestamp) / 1000,
      )
    }

    for (const b of byRoute.values()) {
      ctx.metrics.moduleRelayTxInFlight.set(
        {
          module: this.name,
          src: b.src,
          dst: b.dst,
          relayer_eoa: b.relayer_eoa,
        },
        b.total,
      )
    }

    // Failure-registry observability. Reset by relayer.run() before the
    // module loop, so we only set our own buckets here.
    for (const s of this.failureRegistry.getStats()) {
      const labels = {
        module: this.name,
        src: String(s.source),
        dst: String(s.destination),
        reason: s.reason,
      }
      ctx.metrics.moduleFailureRegistrySize.set(labels, s.count)
      ctx.metrics.moduleFailureRegistryOldestAgeSeconds.set(
        labels,
        Math.max(0, (now - s.oldestLastFailedAt) / 1000),
      )
    }
  }

  /**
   * Whether a prior attempt for `hash` threw. Used for metric labels only —
   * gating the next attempt is shouldAttempt()'s job.
   */
  protected isRetry(hash: string): boolean {
    return this.failureRegistry.hasFailed(hash)
  }

  /**
   * Record a failed attempt. Updates count/timestamp/reason in the failure
   * registry, which may flip the message to permanent (rpc_rejected /
   * expired on first failure, or any reason after MAX_FAILURES). The circuit
   * breaker (shouldAttempt) reads from the same registry.
   */
  protected recordFailure(
    hash: string,
    source: number | bigint,
    destination: number | bigint,
    reason: string,
  ): void {
    this.failureRegistry.recordFailure(
      hash,
      Number(source),
      Number(destination),
      reason,
    )
  }

  /**
   * Classify a hash's relay eligibility — 'ready', 'in_backoff', or
   * 'abandoned'. Call sites use this to attribute skips to the specific cause
   * (vs. the boolean shouldAttempt).
   */
  protected statusFor(hash: string, now: number = Date.now()): RelayStatus {
    return this.failureRegistry.statusFor(hash, now)
  }

  /**
   * Whether the relayer should attempt this message right now. Consults the
   * failure registry — false if permanent or in backoff, true otherwise.
   */
  protected shouldAttempt(hash: string, now: number = Date.now()): boolean {
    return this.failureRegistry.shouldAttempt(hash, now)
  }

  private readonly pendingConfirmations: Array<Promise<void>> = []

  /**
   * Await receipt for a broadcast transaction and emit terminal metrics:
   * - receipt.status === 'success' → executed_total++, duration observed as 'executed'
   * - receipt.status === 'reverted' → failed_total{stage=execution, reason=reverted}, duration observed as 'failed'
   * - receipt wait throws (RPC/timeout) → failed_total{stage=execution, reason=unknown}, duration observed as 'failed'
   *
   * When `dedupKey` is given (the submittedAt key recorded by
   * recordSubmitted), a confirmed revert also releases the session dedup and
   * records a `reverted` failure in the registry — so the message re-enters
   * the normal backoff/retry path instead of being parked until restart
   * (R10). A receipt wait that *throws* does NOT release the dedup: the tx
   * may still land, and re-broadcasting on an RPC blip would double-relay.
   *
   * Fire-and-forget: never rejects. Pending promises are tracked so
   * awaitPendingConfirmations() can drain them for graceful shutdown or tests.
   */
  protected confirmAsync(
    ctx: RunContext,
    publicClient: PublicClient,
    txHash: Hex,
    labels: FunnelLabels,
    attemptStart: number,
    dedupKey?: string,
  ): void {
    const task = (async () => {
      const observe = (outcome: 'executed' | 'failed') => {
        const elapsed = (performance.now() - attemptStart) / 1000
        ctx.metrics.moduleRelayAttemptDurationSeconds.observe(
          { ...labels, outcome },
          elapsed,
        )
      }
      try {
        const receipt = await publicClient.waitForTransactionReceipt({
          hash: txHash,
        })
        if (receipt.status === 'success') {
          ctx.metrics.moduleRelayTxExecutedTotal.inc(labels)
          ctx.metrics.moduleRelayTxLastExecutedTimestamp.set(
            labels,
            Date.now() / 1000,
          )
          observe('executed')
        } else {
          ctx.metrics.moduleRelayAttemptFailedTotal.inc({
            ...labels,
            stage: 'execution',
            reason: 'reverted',
          })
          ctx.log.warn(
            { tx_hash: txHash, ...labels },
            'relay tx reverted during confirmation',
          )
          if (dedupKey !== undefined) {
            // Release the session dedup and record the failure so the message
            // re-enters the backoff/retry path instead of being suppressed by
            // hasSubmitted() until restart (R10).
            this.submittedAt.delete(dedupKey)
            this.recordFailure(
              dedupKey,
              Number(labels.src),
              Number(labels.dst),
              'reverted',
            )
          }
          observe('failed')
        }
      } catch (error) {
        ctx.metrics.moduleRelayAttemptFailedTotal.inc({
          ...labels,
          stage: 'execution',
          reason: 'unknown',
        })
        ctx.log.warn(
          { err: error, tx_hash: txHash, ...labels },
          'relay tx confirmation failed',
        )
        observe('failed')
      }
    })()
    this.pendingConfirmations.push(task)
  }

  /**
   * Observe attempt_duration_seconds at a synchronous terminal site
   * (broadcast/simulation failure). For async confirmations, confirmAsync()
   * handles the observation.
   */
  protected observeAttemptDuration(
    ctx: RunContext,
    labels: FunnelLabels,
    outcome: 'failed',
    attemptStart: number,
  ): void {
    const elapsed = (performance.now() - attemptStart) / 1000
    ctx.metrics.moduleRelayAttemptDurationSeconds.observe(
      { ...labels, outcome },
      elapsed,
    )
  }

  /**
   * Single-pass classification for bridge-owned pending messages:
   * - filters to sender = SuperchainETHBridge, returns the owned slice;
   * - emits module_message_backlog (positive deposit + client present, not
   *   in-flight).
   *
   * Expected failures (no_deposit / budget_blocked / no_gas_price) are
   * counted later via module_message_skipped_total during the attempt loop.
   * In-flight messages are owned (returned) but excluded from backlog —
   * pruneAndEmitInFlight() reports them via module_relay_tx_in_flight.
   */
  protected classifyBridgeMessages(
    pendingMessages: SentMessage[],
    clients: ClientManager,
    depositCache: Map<string, bigint>,
    metrics: RelayerMetrics,
  ): SentMessage[] {
    const bridgeAddr = contracts.superchainETHBridge.address.toLowerCase()
    const ownedPending: SentMessage[] = []
    const backlog = new Map<
      string,
      { src: string; dst: string; count: number }
    >()

    for (const m of pendingMessages) {
      if (m.sender.toLowerCase() !== bridgeAddr) continue
      ownedPending.push(m)
      if (this.hasSubmitted(m.messageHash)) continue

      const hasClient =
        !!clients.getPublicClient(m.destination) &&
        clients.getWalletClients(m.destination).length > 0
      if (!hasClient) continue

      const deposit = depositCache.get(m.txOrigin.toLowerCase()) ?? 0n
      if (deposit <= 0n) continue

      const key = `${m.source}|${m.destination}`
      const b = backlog.get(key) ?? {
        src: String(m.source),
        dst: String(m.destination),
        count: 0,
      }
      b.count++
      backlog.set(key, b)
    }

    for (const b of backlog.values()) {
      metrics.moduleMessageBacklog.set(
        { module: this.name, src: b.src, dst: b.dst, relayer_eoa: '' },
        b.count,
      )
    }
    return ownedPending
  }

  /**
   * Drain all pending fire-and-forget confirmations. Useful for graceful
   * shutdown and for tests that need deterministic metric observations.
   */
  async awaitPendingConfirmations(): Promise<void> {
    await Promise.all(this.pendingConfirmations.splice(0))
  }
}
