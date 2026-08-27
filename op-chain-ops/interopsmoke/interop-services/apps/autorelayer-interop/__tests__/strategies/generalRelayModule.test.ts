import type {
  PonderInteropClient,
  SentMessage,
} from '@eth-optimism/ponder-interop/client'
import { contracts } from '@eth-optimism/viem'
import { Registry } from 'prom-client'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { RelayError } from '@/errors.js'
import { RelayerMetrics } from '@/metrics.js'
import { RelayFailureRegistry } from '@/relay/relayFailureRegistry.js'
import type { ClientManager } from '@/services/clientManager.js'
import { GeneralRelayModule } from '@/strategies/generalRelayModule.js'
import type { RunContext } from '@/strategies/types.js'

vi.mock('@eth-optimism/viem/utils/interop', () => ({
  encodeAccessList: vi.fn().mockReturnValue([]),
}))

interface MockLogger {
  child: ReturnType<typeof vi.fn>
  info: ReturnType<typeof vi.fn>
  warn: ReturnType<typeof vi.fn>
  error: ReturnType<typeof vi.fn>
  debug: ReturnType<typeof vi.fn>
}

const SUPERCHAIN_ETH_BRIDGE = contracts.superchainETHBridge.address

const baseMessage: SentMessage = {
  messageIdentifierHash: '0xabcd',
  messageHash: '0x1234',
  source: 1,
  destination: 901,
  nonce: 1,
  sender: '0x1111111111111111111111111111111111111111',
  target: '0x2222222222222222222222222222222222222222',
  message: '0xd764ad0b',
  logIndex: 5,
  logPayload: '0x5678',
  timestamp: 1234567890,
  blockNumber: 100,
  transactionHash:
    '0x9999999999999999999999999999999999999999999999999999999999999999',
  txOrigin: '0x3333333333333333333333333333333333333333',
}

const RELAYER_EOA = '0x1234567890123456789012345678901234567890'

describe('GeneralRelayModule funnel', () => {
  let module: GeneralRelayModule
  let mockCtx: RunContext
  let mockLogger: MockLogger
  let registry: Registry

  beforeEach(() => {
    mockLogger = {
      child: vi.fn().mockReturnThis(),
      info: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
      debug: vi.fn(),
    }
    registry = new Registry()

    mockCtx = {
      ponderClient: {} as PonderInteropClient,
      clients: {
        getPublicClient: vi.fn().mockReturnValue({}),
        getWalletClient: vi.fn().mockReturnValue({}),
        resolveAccount: vi
          .fn()
          .mockResolvedValue({ address: RELAYER_EOA, type: 'json-rpc' }),
      } as unknown as ClientManager,
      log: mockLogger as unknown as RunContext['log'],
      pendingMessages: [],
      metrics: new RelayerMetrics(registry),
      relayMessage: vi.fn().mockResolvedValue('0xtxhash'),
    }

    // The eth-bridge sender is claimed by another module; general-relay must
    // skip it (Decision A — ownership, not a hard-coded bridge check).
    module = new GeneralRelayModule(
      new RelayFailureRegistry(':memory:'),
      new Set([SUPERCHAIN_ETH_BRIDGE.toLowerCase()]),
    )
  })

  async function scrape(): Promise<string> {
    return await registry.metrics()
  }

  it('classifies messages from a claimed sender (eth-bridge) as no-match', async () => {
    mockCtx.pendingMessages = [
      { ...baseMessage, sender: SUPERCHAIN_ETH_BRIDGE },
    ]

    const result = await module.run(mockCtx)

    expect(result.noMatch).toBe(1)
    expect(result.relayed).toBe(0)
    const output = await scrape()
    expect(output).not.toMatch(
      /relayer_module_relay_tx_broadcast_total\{.*\} [1-9]/,
    )
  })

  it('tracks re-visited in-flight messages in module_relay_tx_in_flight without re-broadcasting', async () => {
    mockCtx.pendingMessages = [baseMessage]

    await module.run(mockCtx) // cycle 1: broadcasts
    await module.run(mockCtx) // cycle 2: already submitted, counted in_flight

    const output = await scrape()
    expect(output).toMatch(/relayer_module_relay_tx_in_flight\{.*\} 1/)
    // Broadcast counter incremented once, not twice
    expect(output).toMatch(/relayer_module_relay_tx_broadcast_total\{.*\} 1/)
  })

  it('increments skipped_total{reason=no_client} when destination has no client', async () => {
    mockCtx.pendingMessages = [baseMessage]
    vi.mocked(mockCtx.clients.getPublicClient).mockReturnValue(undefined)

    await module.run(mockCtx)

    const output = await scrape()
    expect(output).toMatch(
      /relayer_module_message_skipped_total\{.*reason="no_client".*\} 1/,
    )
  })

  it('increments skipped_total{reason=no_account} when resolveAccount returns undefined', async () => {
    mockCtx.pendingMessages = [baseMessage]
    vi.mocked(mockCtx.clients.resolveAccount).mockResolvedValue(undefined)

    await module.run(mockCtx)

    const output = await scrape()
    expect(output).toMatch(
      /relayer_module_message_skipped_total\{.*reason="no_account".*\} 1/,
    )
  })

  it('increments broadcast_total on successful relay with relayer_eoa label', async () => {
    mockCtx.pendingMessages = [baseMessage]

    await module.run(mockCtx)

    const output = await scrape()
    expect(output).toContain(
      `relayer_module_relay_tx_broadcast_total{module="general-relay",src="1",dst="901",relayer_eoa="${RELAYER_EOA.toLowerCase()}"} 1`,
    )
  })

  it('does not pass a fixed gas estimate to relayMessage', async () => {
    mockCtx.pendingMessages = [baseMessage]

    await module.run(mockCtx)

    expect(mockCtx.relayMessage).toHaveBeenCalledWith(
      expect.not.objectContaining({
        estimatedGasCost: expect.anything(),
      }),
    )
  })

  it('increments failed_total with stage and reason from RelayError', async () => {
    mockCtx.pendingMessages = [baseMessage]
    vi.mocked(mockCtx.relayMessage).mockRejectedValueOnce(
      new RelayError({
        stage: 'simulation',
        reason: 'already_relayed',
      }),
    )

    await module.run(mockCtx)

    const output = await scrape()
    expect(output).toMatch(
      /relayer_module_relay_attempt_failed_total\{.*stage="simulation".*reason="already_relayed".*\} 1/,
    )
  })

  it('falls back to stage=broadcast, reason=unknown for non-RelayError throws', async () => {
    mockCtx.pendingMessages = [baseMessage]
    vi.mocked(mockCtx.relayMessage).mockRejectedValueOnce(new Error('boom'))

    await module.run(mockCtx)

    const output = await scrape()
    expect(output).toMatch(
      /relayer_module_relay_attempt_failed_total\{.*stage="broadcast".*reason="unknown".*\} 1/,
    )
  })

  it('increments retry_total on the second attempt after a prior failure', async () => {
    mockCtx.pendingMessages = [baseMessage]
    vi.mocked(mockCtx.relayMessage)
      .mockRejectedValueOnce(
        new RelayError({ stage: 'simulation', reason: 'unknown' }),
      )
      .mockResolvedValueOnce('0xtxhash')

    await module.run(mockCtx) // cycle 1: fails
    await module.run(mockCtx) // cycle 2: retries and succeeds

    const output = await scrape()
    expect(output).toMatch(/relayer_module_relay_attempt_retry_total\{.*\} 1/)
    expect(output).toMatch(/relayer_module_relay_tx_broadcast_total\{.*\} 1/)
    expect(output).toMatch(
      /relayer_module_relay_attempt_failed_total\{.*reason="unknown".*\} 1/,
    )
  })

  it('does not increment retry_total on the first attempt', async () => {
    mockCtx.pendingMessages = [baseMessage]

    await module.run(mockCtx)

    const output = await scrape()
    expect(output).not.toMatch(
      /relayer_module_relay_attempt_retry_total\{.*\} [1-9]/,
    )
  })

  // Decision A: ownership is generic, not hard-coded to the bridge. A message
  // whose sender is claimed by *any* other module is skipped; an unclaimed
  // Promise/Callback-shaped message is relayed (the delivery leg).
  describe('sender ownership (Decision A)', () => {
    it('skips a message from any claimed sender, not just the bridge', async () => {
      const claimedSender = '0x4444444444444444444444444444444444444444'
      const owned = new GeneralRelayModule(
        new RelayFailureRegistry(':memory:'),
        new Set([claimedSender]),
      )
      mockCtx.pendingMessages = [{ ...baseMessage, sender: claimedSender }]

      const result = await owned.run(mockCtx)

      expect(result.noMatch).toBe(1)
      expect(result.relayed).toBe(0)
      expect(mockCtx.relayMessage).not.toHaveBeenCalled()
    })

    it('relays an unclaimed sender (e.g. the Promise contract delivery leg)', async () => {
      const promiseContract = '0xf94c51c00a72a92e8f31ee08e3b93cab24fdf304'
      // Module claims only the bridge; the Promise sender is unclaimed.
      mockCtx.pendingMessages = [
        {
          ...baseMessage,
          sender: promiseContract,
          // Promise.shareResolvedPromise selector
          message: '0x5d8d7b8d',
        },
      ]

      const result = await module.run(mockCtx)

      expect(result.relayed).toBe(1)
      expect(result.noMatch).toBe(0)
      expect(mockCtx.relayMessage).toHaveBeenCalledTimes(1)
    })
  })

  describe('async execution confirmation', () => {
    let publicClient: { waitForTransactionReceipt: ReturnType<typeof vi.fn> }

    beforeEach(() => {
      publicClient = {
        waitForTransactionReceipt: vi.fn(),
      }
      vi.mocked(mockCtx.clients.getPublicClient).mockReturnValue(
        publicClient as any,
      )
    })

    it('increments executed_total when receipt confirms success', async () => {
      mockCtx.pendingMessages = [baseMessage]
      publicClient.waitForTransactionReceipt.mockResolvedValue({
        status: 'success',
      })

      await module.run(mockCtx)
      await module.awaitPendingConfirmations()

      const output = await scrape()
      expect(output).toMatch(/relayer_module_relay_tx_executed_total\{.*\} 1/)
      expect(output).toMatch(
        /relayer_module_relay_attempt_duration_seconds_count\{.*outcome="executed".*\} 1/,
      )
    })

    it('increments failed_total{stage=execution, reason=reverted} when receipt reverts', async () => {
      mockCtx.pendingMessages = [baseMessage]
      publicClient.waitForTransactionReceipt.mockResolvedValue({
        status: 'reverted',
      })

      await module.run(mockCtx)
      await module.awaitPendingConfirmations()

      const output = await scrape()
      expect(output).toMatch(
        /relayer_module_relay_attempt_failed_total\{.*stage="execution".*reason="reverted".*\} 1/,
      )
      expect(output).toMatch(
        /relayer_module_relay_attempt_duration_seconds_count\{.*outcome="failed".*\} 1/,
      )
    })

    it('increments failed_total{stage=execution, reason=unknown} when waitForReceipt throws', async () => {
      mockCtx.pendingMessages = [baseMessage]
      publicClient.waitForTransactionReceipt.mockRejectedValue(
        new Error('RPC timeout'),
      )

      await module.run(mockCtx)
      await module.awaitPendingConfirmations()

      const output = await scrape()
      expect(output).toMatch(
        /relayer_module_relay_attempt_failed_total\{.*stage="execution".*reason="unknown".*\} 1/,
      )
    })

    it('observes attempt_duration_seconds with outcome=failed on sync broadcast failure', async () => {
      mockCtx.pendingMessages = [baseMessage]
      vi.mocked(mockCtx.relayMessage).mockRejectedValueOnce(
        new RelayError({ stage: 'simulation', reason: 'unknown' }),
      )

      await module.run(mockCtx)

      const output = await scrape()
      expect(output).toMatch(
        /relayer_module_relay_attempt_duration_seconds_count\{.*outcome="failed".*\} 1/,
      )
    })
  })
})
