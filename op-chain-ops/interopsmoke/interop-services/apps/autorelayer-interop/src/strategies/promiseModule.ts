import type { RelayFailureRegistry } from '@/relay/relayFailureRegistry.js'

import { BaseRelayModule } from './baseRelayModule.js'
import type { PonderEndpoint, RunContext, RunResult } from './types.js'

const CAN_RESOLVE_ABI = [
  {
    type: 'function',
    name: 'canResolve',
    inputs: [{ name: 'promiseId', type: 'bytes32' }],
    outputs: [{ name: '', type: 'bool' }],
    stateMutability: 'view',
  },
] as const

const RESOLVE_ABI = [
  {
    type: 'function',
    name: 'resolve',
    inputs: [{ name: 'promiseId', type: 'bytes32' }],
    outputs: [],
    stateMutability: 'nonpayable',
  },
] as const

/**
 * Resolves promises that are ready (`canResolve()` is true) by calling
 * `resolve()` directly on the resolver contract. Intra-chain: resolution
 * happens on the chain where the promise lives (src == dst == promise.chainId).
 * Consumes the indexer's pending-promises list.
 */
export class PromiseModule extends BaseRelayModule {
  readonly name = 'promise'
  readonly needs = [
    'pendingPromises',
  ] as const satisfies readonly PonderEndpoint[]

  constructor(failureRegistry: RelayFailureRegistry) {
    super(failureRegistry)
  }

  async run(ctx: RunContext): Promise<RunResult> {
    const log = this.moduleLog(ctx)
    const result: RunResult = { relayed: 0, skipped: 0, failed: 0, noMatch: 0 }

    const pendingPromises = ctx.pendingPromises
    log.info(`${pendingPromises.length} pending promises`)

    // Fan out canResolve() in parallel so we can classify pending promises
    // before the attempt loop. Results are cached and reused below — each
    // promise costs one RPC per cycle, not two. Cache misses (no client or
    // RPC error) fall through to the attempt loop, which classifies the
    // failure into the appropriate skip/fail reason.
    const canResolveCache = new Map<string, boolean>()
    const canResolveResults = await Promise.all(
      pendingPromises.map(async (p) => {
        const client = ctx.clients.getPublicClient(p.chainId)
        if (!client) return null
        try {
          return (await client.readContract({
            address: p.resolver as `0x${string}`,
            abi: CAN_RESOLVE_ABI,
            functionName: 'canResolve',
            args: [p.promiseId as `0x${string}`],
          })) as boolean
        } catch {
          return null
        }
      }),
    )
    for (let i = 0; i < pendingPromises.length; i++) {
      const r = canResolveResults[i]
      if (r !== null) canResolveCache.set(pendingPromises[i].promiseId, r)
    }

    // Backlog: promises that can be resolved right now and we haven't already
    // broadcast this session. Resolution is intra-chain (src == dst == chainId).
    const backlog = new Map<string, { chain: string; count: number }>()
    for (const p of pendingPromises) {
      if (this.hasSubmitted(p.promiseId)) continue
      if (canResolveCache.get(p.promiseId) !== true) continue
      const chain = String(p.chainId)
      const b = backlog.get(chain) ?? { chain, count: 0 }
      b.count++
      backlog.set(chain, b)
    }
    for (const b of backlog.values()) {
      ctx.metrics.moduleMessageBacklog.set(
        { module: this.name, src: b.chain, dst: b.chain, relayer_eoa: '' },
        b.count,
      )
    }

    this.pruneAndEmitInFlight(
      ctx,
      pendingPromises.map((p) => ({
        messageHash: p.promiseId,
        source: p.chainId,
        destination: p.chainId,
      })),
    )

    for (const promise of pendingPromises) {
      const promiseLog = log.child({
        promiseId: promise.promiseId,
        resolver: promise.resolver,
        chainId: promise.chainId,
      })

      // Promise resolution is intra-chain: src == dst == promise.chainId.
      const preAttemptLabels = {
        module: this.name,
        src: String(promise.chainId),
        dst: String(promise.chainId),
        relayer_eoa: '',
      }

      // Client lookup
      const client = ctx.clients.getPublicClient(promise.chainId)
      const walletClient = ctx.clients.getWalletClient(promise.chainId)

      if (!client || !walletClient) {
        ctx.metrics.moduleMessageSkippedTotal.inc({
          ...preAttemptLabels,
          reason: 'no_client',
        })
        promiseLog.warn('no client for chain, skipping...')
        result.skipped++
        continue
      }

      // Account resolution via ClientManager
      const account = await ctx.clients.resolveAccount(walletClient)
      if (!account) {
        ctx.metrics.moduleMessageSkippedTotal.inc({
          ...preAttemptLabels,
          reason: 'no_account',
        })
        promiseLog.warn('no accounts found, skipping...')
        result.skipped++
        continue
      }

      const attemptLabels = {
        module: this.name,
        src: String(promise.chainId),
        dst: String(promise.chainId),
        relayer_eoa: account.address.toLowerCase(),
      }

      const attemptStart = performance.now()
      const promiseId = promise.promiseId as `0x${string}`

      try {
        // Verify resolver contract exists
        const code = await client.getCode({
          address: promise.resolver as `0x${string}`,
        })
        if (code === '0x' || code === undefined || code === null) {
          ctx.metrics.moduleMessageSkippedTotal.inc({
            ...attemptLabels,
            reason: 'no_resolver_code',
          })
          promiseLog.warn('promise creator contract not found, skipping...')
          result.skipped++
          continue
        }

        // Cache-first canResolve: reuse the upfront fan-out result if present
        // so each promise costs one RPC per cycle. Cache miss (no client,
        // upfront RPC errored) falls through to the inline check so the
        // original simulation/resolver_unreachable classification is preserved.
        const cachedCanResolve = canResolveCache.get(promiseId)
        if (cachedCanResolve === false) {
          ctx.metrics.moduleMessageSkippedTotal.inc({
            ...attemptLabels,
            reason: 'promise_not_ready',
          })
          promiseLog.debug(`promise ${promiseId} cannot be resolved yet`)
          result.skipped++
          continue
        }
        if (cachedCanResolve !== true) {
          try {
            const canResolve = await client.readContract({
              address: promise.resolver as `0x${string}`,
              abi: CAN_RESOLVE_ABI,
              functionName: 'canResolve',
              args: [promiseId],
            })
            if (!canResolve) {
              ctx.metrics.moduleMessageSkippedTotal.inc({
                ...attemptLabels,
                reason: 'promise_not_ready',
              })
              promiseLog.debug(`promise ${promiseId} cannot be resolved yet`)
              result.skipped++
              continue
            }
          } catch (error) {
            ctx.metrics.moduleRelayAttemptFailedTotal.inc({
              ...attemptLabels,
              stage: 'simulation',
              reason: 'resolver_unreachable',
            })
            this.observeAttemptDuration(
              ctx,
              attemptLabels,
              'failed',
              attemptStart,
            )
            promiseLog.warn(
              {
                err: error,
                stage: 'simulation',
                reason: 'resolver_unreachable',
                relayer_eoa: account.address.toLowerCase(),
                target_contract: promise.resolver,
              },
              'failed to check if promise can be resolved',
            )
            result.failed++
            continue
          }
        }

        promiseLog.info(
          `promise ${promiseId} can be resolved, attempting to resolve...`,
        )

        // Call resolve directly on resolver contract (NOT via ctx.relayMessage)
        // Promise resolution is fundamentally different from message relay --
        // promises are resolved, not relayed through the messenger.
        //
        // Gas: resolve() executes the callback inside try/catch, so
        // eth_estimateGas returns the minimum at which the OUTER tx succeeds —
        // which includes the callback itself starving at the 63/64 boundary
        // (it then rejects with CallbackExecuted(false, "") and no error
        // data). Estimate explicitly and pad, so heavy callbacks actually run.
        const resolveCall = {
          address: promise.resolver as `0x${string}`,
          abi: RESOLVE_ABI,
          functionName: 'resolve',
          args: [promiseId],
          account,
        } as const
        const resolveGasEstimate = await client.estimateContractGas(resolveCall)
        const resolveTxHash = await walletClient.writeContract({
          ...resolveCall,
          gas: (resolveGasEstimate * 3n) / 2n,
          chain: null,
        })

        ctx.metrics.moduleRelayTxBroadcastTotal.inc(attemptLabels)
        this.recordSubmitted(promiseId, attemptLabels)
        this.confirmAsync(
          ctx,
          client,
          resolveTxHash,
          attemptLabels,
          attemptStart,
          promiseId,
        )
        promiseLog.info(
          {
            tx_hash: resolveTxHash,
            relayer_eoa: account.address.toLowerCase(),
            target_contract: promise.resolver,
            function_selector: '0xca48e9c3',
          },
          'submitted promise resolution',
        )
        result.relayed++
      } catch (error) {
        ctx.metrics.moduleRelayAttemptFailedTotal.inc({
          ...attemptLabels,
          stage: 'broadcast',
          reason: 'unknown',
        })
        this.observeAttemptDuration(ctx, attemptLabels, 'failed', attemptStart)
        promiseLog.warn(
          {
            err: error,
            stage: 'broadcast',
            reason: 'unknown',
            relayer_eoa: account.address.toLowerCase(),
            target_contract: promise.resolver,
          },
          `failed to process promise ${promiseId}`,
        )
        result.failed++
        continue
      }
    }

    return result
  }
}
