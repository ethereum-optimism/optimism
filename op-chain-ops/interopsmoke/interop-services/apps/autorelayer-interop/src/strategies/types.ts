import type {
  PonderInteropClient,
  PonderPromise,
  SentMessage,
  UnsharedResolvedPromise,
} from '@eth-optimism/ponder-interop/client'
import type { MessageIdentifier } from '@eth-optimism/viem/types/interop'
import type { Logger } from 'pino'
import type {
  AccessList,
  Account,
  Address,
  Chain,
  Hex,
  WalletClient,
} from 'viem'

import type { RelayerMetrics } from '@/metrics.js'
import type { ClientManager } from '@/services/clientManager.js'

/**
 * Per-cycle Ponder list endpoints the Relayer pre-fetches. Each module
 * declares which of these it consumes via the `needs` member; the Relayer
 * fetches only the union across enabled modules. Per-call endpoints
 * (e.g. getDepositBalance) are not modeled here — modules call those directly.
 */
export type PonderEndpoint =
  | 'pendingMessages'
  | 'pendingPromises'
  | 'unsharedResolvedPromises'

/**
 * Parameters for relaying a cross-domain message via the orchestrator.
 * Passed to the relayMessage callback on RunContext.
 */
export interface RelayMessageParams {
  id: MessageIdentifier
  destinationChainId: number
  payload: Hex
  account: Account
  accessList: AccessList
  chain: Chain | null
  walletClient: WalletClient
  txOrigin: string
  messageHash: string
  estimatedGasCost?: bigint | null
}

/**
 * Context provided to each module on every run() call.
 * Owned by the Relayer orchestrator.
 */
export interface RunContext {
  ponderClient: PonderInteropClient
  clients: ClientManager
  log: Logger
  pendingMessages: SentMessage[]
  pendingPromises: PonderPromise[]
  unsharedResolvedPromises: UnsharedResolvedPromise[]
  /**
   * Endpoints whose fetch failed this cycle. The corresponding list above is
   * empty, but that emptiness is NOT evidence the work is done — modules must
   * not garbage-collect state (failure registry, session dedup) keyed on a
   * list whose endpoint appears here. Absent/empty means all fetches
   * succeeded.
   */
  failedEndpoints?: ReadonlySet<PonderEndpoint>
  metrics: RelayerMetrics
  relayMessage(params: RelayMessageParams): Promise<Hex>
}

/**
 * Structured result returned by each module after a run.
 */
export interface RunResult {
  relayed: number
  skipped: number
  failed: number
  noMatch: number
}

/**
 * Interface for pluggable relay modules.
 */
export interface RelayModule {
  readonly name: string
  /**
   * Per-cycle Ponder list endpoints this module consumes. The Relayer fetches
   * the union across enabled modules and passes the results via RunContext.
   */
  readonly needs: readonly PonderEndpoint[]
  /**
   * Source addresses whose pending `pendingMessages` this module owns. Modules
   * that consume `pendingMessages` declare their senders here so the catch-all
   * GeneralRelayModule can avoid double-relaying messages another module
   * handles. Modules that don't consume `pendingMessages` omit this.
   */
  readonly ownedSenders?: readonly Address[]
  /**
   * True for a module that owns every `pendingMessages` entry not claimed by
   * another module's `ownedSenders` (GeneralRelayModule). When no enabled
   * module is a catch-all, the relayer counts unclaimed pending messages in
   * the relayer_messages_unowned gauge so a too-narrow ENABLED_MODULES
   * configuration is visible in metrics instead of only in per-module logs.
   */
  readonly catchAll?: boolean
  run(ctx: RunContext): Promise<RunResult>
}
