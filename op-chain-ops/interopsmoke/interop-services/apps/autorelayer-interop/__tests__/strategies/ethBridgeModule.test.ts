import type {
  PonderInteropClient,
  SentMessage,
} from '@eth-optimism/ponder-interop/client'
import { contracts } from '@eth-optimism/viem'
import { Registry } from 'prom-client'
import type { Account, WalletClient } from 'viem'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { RelayFundsDeposited } from '@/deposit/relayFundsDeposited.js'
import { RelayError } from '@/errors.js'
import { RelayerMetrics } from '@/metrics.js'
import { RelayFailureRegistry } from '@/relay/relayFailureRegistry.js'
import type { ClientManager } from '@/services/clientManager.js'
import { EthBridgeModule } from '@/strategies/ethBridgeModule.js'
import type { RunContext } from '@/strategies/types.js'

// Mock encodeAccessList since the module calls it directly
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
const DEFAULT_RELAYER_EOA = '0x1234567890123456789012345678901234567890'

function makeWallet(address = DEFAULT_RELAYER_EOA): {
  account: Account
  walletClient: WalletClient
} {
  return {
    account: { address: address as `0x${string}`, type: 'json-rpc' },
    walletClient: {} as WalletClient,
  }
}

/**
 * Set the happy-path mocks for ClientManager: a single public client and a
 * single resolved wallet. Tests that need richer wiring (multiple wallets,
 * empty results) override after calling this.
 *
 * `gasPrice` defaults to 1n (1 wei per gas) so existing assertions that
 * compare consumption to `200_000n` still match (200_000 × 1 = 200_000).
 * Tests that exercise wei semantics override to a realistic gwei value.
 */
function setupClientMocks(
  ctx: RunContext,
  wallets: Array<{ account: Account; walletClient: WalletClient }> = [
    makeWallet(),
  ],
  gasPrice: bigint = 1n,
): void {
  vi.mocked(ctx.clients.getPublicClient).mockReturnValue({
    getGasPrice: vi.fn().mockResolvedValue(gasPrice),
  } as any)
  vi.mocked(ctx.clients.getWalletClients).mockReturnValue(
    wallets.map((w) => w.walletClient),
  )
  vi.mocked(ctx.clients.resolveWallets).mockResolvedValue(wallets)
}

describe('EthBridgeModule', () => {
  let module: EthBridgeModule
  let mockCtx: RunContext
  let mockLogger: MockLogger
  let mockRelayFunds: RelayFundsDeposited

  const mockSentMessage: SentMessage = {
    messageIdentifierHash: '0xabcd',
    messageHash: '0x1234',
    source: 1,
    destination: 901,
    nonce: 1,
    sender: SUPERCHAIN_ETH_BRIDGE,
    target: '0x2222222222222222222222222222222222222222',
    message: '0x',
    logIndex: 5,
    logPayload: '0x5678',
    timestamp: 1234567890,
    blockNumber: 100,
    transactionHash:
      '0x9999999999999999999999999999999999999999999999999999999999999999',
    txOrigin: '0x3333333333333333333333333333333333333333',
  }

  beforeEach(() => {
    mockLogger = {
      child: vi.fn().mockReturnThis(),
      info: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
      debug: vi.fn(),
    }

    mockRelayFunds = {
      recordConsumption: vi.fn(),
      getConsumed: vi.fn().mockReturnValue(0n),
      getRemainingBudget: vi.fn().mockReturnValue(1000000n),
      hasEnoughBudget: vi.fn().mockReturnValue(true),
      markBlocked: vi.fn(),
      getBlockedForOrigin: vi.fn().mockReturnValue([]),
      clearBlocked: vi.fn(),
      close: vi.fn(),
    } as unknown as RelayFundsDeposited

    mockCtx = {
      ponderClient: {
        getPendingMessages: vi.fn(),
        getDepositBalance: vi.fn().mockResolvedValue({
          depositor: '0x3333333333333333333333333333333333333333',
          totalBalance: '1000000',
          eligible: true,
        }),
      } as unknown as PonderInteropClient,
      clients: {
        getPublicClient: vi.fn(),
        getWalletClients: vi.fn().mockReturnValue([]),
        resolveWallets: vi.fn().mockResolvedValue([]),
      } as unknown as ClientManager,
      log: mockLogger as unknown as RunContext['log'],
      metrics: new RelayerMetrics(new Registry()),
      relayMessage: vi.fn(),
    }

    module = new EthBridgeModule(
      mockRelayFunds,
      new RelayFailureRegistry(':memory:'),
    )
  })

  it('should have name "eth-bridge"', () => {
    expect(module.name).toBe('eth-bridge')
  })

  it('should relay messages provided via ctx.pendingMessages', async () => {
    mockCtx.pendingMessages = [mockSentMessage]
    setupClientMocks(mockCtx)
    vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

    const result = await module.run(mockCtx)

    expect(result).toEqual({ relayed: 1, skipped: 0, failed: 0, noMatch: 0 })
  })

  it('should skip messages when getPublicClient returns undefined', async () => {
    mockCtx.pendingMessages = [mockSentMessage]
    setupClientMocks(mockCtx)
    vi.mocked(mockCtx.clients.getPublicClient).mockReturnValue(undefined)

    const result = await module.run(mockCtx)

    expect(result).toEqual({ relayed: 0, skipped: 1, failed: 0, noMatch: 0 })
    expect(mockLogger.warn).toHaveBeenCalledWith(
      'no client for destination, skipping...',
    )
  })

  it('should skip messages when getWalletClients returns empty', async () => {
    mockCtx.pendingMessages = [mockSentMessage]
    setupClientMocks(mockCtx)
    vi.mocked(mockCtx.clients.getWalletClients).mockReturnValue([])

    const result = await module.run(mockCtx)

    expect(result).toEqual({ relayed: 0, skipped: 1, failed: 0, noMatch: 0 })
    expect(mockLogger.warn).toHaveBeenCalledWith(
      'no client for destination, skipping...',
    )
  })

  it('should skip messages when resolveWallets returns empty', async () => {
    mockCtx.pendingMessages = [mockSentMessage]
    setupClientMocks(mockCtx)
    // walletClients is non-empty (so we pass the no_client gate) but
    // resolveWallets yields no usable accounts (e.g. sponsored endpoint
    // with empty getAddresses() result).
    vi.mocked(mockCtx.clients.resolveWallets).mockResolvedValue([])

    const result = await module.run(mockCtx)

    expect(result).toEqual({ relayed: 0, skipped: 1, failed: 0, noMatch: 0 })
    expect(mockLogger.warn).toHaveBeenCalledWith(
      'no accounts found, skipping...',
    )
  })

  it('should catch validation errors and count them as a skip', async () => {
    mockCtx.pendingMessages = [mockSentMessage]
    setupClientMocks(mockCtx)
    vi.mocked(mockCtx.ponderClient.getDepositBalance).mockResolvedValue({
      depositor: '0x3333333333333333333333333333333333333333',
      totalBalance: '0',
      eligible: false,
    })

    const result = await module.run(mockCtx)

    expect(result).toEqual({ relayed: 0, skipped: 1, failed: 0, noMatch: 0 })
    expect(mockLogger.warn).toHaveBeenCalledWith(
      expect.objectContaining({ err: expect.any(Error) }),
      expect.stringContaining('insufficient deposit'),
    )
  })

  it('should correctly build MessageIdentifier', async () => {
    mockCtx.pendingMessages = [mockSentMessage]
    setupClientMocks(mockCtx)
    vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

    await module.run(mockCtx)

    expect(mockCtx.relayMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        id: {
          origin: contracts.l2ToL2CrossDomainMessenger.address,
          chainId: BigInt(mockSentMessage.source),
          logIndex: BigInt(mockSentMessage.logIndex),
          blockNumber: BigInt(mockSentMessage.blockNumber),
          timestamp: BigInt(mockSentMessage.timestamp),
        },
      }),
    )
  })

  it('should call relayMessage with correct RelayMessageParams', async () => {
    mockCtx.pendingMessages = [mockSentMessage]
    const wallet = makeWallet()
    setupClientMocks(mockCtx, [wallet])
    vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

    await module.run(mockCtx)

    expect(mockCtx.relayMessage).toHaveBeenCalledWith({
      id: expect.objectContaining({
        origin: contracts.l2ToL2CrossDomainMessenger.address,
      }),
      destinationChainId: mockSentMessage.destination,
      payload: mockSentMessage.logPayload,
      accessList: expect.anything(),
      account: wallet.account,
      walletClient: wallet.walletClient,
      chain: null,
      txOrigin: mockSentMessage.txOrigin.toLowerCase(),
      messageHash: mockSentMessage.messageHash,
      estimatedGasCost: 200_000n,
    })
  })

  it('passes the configured gas budget to relayMessage', async () => {
    module = new EthBridgeModule(
      mockRelayFunds,
      new RelayFailureRegistry(':memory:'),
      250_000n,
    )
    mockCtx.pendingMessages = [mockSentMessage]
    setupClientMocks(mockCtx)
    vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

    await module.run(mockCtx)

    expect(mockCtx.relayMessage).toHaveBeenCalledWith(
      expect.objectContaining({ estimatedGasCost: 250_000n }),
    )
  })

  it('uses the account/walletClient pair returned by resolveWallets', async () => {
    mockCtx.pendingMessages = [mockSentMessage]
    const sponsoredWallet = makeWallet(
      '0x5555555555555555555555555555555555555555',
    )
    setupClientMocks(mockCtx, [sponsoredWallet])
    vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

    await module.run(mockCtx)

    expect(mockCtx.clients.resolveWallets).toHaveBeenCalledWith(
      mockSentMessage.destination,
    )
    expect(mockCtx.relayMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        account: sponsoredWallet.account,
        walletClient: sponsoredWallet.walletClient,
      }),
    )
  })

  // --- Sender filtering ---
  describe('sender filtering', () => {
    it('should process messages with sender == SuperchainETHBridge', async () => {
      mockCtx.pendingMessages = [mockSentMessage]
      setupClientMocks(mockCtx)
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

      const result = await module.run(mockCtx)

      expect(result.relayed).toBe(1)
      expect(result.noMatch).toBe(0)
      expect(mockCtx.relayMessage).toHaveBeenCalled()
    })

    it('should increment noMatch for sender != SuperchainETHBridge and NOT call relayMessage', async () => {
      const nonBridgeMessage: SentMessage = {
        ...mockSentMessage,
        sender: '0x1111111111111111111111111111111111111111',
      }
      mockCtx.pendingMessages = [nonBridgeMessage]

      const result = await module.run(mockCtx)

      expect(result).toEqual({ relayed: 0, skipped: 0, failed: 0, noMatch: 1 })
      expect(mockCtx.relayMessage).not.toHaveBeenCalled()
    })

    it('should log noMatch at debug level (not warn)', async () => {
      const nonBridgeMessage: SentMessage = {
        ...mockSentMessage,
        sender: '0x1111111111111111111111111111111111111111',
      }
      mockCtx.pendingMessages = [nonBridgeMessage]

      await module.run(mockCtx)

      expect(mockLogger.debug).toHaveBeenCalledWith(
        'sender is not SuperchainETHBridge, skipping (no-match)',
      )
      expect(mockLogger.warn).not.toHaveBeenCalledWith(
        expect.stringContaining('no-match'),
      )
    })

    it('should handle mixed batch: 2 ETH bridge + 1 non-ETH bridge', async () => {
      const nonBridgeMessage: SentMessage = {
        ...mockSentMessage,
        sender: '0x1111111111111111111111111111111111111111',
        messageHash: '0xdead',
      }
      mockCtx.pendingMessages = [
        mockSentMessage,
        { ...mockSentMessage, messageHash: '0x5678' },
        nonBridgeMessage,
      ]
      setupClientMocks(mockCtx)
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

      const result = await module.run(mockCtx)

      expect(result).toEqual({ relayed: 2, skipped: 0, failed: 0, noMatch: 1 })
    })
  })

  // --- Deposit gate ---
  describe('deposit gate', () => {
    it('should relay when depositor has totalBalance > 0 and sufficient budget', async () => {
      mockCtx.pendingMessages = [mockSentMessage]
      setupClientMocks(mockCtx)
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

      const result = await module.run(mockCtx)

      expect(result.relayed).toBe(1)
      expect(result.failed).toBe(0)
    })

    it('should skip when depositor has totalBalance == 0', async () => {
      mockCtx.pendingMessages = [mockSentMessage]
      setupClientMocks(mockCtx)
      vi.mocked(mockCtx.ponderClient.getDepositBalance).mockResolvedValue({
        depositor: '0x3333333333333333333333333333333333333333',
        totalBalance: '0',
        eligible: false,
      })

      const result = await module.run(mockCtx)

      expect(result).toEqual({ relayed: 0, skipped: 1, failed: 0, noMatch: 0 })
      expect(mockCtx.relayMessage).not.toHaveBeenCalled()
    })

    it('should treat ponder getDepositBalance failure as 0 balance and skip', async () => {
      mockCtx.pendingMessages = [mockSentMessage]
      setupClientMocks(mockCtx)
      vi.mocked(mockCtx.ponderClient.getDepositBalance).mockRejectedValue(
        new Error('Not found'),
      )

      const result = await module.run(mockCtx)

      expect(result).toEqual({ relayed: 0, skipped: 1, failed: 0, noMatch: 0 })
      expect(mockCtx.relayMessage).not.toHaveBeenCalled()
    })
  })

  // --- Budget tracking ---
  describe('budget tracking', () => {
    it('should call recordConsumption after successful relay', async () => {
      mockCtx.pendingMessages = [mockSentMessage]
      setupClientMocks(mockCtx)
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

      await module.run(mockCtx)

      expect(mockRelayFunds.recordConsumption).toHaveBeenCalledWith(
        mockSentMessage.txOrigin.toLowerCase(),
        200_000n,
      )
    })

    it('should relay when remaining budget exactly equals estimatedGasCost', async () => {
      mockCtx.pendingMessages = [mockSentMessage]
      setupClientMocks(mockCtx)
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')
      vi.mocked(mockRelayFunds.hasEnoughBudget).mockReturnValue(true)

      const result = await module.run(mockCtx)

      expect(result.relayed).toBe(1)
    })
  })

  // --- Wei-semantics budget enforcement ---
  // The budget gate must compare deposit balance (wei) against
  // estimatedGasUnits × gasPrice (wei), not raw gas units.
  describe('budget enforcement uses wei × gas semantics', () => {
    it('passes estimatedGasUnits × gasPrice in wei to hasEnoughBudget', async () => {
      mockCtx.pendingMessages = [mockSentMessage]
      // 10 gwei gas price
      setupClientMocks(mockCtx, [makeWallet()], 10_000_000_000n)
      vi.mocked(mockCtx.ponderClient.getDepositBalance).mockResolvedValue({
        depositor: mockSentMessage.txOrigin,
        totalBalance: '10000000000000000000', // 10 ETH
        eligible: true,
      })
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

      await module.run(mockCtx)

      // 200_000 gas × 10 gwei = 2 × 10^15 wei
      expect(mockRelayFunds.hasEnoughBudget).toHaveBeenCalledWith(
        mockSentMessage.txOrigin.toLowerCase(),
        10_000_000_000_000_000_000n,
        2_000_000_000_000_000n,
      )
    })

    it('records consumption in wei (gasUnits × gasPrice) on successful relay', async () => {
      mockCtx.pendingMessages = [mockSentMessage]
      setupClientMocks(mockCtx, [makeWallet()], 10_000_000_000n)
      vi.mocked(mockCtx.ponderClient.getDepositBalance).mockResolvedValue({
        depositor: mockSentMessage.txOrigin,
        totalBalance: '10000000000000000000',
        eligible: true,
      })
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

      await module.run(mockCtx)

      // 200_000 gas × 10 gwei = 2 × 10^15 wei
      expect(mockRelayFunds.recordConsumption).toHaveBeenCalledWith(
        mockSentMessage.txOrigin.toLowerCase(),
        2_000_000_000_000_000n,
      )
    })

    it('hard-blocks the second message when same tx.origin and budget covers only one', async () => {
      const m1 = { ...mockSentMessage, messageHash: '0xfeed01' }
      const m2 = { ...mockSentMessage, messageHash: '0xfeed02' }
      mockCtx.pendingMessages = [m1, m2]
      setupClientMocks(mockCtx, [makeWallet()], 10_000_000_000n) // 10 gwei
      // 200_000 × 10 gwei = 2×10^15 wei per relay; deposit covers 1.25.
      vi.mocked(mockCtx.ponderClient.getDepositBalance).mockResolvedValue({
        depositor: mockSentMessage.txOrigin,
        totalBalance: '2500000000000000',
        eligible: true,
      })
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')
      // Faithful budget check so reservations actually drive the outcome.
      vi.mocked(mockRelayFunds.hasEnoughBudget).mockImplementation(
        (_origin, deposit, cost) => deposit >= cost,
      )

      const result = await module.run(mockCtx)

      expect(result.relayed).toBe(1)
      expect(result.skipped).toBe(1)
      expect(mockCtx.relayMessage).toHaveBeenCalledTimes(1)
      expect(mockRelayFunds.markBlocked).toHaveBeenCalledWith(
        '0xfeed02',
        mockSentMessage.txOrigin.toLowerCase(),
      )
      expect(mockRelayFunds.markBlocked).not.toHaveBeenCalledWith(
        '0xfeed01',
        expect.anything(),
      )
    })

    it('does not let one origin starve a different origin within the same run', async () => {
      const originA = '0xaaaa000000000000000000000000000000000000'
      const originB = '0xbbbb000000000000000000000000000000000000'
      const mA = {
        ...mockSentMessage,
        messageHash: '0xa1',
        txOrigin: originA,
      }
      const mB = {
        ...mockSentMessage,
        messageHash: '0xb1',
        txOrigin: originB,
      }
      mockCtx.pendingMessages = [mA, mB]
      setupClientMocks(mockCtx, [makeWallet()], 10_000_000_000n)
      // Each origin has exactly one relay's worth of deposit.
      vi.mocked(mockCtx.ponderClient.getDepositBalance).mockResolvedValue({
        depositor: '0x0',
        totalBalance: '2000000000000000',
        eligible: true,
      })
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')
      vi.mocked(mockRelayFunds.hasEnoughBudget).mockImplementation(
        (_origin, deposit, cost) => deposit >= cost,
      )

      const result = await module.run(mockCtx)

      expect(result.relayed).toBe(2)
      expect(result.skipped).toBe(0)
      expect(mockRelayFunds.markBlocked).not.toHaveBeenCalled()
    })

    it('skips with reason no_gas_price when getGasPrice fails for destination', async () => {
      mockCtx.pendingMessages = [mockSentMessage]
      const wallet = makeWallet()
      vi.mocked(mockCtx.clients.getPublicClient).mockReturnValue({
        getGasPrice: vi.fn().mockRejectedValue(new Error('rpc unavailable')),
      } as any)
      vi.mocked(mockCtx.clients.getWalletClients).mockReturnValue([
        wallet.walletClient,
      ])
      vi.mocked(mockCtx.clients.resolveWallets).mockResolvedValue([wallet])

      const result = await module.run(mockCtx)

      expect(result).toEqual({ relayed: 0, skipped: 1, failed: 0, noMatch: 0 })
      expect(mockCtx.relayMessage).not.toHaveBeenCalled()
      expect(mockRelayFunds.markBlocked).not.toHaveBeenCalled()
    })
  })

  // --- Deposit balance caching ---
  describe('deposit cache', () => {
    it('should call getDepositBalance only once for two messages from same txOrigin', async () => {
      const messages = [
        { ...mockSentMessage, messageHash: '0x1111' },
        { ...mockSentMessage, messageHash: '0x2222' },
      ]
      mockCtx.pendingMessages = messages
      setupClientMocks(mockCtx)
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

      await module.run(mockCtx)

      expect(mockCtx.ponderClient.getDepositBalance).toHaveBeenCalledTimes(1)
    })

    it('should call getDepositBalance once per unique txOrigin', async () => {
      const messages = [
        {
          ...mockSentMessage,
          txOrigin: '0xaaaa000000000000000000000000000000000000',
          messageHash: '0x1111',
        },
        {
          ...mockSentMessage,
          txOrigin: '0xbbbb000000000000000000000000000000000000',
          messageHash: '0x2222',
        },
      ]
      mockCtx.pendingMessages = messages
      setupClientMocks(mockCtx)
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

      await module.run(mockCtx)

      expect(mockCtx.ponderClient.getDepositBalance).toHaveBeenCalledTimes(2)
    })
  })

  // --- Double-relay prevention ---
  describe('double-relay prevention', () => {
    it('should not call recordConsumption when relayMessage throws (simulation failure)', async () => {
      mockCtx.pendingMessages = [mockSentMessage]
      setupClientMocks(mockCtx)
      vi.mocked(mockCtx.relayMessage).mockRejectedValue(
        new Error('Simulation failed'),
      )

      const result = await module.run(mockCtx)

      expect(result.failed).toBe(1)
      expect(mockRelayFunds.recordConsumption).not.toHaveBeenCalled()
    })
  })

  // --- Edge cases ---
  describe('edge cases', () => {
    it('should return empty result for empty pending messages list', async () => {
      mockCtx.pendingMessages = []

      const result = await module.run(mockCtx)

      expect(result).toEqual({ relayed: 0, skipped: 0, failed: 0, noMatch: 0 })
    })
  })

  // --- Funnel counters ---
  describe('funnel counters', () => {
    let registry: Registry

    beforeEach(() => {
      registry = new Registry()
      mockCtx.metrics = new RelayerMetrics(registry)
      setupClientMocks(mockCtx)
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')
    })

    async function scrape(): Promise<string> {
      return await registry.metrics()
    }

    it('classifies non-SuperchainETHBridge messages as no-match (general-relay owns them)', async () => {
      mockCtx.pendingMessages = [
        {
          ...mockSentMessage,
          sender: '0xdeadbeef00000000000000000000000000000000',
        },
      ]

      const result = await module.run(mockCtx)

      expect(result.noMatch).toBe(1)
      expect(result.relayed).toBe(0)
      expect(await scrape()).not.toMatch(
        /relayer_module_relay_tx_broadcast_total\{.*\} [1-9]/,
      )
    })

    it('increments skipped_total{reason=no_client} when client missing', async () => {
      mockCtx.pendingMessages = [mockSentMessage]
      vi.mocked(mockCtx.clients.getPublicClient).mockReturnValue(undefined)

      await module.run(mockCtx)

      expect(await scrape()).toMatch(
        /relayer_module_message_skipped_total\{.*reason="no_client".*\} 1/,
      )
    })

    it('increments skipped_total{reason=no_account} when resolveWallets returns empty', async () => {
      mockCtx.pendingMessages = [mockSentMessage]
      vi.mocked(mockCtx.clients.resolveWallets).mockResolvedValue([])

      await module.run(mockCtx)

      expect(await scrape()).toMatch(
        /relayer_module_message_skipped_total\{.*reason="no_account".*\} 1/,
      )
    })

    it('increments skipped_total{reason=insufficient_deposit_bal} when depositor has zero balance', async () => {
      mockCtx.pendingMessages = [mockSentMessage]
      vi.mocked(mockCtx.ponderClient.getDepositBalance).mockResolvedValue({
        depositor: mockSentMessage.txOrigin,
        totalBalance: '0',
        eligible: false,
      })

      await module.run(mockCtx)

      expect(await scrape()).toMatch(
        /relayer_module_message_skipped_total\{.*reason="insufficient_deposit_bal".*\} 1/,
      )
    })

    it('hard-blocks and persists blocked state when budget exhausted, does not broadcast', async () => {
      mockCtx.pendingMessages = [mockSentMessage]
      vi.mocked(mockRelayFunds.hasEnoughBudget).mockReturnValue(false)

      const result = await module.run(mockCtx)

      expect(result).toEqual({ relayed: 0, skipped: 1, failed: 0, noMatch: 0 })
      expect(mockCtx.relayMessage).not.toHaveBeenCalled()
      expect(mockRelayFunds.markBlocked).toHaveBeenCalledWith(
        mockSentMessage.messageHash,
        mockSentMessage.txOrigin.toLowerCase(),
      )
      // budget-blocked also funnels into insufficient_deposit_bal — same
      // operator question ("this user can't pay") from a different internal path
      expect(await scrape()).toMatch(
        /relayer_module_message_skipped_total\{.*reason="insufficient_deposit_bal".*\} 1/,
      )
    })

    it('increments broadcast_total with relayer_eoa label on successful relay', async () => {
      mockCtx.pendingMessages = [mockSentMessage]

      await module.run(mockCtx)

      expect(await scrape()).toContain(
        `relayer_module_relay_tx_broadcast_total{module="eth-bridge",src="1",dst="901",relayer_eoa="${DEFAULT_RELAYER_EOA.toLowerCase()}"} 1`,
      )
    })

    it('increments failed_total with stage/reason from RelayError', async () => {
      mockCtx.pendingMessages = [mockSentMessage]
      vi.mocked(mockCtx.relayMessage).mockRejectedValueOnce(
        new RelayError({ stage: 'simulation', reason: 'already_relayed' }),
      )

      await module.run(mockCtx)

      expect(await scrape()).toMatch(
        /relayer_module_relay_attempt_failed_total\{.*stage="simulation".*reason="already_relayed".*\} 1/,
      )
    })
  })

  // --- Session dedup (inherited from BaseRelayModule) ---
  describe('session dedup', () => {
    it('skips messages already submitted this session', async () => {
      mockCtx.pendingMessages = [mockSentMessage]
      setupClientMocks(mockCtx)
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

      // First run: relays successfully, records hash
      const firstResult = await module.run(mockCtx)
      expect(firstResult).toEqual({
        relayed: 1,
        skipped: 0,
        failed: 0,
        noMatch: 0,
      })

      // Second run (same message still in ctx -- Ponder hasn't indexed yet)
      // should skip via dedup rather than re-submit
      const secondResult = await module.run(mockCtx)
      expect(secondResult).toEqual({
        relayed: 0,
        skipped: 1,
        failed: 0,
        noMatch: 0,
      })
      expect(mockCtx.relayMessage).toHaveBeenCalledTimes(1)
    })
  })

  // --- Multi-EOA parallel broadcast ---
  describe('multi-EOA parallel broadcast', () => {
    let registry: Registry

    beforeEach(() => {
      registry = new Registry()
      mockCtx.metrics = new RelayerMetrics(registry)
    })

    async function scrape(): Promise<string> {
      return await registry.metrics()
    }

    it('distributes 3 ready messages across 2 wallets, both wallets used', async () => {
      const wallets = [
        makeWallet('0xaaaa000000000000000000000000000000000001'),
        makeWallet('0xbbbb000000000000000000000000000000000002'),
      ]
      mockCtx.pendingMessages = [
        { ...mockSentMessage, messageHash: '0x1111' },
        { ...mockSentMessage, messageHash: '0x2222' },
        { ...mockSentMessage, messageHash: '0x3333' },
      ]
      setupClientMocks(mockCtx, wallets)
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

      const result = await module.run(mockCtx)

      expect(result.relayed).toBe(3)
      expect(mockCtx.relayMessage).toHaveBeenCalledTimes(3)

      const calls = vi.mocked(mockCtx.relayMessage).mock.calls
      const accountsUsed = new Set(
        calls.map((c) => (c[0] as { account: Account }).account.address),
      )
      expect(accountsUsed.size).toBe(2)
      expect(accountsUsed.has(wallets[0].account.address)).toBe(true)
      expect(accountsUsed.has(wallets[1].account.address)).toBe(true)
    })

    it('emits broadcast_total per relayer_eoa when multiple wallets relay', async () => {
      const wallets = [
        makeWallet('0xaaaa000000000000000000000000000000000001'),
        makeWallet('0xbbbb000000000000000000000000000000000002'),
      ]
      mockCtx.pendingMessages = [
        { ...mockSentMessage, messageHash: '0x1111' },
        { ...mockSentMessage, messageHash: '0x2222' },
      ]
      setupClientMocks(mockCtx, wallets)
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

      await module.run(mockCtx)

      const output = await scrape()
      expect(output).toContain(
        `relayer_module_relay_tx_broadcast_total{module="eth-bridge",src="1",dst="901",relayer_eoa="${wallets[0].account.address.toLowerCase()}"} 1`,
      )
      expect(output).toContain(
        `relayer_module_relay_tx_broadcast_total{module="eth-bridge",src="1",dst="901",relayer_eoa="${wallets[1].account.address.toLowerCase()}"} 1`,
      )
    })

    it('drains queue when more relays than wallets are available', async () => {
      const wallet = makeWallet()
      mockCtx.pendingMessages = [
        { ...mockSentMessage, messageHash: '0x1111' },
        { ...mockSentMessage, messageHash: '0x2222' },
        { ...mockSentMessage, messageHash: '0x3333' },
        { ...mockSentMessage, messageHash: '0x4444' },
      ]
      setupClientMocks(mockCtx, [wallet])
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

      const result = await module.run(mockCtx)

      expect(result.relayed).toBe(4)
      expect(mockCtx.relayMessage).toHaveBeenCalledTimes(4)
    })

    it('continues with remaining relays when one relay fails on a single wallet', async () => {
      const wallet = makeWallet()
      mockCtx.pendingMessages = [
        { ...mockSentMessage, messageHash: '0x1111' },
        { ...mockSentMessage, messageHash: '0x2222' },
        { ...mockSentMessage, messageHash: '0x3333' },
      ]
      setupClientMocks(mockCtx, [wallet])
      vi.mocked(mockCtx.relayMessage)
        .mockResolvedValueOnce('0xtxhash1')
        .mockRejectedValueOnce(
          new RelayError({ stage: 'broadcast', reason: 'unknown' }),
        )
        .mockResolvedValueOnce('0xtxhash3')

      const result = await module.run(mockCtx)

      expect(result.relayed).toBe(2)
      expect(result.failed).toBe(1)
    })

    it('resolves wallets only once per cycle even with many messages', async () => {
      const wallets = [makeWallet('0xaaaa000000000000000000000000000000000001')]
      mockCtx.pendingMessages = [
        { ...mockSentMessage, messageHash: '0x1111' },
        { ...mockSentMessage, messageHash: '0x2222' },
        { ...mockSentMessage, messageHash: '0x3333' },
        { ...mockSentMessage, messageHash: '0x4444' },
        { ...mockSentMessage, messageHash: '0x5555' },
      ]
      setupClientMocks(mockCtx, wallets)
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

      await module.run(mockCtx)

      expect(mockCtx.clients.resolveWallets).toHaveBeenCalledTimes(1)
    })
  })

  describe('reverted relay tx (R10)', () => {
    it('releases session dedup and records a registry failure when the relay tx reverts on-chain', async () => {
      const failureRegistry = new RelayFailureRegistry(':memory:')
      module = new EthBridgeModule(mockRelayFunds, failureRegistry)

      mockCtx.pendingMessages = [mockSentMessage]
      setupClientMocks(mockCtx)
      // Public client whose receipt wait reports an on-chain revert.
      vi.mocked(mockCtx.clients.getPublicClient).mockReturnValue({
        getGasPrice: vi.fn().mockResolvedValue(1n),
        waitForTransactionReceipt: vi
          .fn()
          .mockResolvedValue({ status: 'reverted' }),
      } as any)
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

      // Cycle 1: broadcast succeeds, then the tx reverts during confirmation.
      await module.run(mockCtx)
      await module.awaitPendingConfirmations()

      // The revert must be recorded in the durable failure registry...
      expect(failureRegistry.hasFailed(mockSentMessage.messageHash)).toBe(true)

      // ...and the session dedup must be released: cycle 2 skips via the
      // failure registry (in_backoff — visible in metrics), not via the
      // silent submittedAt dedup that would park the message until restart.
      const registry = new Registry()
      mockCtx.metrics = new RelayerMetrics(registry)
      mockCtx.pendingMessages = [mockSentMessage]
      const result = await module.run(mockCtx)
      expect(result.relayed).toBe(0)
      const output = await registry.metrics()
      expect(output).toMatch(
        /relayer_module_message_skipped_total\{module="eth-bridge",src="1",dst="901",relayer_eoa="",reason="in_backoff"\} 1/,
      )
    })
  })

  describe('circuit breaker', () => {
    it('skips a message that the failure registry has marked permanent', async () => {
      const failureRegistry = new RelayFailureRegistry(':memory:')
      // Pre-mark this hash as permanent (simulates a prior cycle that
      // classified the failure as rpc_rejected).
      failureRegistry.recordFailure(
        mockSentMessage.messageHash,
        mockSentMessage.source,
        mockSentMessage.destination,
        'rpc_rejected',
      )
      module = new EthBridgeModule(mockRelayFunds, failureRegistry)

      mockCtx.pendingMessages = [mockSentMessage]
      setupClientMocks(mockCtx)

      const result = await module.run(mockCtx)

      expect(result).toEqual({ relayed: 0, skipped: 1, failed: 0, noMatch: 0 })
      expect(mockCtx.relayMessage).not.toHaveBeenCalled()
    })

    it('skips a message within the backoff window of a recent failure', async () => {
      const failureRegistry = new RelayFailureRegistry(':memory:')
      // Single transient failure with `now` = recent — backoff is 5s, so a
      // re-attempt right away should be suppressed.
      failureRegistry.recordFailure(
        mockSentMessage.messageHash,
        mockSentMessage.source,
        mockSentMessage.destination,
        'unknown',
        Date.now(),
      )
      module = new EthBridgeModule(mockRelayFunds, failureRegistry)

      mockCtx.pendingMessages = [mockSentMessage]
      setupClientMocks(mockCtx)

      const result = await module.run(mockCtx)

      expect(result.relayed).toBe(0)
      expect(result.skipped).toBe(1)
      expect(mockCtx.relayMessage).not.toHaveBeenCalled()
    })

    it('preserves failure registry entries when the pending fetch failed (ponder blip)', async () => {
      const failureRegistry = new RelayFailureRegistry(':memory:')
      failureRegistry.recordFailure(
        mockSentMessage.messageHash,
        mockSentMessage.source,
        mockSentMessage.destination,
        'unknown',
        Date.now(),
      )
      module = new EthBridgeModule(mockRelayFunds, failureRegistry)

      // A Ponder outage degrades the fetch to an empty list; the relayer marks
      // the endpoint as failed so modules know the emptiness is not real.
      mockCtx.pendingMessages = []
      mockCtx.failedEndpoints = new Set(['pendingMessages'])
      setupClientMocks(mockCtx)

      await module.run(mockCtx)

      expect(failureRegistry.hasFailed(mockSentMessage.messageHash)).toBe(true)
    })

    it('preserves session dedup across a ponder blip (no re-broadcast)', async () => {
      const failureRegistry = new RelayFailureRegistry(':memory:')
      module = new EthBridgeModule(mockRelayFunds, failureRegistry)
      setupClientMocks(mockCtx)
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

      // Cycle 1: relay the message (records session dedup).
      mockCtx.pendingMessages = [mockSentMessage]
      await module.run(mockCtx)
      expect(mockCtx.relayMessage).toHaveBeenCalledTimes(1)

      // Cycle 2: ponder blip — empty list, endpoint flagged as failed.
      mockCtx.pendingMessages = []
      mockCtx.failedEndpoints = new Set(['pendingMessages'])
      await module.run(mockCtx)

      // Cycle 3: ponder recovers, message still pending (indexer lag).
      mockCtx.pendingMessages = [mockSentMessage]
      mockCtx.failedEndpoints = undefined
      await module.run(mockCtx)

      // Still deduped — no second broadcast.
      expect(mockCtx.relayMessage).toHaveBeenCalledTimes(1)
    })

    it('never GCs the shared registry from a module run (R8: GC is the relayer core job)', async () => {
      const failureRegistry = new RelayFailureRegistry(':memory:')
      failureRegistry.recordFailure(
        mockSentMessage.messageHash,
        mockSentMessage.source,
        mockSentMessage.destination,
        'unknown',
        Date.now(),
      )
      module = new EthBridgeModule(mockRelayFunds, failureRegistry)

      // Even with a genuinely-empty pending slice, a module must not GC the
      // shared registry — it only sees its own slice, and GCing against it
      // would wipe other modules' rows. Reclaiming rows whose keys are no
      // longer pending anywhere is Relayer.gcFailureRegistry()'s job (tested
      // in relayer.test.ts).
      mockCtx.pendingMessages = []
      setupClientMocks(mockCtx)

      await module.run(mockCtx)

      expect(failureRegistry.hasFailed(mockSentMessage.messageHash)).toBe(true)
    })

    it('attempts a message whose backoff window has elapsed', async () => {
      const failureRegistry = new RelayFailureRegistry(':memory:')
      // Past failure long ago — backoff window has elapsed.
      failureRegistry.recordFailure(
        mockSentMessage.messageHash,
        mockSentMessage.source,
        mockSentMessage.destination,
        'unknown',
        Date.now() - 60_000,
      )
      module = new EthBridgeModule(mockRelayFunds, failureRegistry)

      mockCtx.pendingMessages = [mockSentMessage]
      setupClientMocks(mockCtx)
      vi.mocked(mockCtx.relayMessage).mockResolvedValue('0xtxhash')

      const result = await module.run(mockCtx)

      expect(result.relayed).toBe(1)
      expect(mockCtx.relayMessage).toHaveBeenCalled()
    })
  })
})
