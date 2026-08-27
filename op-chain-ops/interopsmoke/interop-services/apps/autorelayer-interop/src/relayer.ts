import type {
  PaginationParams,
  PonderInteropClient,
  SentMessage,
} from '@eth-optimism/ponder-interop/client'
import type { Logger } from 'pino'
import type { Address, Hex } from 'viem'
import { formatEther } from 'viem'

import { classifyPonderError } from '@/errors.js'
import type { RelayFailureRegistry } from '@/relay/relayFailureRegistry.js'
import {
  relayViaGasTank,
  relayViaL2ToL2CrossDomainMessenger,
} from '@/relay/txSubmitter.js'
import type { ClientManager } from '@/services/clientManager.js'
import type {
  PonderEndpoint,
  RelayMessageParams,
  RelayModule,
  RunContext,
} from '@/strategies/types.js'

/**
 * Pagination defaults. All three are overridable via RelayerConfig.pagination
 * (wired from PONDER_PAGE_LIMIT / PONDER_MAX_PER_CYCLE / PONDER_MAX_SCAN_PER_CYCLE
 * env vars in app.ts).
 *
 * pageLimit: rows per request; 10000 matches ponder-interop's server-side
 * MAX_LIMIT — asking for more is silently capped there.
 * maxAttemptablePerCycle: per-cycle budget of ATTEMPTABLE rows (rows not
 * already abandoned in the failure registry); bounds per-cycle work while
 * a large backlog drains over multiple cycles.
 * maxScanPerCycle: absolute bound on rows fetched per cycle, counting
 * abandoned rows too; bounds the cost of paging past a head of stuck
 * messages (see fetchPaginatedList).
 */
const DEFAULT_PAGE_LIMIT = 10000
const DEFAULT_MAX_ATTEMPTABLE_PER_CYCLE = 10000
const DEFAULT_MAX_SCAN_PER_CYCLE = 50000

export interface PaginationConfig {
  pageLimit?: number
  maxAttemptablePerCycle?: number
  maxScanPerCycle?: number
}

/**
 * A fully- or partially-fetched paginated list. `complete` is true only when
 * the server signalled end-of-stream (empty page) — i.e. we saw the entire
 * pending set. When false (budget or scan cap hit first), absence of a key
 * from `items` says nothing, so failure-registry GC must be skipped.
 */
interface PaginatedResult<T> {
  items: T[]
  complete: boolean
}

export interface RelayerConfig {
  ponderInteropApi: string
  ponderClient: PonderInteropClient
  clients: ClientManager
  pagination?: PaginationConfig
  /**
   * The failure registry shared by all modules. The relayer core owns its
   * garbage collection: rows are retained while their key (message hash or
   * promise id) is pending per ANY fetched list, and GC is skipped entirely
   * on cycles with a failed fetch. Modules only read/write individual
   * entries — see gcFailureRegistry() for why per-module GC was unsound (R8).
   */
  failureRegistry: RelayFailureRegistry
  gasTankAddress?: Address
}

export class Relayer {
  protected readonly log: Logger
  private readonly config: RelayerConfig
  private readonly modules: RelayModule[]
  // Union of the Ponder list endpoints across enabled modules. Lists not
  // needed by any module are skipped each cycle.
  private readonly neededEndpoints: ReadonlySet<PonderEndpoint>
  private readonly pageLimit: number
  private readonly maxAttemptablePerCycle: number
  private readonly maxScanPerCycle: number
  private ctx: RunContext | undefined

  constructor(log: Logger, config: RelayerConfig, modules: RelayModule[]) {
    this.config = config
    this.modules = modules
    this.log = log.child({ module: 'relayer' })
    this.neededEndpoints = new Set(modules.flatMap((m) => m.needs))
    this.pageLimit = config.pagination?.pageLimit ?? DEFAULT_PAGE_LIMIT
    this.maxAttemptablePerCycle =
      config.pagination?.maxAttemptablePerCycle ??
      DEFAULT_MAX_ATTEMPTABLE_PER_CYCLE
    this.maxScanPerCycle =
      config.pagination?.maxScanPerCycle ?? DEFAULT_MAX_SCAN_PER_CYCLE
  }

  /**
   * Sets the RunContext after construction. Needed because RunContext.relayMessage
   * binds back to this Relayer instance (circular dependency resolved via two-phase init).
   */
  setContext(ctx: RunContext): void {
    this.ctx = ctx
  }

  /**
   * Fetches the Ponder lists needed by enabled modules once per cycle, then
   * iterates registered modules and calls run() on each with the resolved
   * lists passed via RunContext. Only endpoints in `neededEndpoints` are
   * fetched; the rest stay empty. Fetch failures degrade gracefully: the
   * affected list is empty for this cycle and modules proceed.
   */
  async run(): Promise<void> {
    if (!this.ctx) {
      throw new Error('RunContext not set -- call setContext() before run()')
    }

    const metrics = this.ctx.metrics
    const cycleStart = performance.now()
    metrics.cyclesTotal.inc()

    // Hashes already marked permanent in the failure registry. They sit at
    // the head of Ponder's oldest-first /messages/pending ordering until they
    // expire server-side; without excluding them from the per-cycle budget,
    // ≥ maxAttemptablePerCycle stuck messages starve everything behind them
    // out of the fetch window entirely (R11 — the "permastuck" mechanism).
    const abandonedKeys = new Set(
      this.config.failureRegistry.getPermanent().map((p) => p.messageHash),
    )

    // pendingMessages is the baseline list (eth-bridge / general-relay) and is
    // always fetched. The promise-specific lists are gated on `neededEndpoints`
    // so an eth-bridge-only (or general-relay-only) deployment never calls the
    // /promises/* endpoints.
    const [messagesResult, promisesResult, unsharedResult] = await Promise.all([
      this.fetchPaginatedList(
        (params) => this.config.ponderClient.getPendingMessages(params),
        'pending_messages',
        (m) => !m.messageHash || !abandonedKeys.has(m.messageHash.toLowerCase()),
      ),
      this.maybePaginatedFetch(
        'pendingPromises',
        (params) => this.config.ponderClient.getPendingPromises(params),
        'pending_promises',
      ),
      this.maybeFetch(
        'unsharedResolvedPromises',
        () => this.config.ponderClient.getUnsharedResolvedPromises(),
        'unshared_resolved_promises',
      ),
    ])

    // A failed fetch (null) degrades to an empty list for module iteration,
    // but the endpoint is flagged so consumers of that list never mistake
    // "Ponder was down" for "nothing is pending" — GC of durable failure
    // state and session dedup must be skipped for flagged endpoints (R12).
    const failedEndpoints = new Set<PonderEndpoint>()
    if (messagesResult === null) failedEndpoints.add('pendingMessages')
    if (promisesResult === null) failedEndpoints.add('pendingPromises')
    if (unsharedResult === null) failedEndpoints.add('unsharedResolvedPromises')
    const pendingMessages = messagesResult?.items ?? []
    const pendingPromises = promisesResult?.items ?? []
    const unsharedResolvedPromises = unsharedResult ?? []

    this.updateMessagesPendingGauge(pendingMessages)
    this.updateUnownedGauge(pendingMessages)

    // GC additionally requires a COMPLETE view of the pending sets: when a
    // list was truncated at the budget/scan cap, a key's absence from the
    // window says nothing about whether it is still pending (R11's second
    // mechanism: backlog beyond the window eroding registry rows).
    const sawCompletePendingSets =
      (messagesResult?.complete ?? false) && (promisesResult?.complete ?? false)
    this.gcFailureRegistry(failedEndpoints, sawCompletePendingSets, {
      pendingMessages,
      pendingPromises,
      unsharedResolvedPromises,
    })

    // Clear per-cycle module gauges; modules re-set their own buckets during
    // run(). Without the reset, a (src,dst) bucket that goes to zero would
    // stick at its previous value forever.
    metrics.moduleMessageBacklog.reset()
    metrics.moduleRelayTxInFlight.reset()
    metrics.moduleRelayTxInFlightAgeSeconds.reset()
    metrics.moduleFailureRegistrySize.reset()
    metrics.moduleFailureRegistryOldestAgeSeconds.reset()

    const cycleCtx: RunContext = {
      ...this.ctx,
      pendingMessages,
      pendingPromises,
      unsharedResolvedPromises,
      failedEndpoints,
    }

    for (const mod of this.modules) {
      try {
        const result = await mod.run(cycleCtx)
        this.log.info({ module: mod.name, ...result }, 'module completed')
      } catch (error) {
        this.log.error({ err: error, module: mod.name }, 'module run failed')
      }
    }

    // Per-cycle EOA balance snapshot. Independent of module success: even if
    // a module crashes, balances still surface so funding alerts work.
    await this.updateEoaBalances()

    metrics.cycleDurationSeconds.observe(
      (performance.now() - cycleStart) / 1000,
    )
  }

  /**
   * Garbage-collects the shared failure registry against the union of every
   * pending key across all fetched lists — message hashes and promise ids
   * live in the same registry keyspace. Centralized here because per-module
   * GC against a module's own pending set deleted every OTHER module's rows
   * each cycle (failure counts reset to 1, backoff stuck at its 5s floor,
   * MAX_FAILURES unreachable — R8).
   *
   * Skipped whenever any fetch failed this cycle (an empty-because-Ponder-
   * was-down list must not be read as "nothing is pending", R12) and
   * whenever a pending list was truncated at the fetch caps (a key beyond
   * the window is still live even though we didn't see it, R11).
   */
  private gcFailureRegistry(
    failedEndpoints: ReadonlySet<PonderEndpoint>,
    sawCompletePendingSets: boolean,
    lists: {
      pendingMessages: ReadonlyArray<{ messageHash: string }>
      pendingPromises: ReadonlyArray<{ promiseId: string }>
      unsharedResolvedPromises: ReadonlyArray<{ promiseId: string }>
    },
  ): void {
    if (failedEndpoints.size > 0) {
      this.log.warn(
        { failedEndpoints: [...failedEndpoints] },
        'skipping failure-registry GC: ponder fetch failed this cycle',
      )
      return
    }
    if (!sawCompletePendingSets) {
      this.log.info(
        'skipping failure-registry GC: pending window truncated at fetch caps',
      )
      return
    }

    const liveKeys = new Set<string>()
    for (const m of lists.pendingMessages) {
      if (m.messageHash) liveKeys.add(m.messageHash)
    }
    for (const p of lists.pendingPromises) {
      if (p.promiseId) liveKeys.add(p.promiseId)
    }
    for (const u of lists.unsharedResolvedPromises) {
      if (u.promiseId) liveKeys.add(u.promiseId)
    }

    const removed = this.config.failureRegistry.gc(liveKeys)
    if (removed > 0) {
      this.log.debug(
        { removed, live: liveKeys.size },
        'failure-registry GC removed rows no longer pending',
      )
    }
  }

  /**
   * Reads native balance for every (EOA, chain) pair the relayer holds keys
   * for and emits `relayer_eoa_balance_eth`. Individual RPC failures are
   * logged and skipped — a dead RPC on one chain shouldn't suppress others.
   * Reset+repopulate each cycle so EOAs that disappear (config change) don't
   * leave stale buckets.
   */
  private async updateEoaBalances(): Promise<void> {
    if (!this.ctx) return
    const metrics = this.ctx.metrics
    const eoas = this.config.clients.listSigningEoas()
    metrics.relayerEoaBalanceEth.reset()
    await Promise.all(
      eoas.map(async ({ address, chainId }) => {
        const publicClient = this.config.clients.getPublicClient(chainId)
        if (!publicClient) return
        try {
          const wei = await publicClient.getBalance({ address })
          metrics.relayerEoaBalanceEth.set(
            {
              relayer_eoa: address.toLowerCase(),
              chain_id: String(chainId),
            },
            parseFloat(formatEther(wei)),
          )
        } catch (error) {
          this.log.warn(
            { err: error, address, chainId },
            'failed to read EOA balance',
          )
        }
      }),
    )
  }

  /**
   * Counts pending messages that no enabled module owns into
   * relayer_messages_unowned. With a catch-all module (general-relay)
   * enabled, everything is owned and the gauge stays zero. Without one,
   * a message whose sender is outside every module's ownedSenders will
   * never be attempted — that's a configuration gap worth alerting on,
   * not just an entry in a per-module info log (R1).
   */
  private updateUnownedGauge(messages: SentMessage[]): void {
    if (!this.ctx) return
    const gauge = this.ctx.metrics.messagesUnowned
    gauge.reset()
    if (this.modules.some((m) => m.catchAll)) return

    const claimedSenders = new Set(
      this.modules
        .flatMap((m) => m.ownedSenders ?? [])
        .map((addr) => addr.toLowerCase()),
    )
    const counts = new Map<string, number>()
    for (const msg of messages) {
      if (msg.sender && claimedSenders.has(msg.sender.toLowerCase())) continue
      const key = `${msg.source}|${msg.destination}`
      counts.set(key, (counts.get(key) ?? 0) + 1)
    }
    for (const [key, count] of counts) {
      const [src, dst] = key.split('|')
      gauge.set({ src, dst }, count)
    }
  }

  private updateMessagesPendingGauge(messages: SentMessage[]): void {
    if (!this.ctx) return
    const gauge = this.ctx.metrics.messagesFromIndexer
    gauge.reset()
    const counts = new Map<string, number>()
    for (const msg of messages) {
      const key = `${msg.source}|${msg.destination}`
      counts.set(key, (counts.get(key) ?? 0) + 1)
    }
    for (const [key, count] of counts) {
      const [src, dst] = key.split('|')
      gauge.set({ src, dst }, count)
    }
  }

  /**
   * Paginated fetch gated on whether any enabled module needs this endpoint.
   * Returns an empty complete list (no network call) when unneeded; `null`
   * when the fetch failed.
   */
  private async maybePaginatedFetch<T>(
    need: PonderEndpoint,
    fn: (params: PaginationParams) => Promise<T[]>,
    endpoint: string,
  ): Promise<PaginatedResult<T> | null> {
    if (!this.neededEndpoints.has(need)) return { items: [], complete: true }
    return this.fetchPaginatedList(fn, endpoint)
  }

  /**
   * Single (non-paginated) fetch gated on `neededEndpoints`. Returns an empty
   * list (no network call) when unneeded; `null` when the fetch failed.
   */
  private async maybeFetch<T>(
    need: PonderEndpoint,
    fn: () => Promise<T[]>,
    endpoint: string,
  ): Promise<T[] | null> {
    if (!this.neededEndpoints.has(need)) return []
    return this.fetchList(fn, endpoint)
  }

  /**
   * Fetches a list page-by-page until the server signals end-of-stream with
   * an empty page (complete: true), or until `maxAttemptablePerCycle` rows
   * counted by `countsTowardBudget` have been collected or `maxScanPerCycle`
   * total rows have been fetched (both complete: false).
   *
   * `countsTowardBudget` exists to defeat head-of-line starvation (R11):
   * /messages/pending is served oldest-first, and permanently-failed
   * (abandoned) messages pile up at the head until server-side expiry
   * flushes them. They must be fetched (GC and metrics need to see them)
   * but must NOT consume the per-cycle work budget — otherwise a head of
   * ≥ budget stuck messages permanently hides every fresh message behind
   * it. Rows the predicate rejects still land in `items`; modules skip
   * them cheaply via the failure-registry gate. `maxScanPerCycle` bounds
   * the total paging cost.
   *
   * Returns `null` if any page fetch fails — a partially-fetched list is
   * treated the same as no list, because consumers can't tell which rows
   * are missing.
   */
  private async fetchPaginatedList<T>(
    fn: (params: PaginationParams) => Promise<T[]>,
    endpoint: string,
    countsTowardBudget?: (item: T) => boolean,
  ): Promise<PaginatedResult<T> | null> {
    const all: T[] = []
    let offset = 0
    let budgetUsed = 0

    while (
      budgetUsed < this.maxAttemptablePerCycle &&
      all.length < this.maxScanPerCycle
    ) {
      const limit = Math.min(this.pageLimit, this.maxScanPerCycle - all.length)
      const page = await this.fetchList(() => fn({ limit, offset }), endpoint)
      if (page === null) return null
      // End-of-stream is signalled by an empty page, not by a short page:
      // the server may silently cap `limit` (Ponder caps at MAX_LIMIT), so
      // a page smaller than what we requested does NOT mean we're done.
      if (page.length === 0) return { items: all, complete: true }
      for (const item of page) {
        if (countsTowardBudget === undefined || countsTowardBudget(item)) {
          budgetUsed++
        }
      }
      all.push(...page)
      offset += page.length
    }

    return { items: all, complete: false }
  }

  /**
   * Single fetch with metrics + logging. Returns `null` on failure — callers
   * degrade to an empty list but must flag the endpoint as failed so GC of
   * durable state is skipped (see run()).
   */
  private async fetchList<T>(
    fn: () => Promise<T[]>,
    endpoint: string,
  ): Promise<T[] | null> {
    const metrics = this.ctx?.metrics
    const start = performance.now()
    try {
      const result = await fn()
      const durationSec = (performance.now() - start) / 1000
      if (metrics) {
        metrics.ponderRequestDurationSeconds.observe({ endpoint }, durationSec)
        metrics.ponderLastSuccessTimestamp.set({ endpoint }, Date.now() / 1000)
      }
      return result
    } catch (error) {
      const durationSec = (performance.now() - start) / 1000
      if (metrics) {
        metrics.ponderRequestDurationSeconds.observe({ endpoint }, durationSec)
        metrics.ponderErrorsTotal.inc({
          endpoint,
          kind: classifyPonderError(error),
        })
      }
      this.log.error(
        { err: error, endpoint },
        'ponder fetch failed, using empty list for this cycle',
      )
      return null
    }
  }

  /**
   * Submission entry point exposed to RunContext callback.
   * Dispatches to L2ToL2CDM or GasTank based on gasTankAddress config.
   */
  async submitRelayMessage(params: RelayMessageParams): Promise<Hex> {
    const { destinationChainId } = params

    const publicClient = this.config.clients.getPublicClient(destinationChainId)
    const walletClient = params.walletClient

    if (!publicClient) {
      throw new Error(`no client for chain ${destinationChainId}`)
    }

    const msgLog = this.log.child({
      chainId: destinationChainId,
    })

    msgLog.info(
      { gasTankAddress: this.config.gasTankAddress ?? null },
      'submitRelayMessage: dispatching',
    )

    if (this.config.gasTankAddress) {
      msgLog.info('routing via GasTank')
      return relayViaGasTank({
        publicClient,
        walletClient,
        gasTankAddress: this.config.gasTankAddress,
        params,
        msgLog,
      })
    }

    msgLog.info('routing via L2ToL2CDM')
    return relayViaL2ToL2CrossDomainMessenger({
      publicClient,
      walletClient,
      params,
      msgLog,
    })
  }
}
