import type { SentMessage } from '@eth-optimism/ponder-interop/client'
import { contracts } from '@eth-optimism/viem'
import type { MessageIdentifier } from '@eth-optimism/viem/types/interop'
import { encodeAccessList } from '@eth-optimism/viem/utils/interop'

import { RelayError } from '@/errors.js'
import type { RelayFailureRegistry } from '@/relay/relayFailureRegistry.js'

import { BaseRelayModule } from './baseRelayModule.js'
import type { PonderEndpoint, RunContext, RunResult } from './types.js'

// Known function selectors for intent decoding
const KNOWN_SELECTORS: Record<string, string> = {
  // Promise contract
  '0x9bc97158':
    'Promise.receiveResolverTransfer — delivering resolver transfer to destination chain',
  '0x5d8d7b8d': 'Promise.shareResolvedPromise — sharing resolution cross-chain',
  '0xbb3d271c':
    'Promise.transferResolve — transferring resolution to new chain',
  '0xca48e9c3': 'Promise.resolve — resolving a promise',
  // Callback contract
  '0xc08aeccd':
    'Callback.receiveCallbackRegistration — registering cross-chain callback on destination',
  '0x5c23bdf5': 'Callback.resolve — executing a callback',
  // SuperchainETHBridge
  '0x1532ec34': 'SuperchainETHBridge.sendETH',
  '0x56cf5429': 'SuperchainETHBridge.relayETH',
  // L2ToL2CDM
  '0xd764ad0b': 'L2ToL2CDM.relayMessage',
}

/**
 * Extracts intent information from a cross-chain message.
 */
function describeIntent(message: {
  sender: string
  target: string
  message: string
}): { sender: string; target: string; selector: string; fnName: string } {
  const selector = message.message.slice(0, 10)
  const fnName = KNOWN_SELECTORS[selector] ?? 'unknown'
  return {
    sender: message.sender,
    target: message.target,
    selector,
    fnName,
  }
}

/**
 * General-purpose relay module for all L2ToL2CDM messages that aren't owned by
 * a specialized module (e.g. EthBridgeModule). This is the catch-all that
 * delivers, among other things, the cross-domain messages emitted by promise
 * sharing and callback registration (Promise.shareResolvedPromise →
 * receiveSharedPromise, Callback.thenOn → receiveCallbackRegistration, …).
 *
 * Ownership is explicit: the relayer passes the union of every other enabled
 * module's `ownedSenders`, and this module skips any message whose sender is in
 * that set. That replaces the old hard-coded "skip SuperchainETHBridge" check
 * so adding a new specialized module never silently double-relays.
 *
 * No deposit gating or validation in Phase 1 — the relayer EOA pays.
 *
 * Tracks relayed message hashes in memory to avoid re-submitting while waiting
 * for Ponder to index RelayedMessage events on the destination chain.
 */
export class GeneralRelayModule extends BaseRelayModule {
  readonly name = 'general-relay'
  readonly needs = [
    'pendingMessages',
  ] as const satisfies readonly PonderEndpoint[]
  // Owns every pending message not claimed by another module's ownedSenders.
  // Signals the relayer that no pending message can be "unowned" this cycle.
  readonly catchAll = true

  // Lowercased sender addresses claimed by other enabled modules. Messages
  // from these senders are skipped (counted as noMatch) — their owning module
  // relays them.
  private readonly claimedSenders: ReadonlySet<string>

  constructor(
    failureRegistry: RelayFailureRegistry,
    claimedSenders: ReadonlySet<string> = new Set(),
  ) {
    super(failureRegistry)
    this.claimedSenders = claimedSenders
  }

  async run(ctx: RunContext): Promise<RunResult> {
    const log = this.moduleLog(ctx)
    const result: RunResult = { relayed: 0, skipped: 0, failed: 0, noMatch: 0 }

    const pendingMessages = ctx.pendingMessages
    log.info(`${pendingMessages.length} pending messages`)

    // Classify pending messages this module owns (sender not claimed by another
    // enabled module):
    //   - ready → module_message_backlog (has client for dest, not in-flight)
    // In-flight (already broadcast this session) handled by pruneAndEmitInFlight.
    // No-client messages are owned (so pruneAndEmitInFlight sees them) but
    // excluded from backlog; the attempt loop counts them as skipped{no_client}.
    const ownedPending: SentMessage[] = []
    const backlog = new Map<
      string,
      { src: string; dst: string; count: number }
    >()
    for (const m of pendingMessages) {
      if (this.claimedSenders.has(m.sender.toLowerCase())) continue
      ownedPending.push(m)
      if (this.hasSubmitted(m.messageHash)) continue
      const hasClient =
        !!ctx.clients.getPublicClient(m.destination) &&
        !!ctx.clients.getWalletClient(m.destination)
      if (!hasClient) continue
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
      ctx.metrics.moduleMessageBacklog.set(
        { module: this.name, src: b.src, dst: b.dst, relayer_eoa: '' },
        b.count,
      )
    }
    this.pruneAndEmitInFlight(ctx, ownedPending)

    for (const message of pendingMessages) {
      const intent = describeIntent(message)
      const msgLog = log.child({
        source: message.source,
        destination: message.destination,
        messageHash: message.messageHash,
      })

      const preAttemptLabels = {
        module: this.name,
        src: String(message.source),
        dst: String(message.destination),
        relayer_eoa: '',
      }

      // Skip messages claimed by another enabled module (e.g. eth-bridge).
      if (this.claimedSenders.has(message.sender.toLowerCase())) {
        result.noMatch++
        continue
      }

      // Skip messages already relayed this session (waiting for the indexer).
      // These are counted by module_relay_tx_in_flight, not skipped_total.
      if (this.hasSubmitted(message.messageHash)) {
        msgLog.debug('already relayed this session, skipping')
        result.skipped++
        continue
      }

      // Client lookup
      const client = ctx.clients.getPublicClient(message.destination)
      const walletClient = ctx.clients.getWalletClient(message.destination)

      if (!client || !walletClient) {
        ctx.metrics.moduleMessageSkippedTotal.inc({
          ...preAttemptLabels,
          reason: 'no_client',
        })
        msgLog.warn('no client for destination, skipping...')
        result.skipped++
        continue
      }

      const account = await ctx.clients.resolveAccount(walletClient)
      if (!account) {
        ctx.metrics.moduleMessageSkippedTotal.inc({
          ...preAttemptLabels,
          reason: 'no_account',
        })
        msgLog.warn('no accounts found, skipping...')
        result.skipped++
        continue
      }

      const attemptLabels = {
        module: this.name,
        src: String(message.source),
        dst: String(message.destination),
        relayer_eoa: account.address.toLowerCase(),
      }

      if (this.isRetry(message.messageHash)) {
        ctx.metrics.moduleRelayAttemptRetryTotal.inc(attemptLabels)
      }

      const attemptStart = performance.now()

      // Build MessageIdentifier
      const id: MessageIdentifier = {
        origin: contracts.l2ToL2CrossDomainMessenger.address,
        chainId: BigInt(message.source),
        logIndex: BigInt(message.logIndex),
        blockNumber: BigInt(message.blockNumber),
        timestamp: BigInt(message.timestamp),
      }
      const payload = message.logPayload as `0x${string}`
      const accessList = encodeAccessList(id, payload)

      try {
        const relayTxHash = await ctx.relayMessage({
          id,
          destinationChainId: message.destination,
          payload,
          accessList,
          account,
          chain: null,
          walletClient,
          txOrigin: message.sender.toLowerCase(),
          messageHash: message.messageHash,
        })
        ctx.metrics.moduleRelayTxBroadcastTotal.inc(attemptLabels)
        msgLog.info(
          {
            tx_hash: relayTxHash,
            relayer_eoa: account.address.toLowerCase(),
            target_contract: intent.target,
            function_selector: intent.selector,
            intent: intent.fnName,
            sender: intent.sender,
          },
          `relayed [${intent.fnName}] from ${intent.sender} → ${intent.target}`,
        )
        this.recordSubmitted(message.messageHash, attemptLabels)
        this.confirmAsync(
          ctx,
          client,
          relayTxHash,
          attemptLabels,
          attemptStart,
          message.messageHash,
        )
        result.relayed++
      } catch (error) {
        const stage = error instanceof RelayError ? error.stage : 'broadcast'
        const reason = error instanceof RelayError ? error.reason : 'unknown'
        ctx.metrics.moduleRelayAttemptFailedTotal.inc({
          ...attemptLabels,
          stage,
          reason,
        })
        this.observeAttemptDuration(ctx, attemptLabels, 'failed', attemptStart)
        this.recordFailure(
          message.messageHash,
          message.source,
          message.destination,
          reason,
        )
        msgLog.warn(
          {
            err: error,
            stage,
            reason,
            relayer_eoa: account.address.toLowerCase(),
            target_contract: intent.target,
            function_selector: intent.selector,
            intent: intent.fnName,
          },
          `relay failed, skipping...`,
        )
        result.failed++
        continue
      }
    }

    return result
  }
}
