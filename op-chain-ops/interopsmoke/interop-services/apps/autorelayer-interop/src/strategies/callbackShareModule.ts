import type { RelayFailureRegistry } from '@/relay/relayFailureRegistry.js'

import { BaseRelayModule } from './baseRelayModule.js'
import type { PonderEndpoint, RunContext, RunResult } from './types.js'

const SHARE_RESOLVED_PROMISE_ABI = [
  {
    type: 'function',
    name: 'shareResolvedPromise',
    inputs: [
      { name: '_chainId', type: 'uint256', internalType: 'uint256' },
      { name: '_promiseId', type: 'bytes32', internalType: 'bytes32' },
    ],
    outputs: [],
    stateMutability: 'nonpayable',
  },
] as const

/**
 * Shares resolved promises to chains that have callbacks waiting on them by
 * calling `Promise.shareResolvedPromise(destChainId, promiseId)` on the chain
 * where the promise is resolved. That emits an L2→L2 message which
 * GeneralRelayModule then delivers to the destination.
 *
 * Uses the server-side `/promises/unshared-resolved` endpoint which only
 * returns promises with callback chains that haven't received the resolution
 * yet — no client-side dedup set needed.
 */
export class CallbackShareModule extends BaseRelayModule {
  readonly name = 'callback-share'
  readonly needs = [
    'unsharedResolvedPromises',
  ] as const satisfies readonly PonderEndpoint[]

  private readonly promiseAddress: `0x${string}`

  constructor(
    promiseAddress: `0x${string}`,
    failureRegistry: RelayFailureRegistry,
  ) {
    super(failureRegistry)
    this.promiseAddress = promiseAddress
  }

  async run(ctx: RunContext): Promise<RunResult> {
    const log = this.moduleLog(ctx)
    const result: RunResult = { relayed: 0, skipped: 0, failed: 0, noMatch: 0 }

    const unshared = ctx.unsharedResolvedPromises
    if (unshared.length === 0) return result
    log.info(`${unshared.length} resolved promises need cross-chain sharing`)

    // Report pre-attempt backlog: each (resolvedChain -> destChain) share is
    // one unit of work. One resolved promise may fan out to many destChains.
    // Used with module_relay_tx_last_executed_timestamp for stall detection.
    const backlog = new Map<
      string,
      { src: string; dst: string; count: number }
    >()
    for (const p of unshared) {
      for (const destChainId of p.pendingChainIds) {
        const key = `${p.chainId}|${destChainId}`
        const b = backlog.get(key) ?? {
          src: String(p.chainId),
          dst: String(destChainId),
          count: 0,
        }
        b.count++
        backlog.set(key, b)
      }
    }
    for (const b of backlog.values()) {
      ctx.metrics.moduleMessageBacklog.set(
        { module: this.name, src: b.src, dst: b.dst, relayer_eoa: '' },
        b.count,
      )
    }

    for (const promise of unshared) {
      const promiseId = promise.promiseId as `0x${string}`

      // Pre-attempt labels use src=resolved chain, dst='' (unknown dest -- one
      // unshared promise may fan out to many pendingChainIds below).
      const sourceLabels = {
        module: this.name,
        src: String(promise.chainId),
        dst: '',
        relayer_eoa: '',
      }

      // Share from the chain where the promise is resolved
      const publicClient = ctx.clients.getPublicClient(promise.chainId)
      const walletClient = ctx.clients.getWalletClient(promise.chainId)
      if (!publicClient || !walletClient) {
        ctx.metrics.moduleMessageSkippedTotal.inc({
          ...sourceLabels,
          reason: 'no_client',
        })
        log.warn(
          { chainId: promise.chainId, promiseId },
          'no client for resolved chain, skipping',
        )
        result.skipped++
        continue
      }

      const account = await ctx.clients.resolveAccount(walletClient)
      if (!account) {
        ctx.metrics.moduleMessageSkippedTotal.inc({
          ...sourceLabels,
          reason: 'no_account',
        })
        log.warn(
          { chainId: promise.chainId, promiseId },
          'no accounts found, skipping',
        )
        result.skipped++
        continue
      }

      for (const destChainId of promise.pendingChainIds) {
        const shareLog = log.child({ promiseId, destChainId })

        const attemptLabels = {
          module: this.name,
          src: String(promise.chainId),
          dst: String(destChainId),
          relayer_eoa: account.address.toLowerCase(),
        }

        const attemptStart = performance.now()

        try {
          const shareTxHash = await walletClient.writeContract({
            address: this.promiseAddress,
            abi: SHARE_RESOLVED_PROMISE_ABI,
            functionName: 'shareResolvedPromise',
            args: [BigInt(destChainId), promiseId],
            account,
            chain: null,
          })
          ctx.metrics.moduleRelayTxBroadcastTotal.inc(attemptLabels)
          this.confirmAsync(
            ctx,
            publicClient,
            shareTxHash,
            attemptLabels,
            attemptStart,
          )
          shareLog.info(
            {
              tx_hash: shareTxHash,
              relayer_eoa: account.address.toLowerCase(),
              target_contract: this.promiseAddress,
              function_selector: '0x5d8d7b8d',
            },
            'shared resolved promise cross-chain',
          )
          result.relayed++
        } catch (err) {
          ctx.metrics.moduleRelayAttemptFailedTotal.inc({
            ...attemptLabels,
            stage: 'broadcast',
            reason: 'unknown',
          })
          this.observeAttemptDuration(
            ctx,
            attemptLabels,
            'failed',
            attemptStart,
          )
          shareLog.warn(
            {
              err,
              stage: 'broadcast',
              reason: 'unknown',
              relayer_eoa: account.address.toLowerCase(),
              target_contract: this.promiseAddress,
            },
            'failed to share resolved promise',
          )
          result.failed++
        }
      }
    }

    return result
  }
}
