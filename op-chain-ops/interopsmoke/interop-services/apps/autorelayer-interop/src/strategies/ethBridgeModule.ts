import type { SentMessage } from '@eth-optimism/ponder-interop/client'
import { contracts } from '@eth-optimism/viem'
import type { MessageIdentifier } from '@eth-optimism/viem/types/interop'
import { encodeAccessList } from '@eth-optimism/viem/utils/interop'
import type { Logger } from 'pino'
import type { Account, Address, WalletClient } from 'viem'

import type { RelayFundsDeposited } from '@/deposit/relayFundsDeposited.js'
import { RelayError } from '@/errors.js'
import type { RelayFailureRegistry } from '@/relay/relayFailureRegistry.js'

import { BaseRelayModule } from './baseRelayModule.js'
import type {
  PonderEndpoint,
  RelayMessageParams,
  RunContext,
  RunResult,
} from './types.js'

/**
 * Default estimated gas cost per relay: 200_000 gas units
 */
const DEFAULT_ESTIMATED_GAS_COST = 200_000n
const DEPOSIT_LOOKUP_CONCURRENCY = 32

type ReadyRelay = Omit<RelayMessageParams, 'account' | 'walletClient'> & {
  attemptStart: number
  retry: boolean
  gasPrice: bigint
}

interface ReadyWallet {
  account: Account
  walletClient: WalletClient
}

type ReadyRelaysByChain = Map<number, ReadyRelay[]>
type ReadyWalletsByChain = Map<number, ReadyWallet[]>

export class EthBridgeModule extends BaseRelayModule {
  readonly name = 'eth-bridge'
  readonly needs = ['pendingMessages'] as const satisfies readonly PonderEndpoint[]
  // This module owns SuperchainETHBridge pending messages; declaring it here
  // lets GeneralRelayModule treat them as claimed and skip them.
  readonly ownedSenders: readonly Address[] = [
    contracts.superchainETHBridge.address,
  ]
  private readonly relayFunds: RelayFundsDeposited
  private readonly estimatedGasCost: bigint

  constructor(
    relayFunds: RelayFundsDeposited,
    failureRegistry: RelayFailureRegistry,
    estimatedGasCost: bigint = DEFAULT_ESTIMATED_GAS_COST,
  ) {
    super(failureRegistry)
    this.relayFunds = relayFunds
    this.estimatedGasCost = estimatedGasCost
  }

  async run(ctx: RunContext): Promise<RunResult> {
    const log = this.moduleLog(ctx)
    const result: RunResult = { relayed: 0, skipped: 0, failed: 0, noMatch: 0 }

    const pendingMessages = ctx.pendingMessages
    log.info(`${pendingMessages.length} pending messages`)

    // Cache lookups need to run against the bridge-owned slice, but
    // classifyBridgeMessages also emits backlog/config-error gauges that
    // depend on depositCache. Resolve in two phases: pre-filter for the
    // caches, then classify once both caches are populated.
    const bridgePending = pendingMessages.filter(
      (msg) =>
        msg.sender.toLowerCase() ===
        contracts.superchainETHBridge.address.toLowerCase(),
    )
    const depositCache = await this.cacheDepositBalances(
      ctx,
      bridgePending,
      log,
    )
    const gasPriceCache = await this.cacheGasPrices(ctx, bridgePending, log)
    const readyWalletsByChain = new Map<number, ReadyWallet[]>()

    const ownedPending = this.classifyBridgeMessages(
      pendingMessages,
      ctx.clients,
      depositCache,
      ctx.metrics,
    )
    this.pruneAndEmitInFlight(ctx, ownedPending)

    const readyByChain = await this.collectReadyRelays(
      ctx,
      pendingMessages,
      depositCache,
      gasPriceCache,
      readyWalletsByChain,
      result,
      log,
    )

    await this.broadcastRelays(ctx, readyByChain, readyWalletsByChain, result)

    return result
  }

  /**
   * Validates a pending message against deposit balance and budget.
   * Compares the deposit balance (wei) minus any same-run reservations
   * against estimatedGasUnits × gasPrice (wei). The `reservedThisRun`
   * argument is the per-run shadow ledger of cost already promised to other
   * messages from the same tx.origin in this run — without it, two messages
   * sharing a tx.origin would both pass against a frozen consumed value
   * (TOCTOU). On insufficient budget, persists the message to the
   * budget_blocked table and throws — the caller skips the relay; the row
   * is cleared on a future successful relay.
   */
  protected async validate(
    message: SentMessage,
    depositBalance: bigint,
    gasPrice: bigint,
    reservedThisRun: bigint = 0n,
  ): Promise<void> {
    const origin = message.txOrigin.toLowerCase()
    if (depositBalance <= 0n) {
      throw new Error(`no deposit for ${origin}`)
    }
    const estimatedCostWei = this.estimatedGasCost * gasPrice
    const effectiveDeposit =
      depositBalance > reservedThisRun ? depositBalance - reservedThisRun : 0n
    if (
      !this.relayFunds.hasEnoughBudget(
        origin,
        effectiveDeposit,
        estimatedCostWei,
      )
    ) {
      this.relayFunds.markBlocked(message.messageHash, origin)
      throw new Error(`budget-blocked for ${origin}`)
    }
  }

  /**
   * Queries the destination chain's gas price (wei per gas unit) once per
   * unique destination per run. Missing entries (RPC failure / no client)
   * cause messages targeting that destination to be skipped with reason
   * `no_gas_price` rather than relayed against an unknown cost.
   */
  private async cacheGasPrices(
    ctx: RunContext,
    pendingMessages: SentMessage[],
    log: Logger,
  ): Promise<Map<number, bigint>> {
    const cache = new Map<number, bigint>()
    const destinations = [
      ...new Set(pendingMessages.map((msg) => msg.destination)),
    ]
    await Promise.all(
      destinations.map(async (dest) => {
        const client = ctx.clients.getPublicClient(dest)
        if (!client) return
        try {
          const price = await client.getGasPrice()
          cache.set(dest, price)
        } catch (error) {
          log.error({ error, dest }, 'failed to get gas price for destination')
        }
      }),
    )
    return cache
  }

  private async cacheDepositBalances(
    ctx: RunContext,
    pendingMessages: SentMessage[],
    log: Logger,
  ): Promise<Map<string, bigint>> {
    const depositCache = new Map<string, bigint>()
    const depositOrigins = [
      ...new Set(pendingMessages.map((msg) => msg.txOrigin.toLowerCase())),
    ]
    let nextDepositIndex = 0
    await Promise.all(
      Array.from(
        {
          length: Math.min(DEPOSIT_LOOKUP_CONCURRENCY, depositOrigins.length),
        },
        async () => {
          while (nextDepositIndex < depositOrigins.length) {
            const origin = depositOrigins[nextDepositIndex++]
            if (!origin) continue

            try {
              const deposit = await ctx.ponderClient.getDepositBalance(origin)
              depositCache.set(origin, BigInt(deposit.totalBalance))
            } catch (error) {
              log.error(
                { error },
                `failed to get deposit balance for ${origin}`,
              )
              depositCache.set(origin, 0n)
            }
          }
        },
      ),
    )
    return depositCache
  }

  private async resolveReadyWallets(
    ctx: RunContext,
    destination: number,
    readyWalletsByChain: ReadyWalletsByChain,
    log: Logger,
  ): Promise<ReadyWallet[]> {
    const cached = readyWalletsByChain.get(destination)
    if (cached) return cached

    try {
      const readyWallets = await ctx.clients.resolveWallets(destination)
      readyWalletsByChain.set(destination, readyWallets as ReadyWallet[])
      return readyWallets as ReadyWallet[]
    } catch (error) {
      log.warn({ error, destination }, 'failed to resolve wallets')
      return []
    }
  }

  private async collectReadyRelays(
    ctx: RunContext,
    pendingMessages: SentMessage[],
    depositCache: Map<string, bigint>,
    gasPriceCache: Map<number, bigint>,
    readyWalletsByChain: ReadyWalletsByChain,
    result: RunResult,
    log: Logger,
  ): Promise<ReadyRelaysByChain> {
    const readyByChain = new Map<number, ReadyRelay[]>()
    // Per-run shadow ledger: cost (wei) already approved this run, keyed by
    // tx.origin. Seals the validate-vs-broadcast TOCTOU race for messages
    // sharing a tx.origin. Reservations are NOT persisted — failed broadcasts
    // don't accumulate consumption.
    const reservedThisRun = new Map<string, bigint>()

    for (const message of pendingMessages) {
      const msgLog = log.child({
        source: message.source,
        destination: message.destination,
        messageHash: message.messageHash,
        txHash: message.transactionHash,
      })

      const preAttemptLabels = {
        module: this.name,
        src: String(message.source),
        dst: String(message.destination),
        relayer_eoa: '',
      }

      if (
        message.sender.toLowerCase() !==
        contracts.superchainETHBridge.address.toLowerCase()
      ) {
        msgLog.debug('sender is not SuperchainETHBridge, skipping (no-match)')
        result.noMatch++
        continue
      }

      // Suppress re-broadcasts during the window between submission and the
      // indexer reporting the relay event. These messages are counted by
      // module_relay_tx_in_flight (not module_message_skipped_total).
      if (this.hasSubmitted(message.messageHash)) {
        msgLog.debug('already submitted this session, skipping')
        result.skipped++
        continue
      }

      // Skip if the failure registry says we shouldn't attempt. Split into two
      // skip reasons so operators see *which* state: 'in_backoff' (transient,
      // within an exponential-backoff window after a recent failure) vs.
      // 'abandoned' (permanently flagged as unrelayable). Abandoned entries
      // also surface via module_failure_registry_size; both paths use this
      // skipped-total counter so cycle accounting balances.
      const relayStatus = this.statusFor(message.messageHash)
      if (relayStatus !== 'ready') {
        ctx.metrics.moduleMessageSkippedTotal.inc({
          ...preAttemptLabels,
          reason: relayStatus,
        })
        msgLog.debug({ relayStatus }, 'skipping per failure registry')
        result.skipped++
        continue
      }

      const client = ctx.clients.getPublicClient(message.destination)
      const walletClients = ctx.clients.getWalletClients(message.destination)

      if (!client || walletClients.length === 0) {
        ctx.metrics.moduleMessageSkippedTotal.inc({
          ...preAttemptLabels,
          reason: 'no_client',
        })
        msgLog.warn('no client for destination, skipping...')
        result.skipped++
        continue
      }

      const readyWallets = await this.resolveReadyWallets(
        ctx,
        message.destination,
        readyWalletsByChain,
        log,
      )
      if (readyWallets.length === 0) {
        ctx.metrics.moduleMessageSkippedTotal.inc({
          ...preAttemptLabels,
          reason: 'no_account',
        })
        msgLog.warn('no accounts found, skipping...')
        result.skipped++
        continue
      }

      const gasPrice = gasPriceCache.get(message.destination)
      if (gasPrice === undefined) {
        ctx.metrics.moduleMessageSkippedTotal.inc({
          ...preAttemptLabels,
          reason: 'no_gas_price',
        })
        msgLog.warn('gas price unavailable for destination, skipping')
        result.skipped++
        continue
      }

      const attemptStart = performance.now()
      const originKey = message.txOrigin.toLowerCase()
      const depositBalance = depositCache.get(originKey) ?? 0n
      const reserved = reservedThisRun.get(originKey) ?? 0n

      try {
        await this.validate(message, depositBalance, gasPrice, reserved)
      } catch (error) {
        // Both "literally zero deposit" and "deposit < estimated cost" surface
        // as the same operator question: this user can't pay. Single reason
        // simplifies the dashboard; the markBlocked side-effect keeps the
        // budget-vs-zero distinction internal.
        ctx.metrics.moduleMessageSkippedTotal.inc({
          ...preAttemptLabels,
          reason: 'insufficient_deposit_bal',
        })
        const isBudgetBlocked =
          error instanceof Error && error.message.includes('budget-blocked')
        msgLog.warn(
          { err: error, isBudgetBlocked },
          isBudgetBlocked
            ? 'insufficient deposit (budget-blocked), will retry on balance change'
            : `insufficient deposit, skipping message ${message.messageHash}`,
        )
        result.skipped++
        continue
      }

      reservedThisRun.set(
        originKey,
        reserved + this.estimatedGasCost * gasPrice,
      )
      const relay = this.buildReadyRelay(message, attemptStart, gasPrice)
      const queue = readyByChain.get(message.destination) ?? []
      queue.push(relay)
      readyByChain.set(message.destination, queue)
    }

    return readyByChain
  }

  private buildReadyRelay(
    message: SentMessage,
    attemptStart: number,
    gasPrice: bigint,
  ): ReadyRelay {
    const id: MessageIdentifier = {
      origin: contracts.l2ToL2CrossDomainMessenger.address,
      chainId: BigInt(message.source),
      logIndex: BigInt(message.logIndex),
      blockNumber: BigInt(message.blockNumber),
      timestamp: BigInt(message.timestamp),
    }
    const payload = message.logPayload as `0x${string}`
    const accessList = encodeAccessList(id, payload)

    return {
      id,
      destinationChainId: message.destination,
      payload,
      accessList,
      chain: null,
      txOrigin: message.txOrigin.toLowerCase(),
      messageHash: message.messageHash,
      estimatedGasCost: this.estimatedGasCost,
      attemptStart,
      retry: this.isRetry(message.messageHash),
      gasPrice,
    }
  }

  private async broadcastRelays(
    ctx: RunContext,
    readyByChain: ReadyRelaysByChain,
    readyWalletsByChain: ReadyWalletsByChain,
    result: RunResult,
  ): Promise<void> {
    await Promise.all(
      [...readyByChain.entries()].map(async ([dest, relays]) => {
        const wallets = readyWalletsByChain.get(dest) ?? []
        if (wallets.length === 0) return
        // Shared per-destination public client used by confirmAsync to wait
        // for receipts. Resolved once per chain to avoid repeated lookups in
        // the inner per-wallet loop.
        const publicClient = ctx.clients.getPublicClient(dest)
        let nextRelayIndex = 0

        await Promise.all(
          wallets.map(async ({ account, walletClient }) => {
            while (true) {
              const readyRelay = relays[nextRelayIndex++]
              if (!readyRelay) break

              const { attemptStart, retry, gasPrice, ...base } = readyRelay
              const relayParams: RelayMessageParams = {
                ...base,
                account,
                walletClient,
              }
              const attemptLabels = {
                module: this.name,
                src: String(relayParams.id.chainId),
                dst: String(dest),
                relayer_eoa: account.address.toLowerCase(),
              }

              if (retry) {
                ctx.metrics.moduleRelayAttemptRetryTotal.inc(attemptLabels)
              }

              try {
                const txHash = await ctx.relayMessage(relayParams)
                ctx.metrics.moduleRelayTxBroadcastTotal.inc(attemptLabels)
                result.relayed++
                this.relayFunds.recordConsumption(
                  relayParams.txOrigin,
                  this.estimatedGasCost * gasPrice,
                )
                this.relayFunds.clearBlocked(relayParams.messageHash)
                this.recordSubmitted(relayParams.messageHash, attemptLabels)
                // Fire-and-forget receipt wait so module_relay_tx_executed_total
                // and the attempt_duration_seconds executed/failed observation
                // fire for eth-bridge — closing the friction §2.F1 asymmetry
                // gap. publicClient may be undefined in rare config-edge cases;
                // skip cleanly rather than throwing.
                if (publicClient) {
                  this.confirmAsync(
                    ctx,
                    publicClient,
                    txHash,
                    attemptLabels,
                    attemptStart,
                    relayParams.messageHash,
                  )
                }
              } catch (error) {
                this.recordRelayFailure(
                  ctx,
                  attemptLabels,
                  error,
                  'broadcast',
                  relayParams.messageHash,
                  attemptStart,
                  result,
                )
              }
            }
          }),
        )
      }),
    )
  }

  private recordRelayFailure(
    ctx: RunContext,
    attemptLabels: {
      module: string
      src: string
      dst: string
      relayer_eoa: string
    },
    error: unknown,
    stageFallback: 'simulation' | 'broadcast',
    messageHash: string,
    attemptStart: number,
    result: RunResult,
  ): void {
    const stage = error instanceof RelayError ? error.stage : stageFallback
    const reason = error instanceof RelayError ? error.reason : 'unknown'
    ctx.metrics.moduleRelayAttemptFailedTotal.inc({
      ...attemptLabels,
      stage,
      reason,
    })
    this.observeAttemptDuration(ctx, attemptLabels, 'failed', attemptStart)
    this.recordFailure(
      messageHash,
      Number(attemptLabels.src),
      Number(attemptLabels.dst),
      reason,
    )
    result.failed++
  }
}
