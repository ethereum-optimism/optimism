import type {
  PonderInteropClient,
  PonderPromise,
} from '@eth-optimism/ponder-interop/client'
import { Registry } from 'prom-client'
import type { PublicClient, WalletClient } from 'viem'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { RelayerMetrics } from '@/metrics.js'
import { RelayFailureRegistry } from '@/relay/relayFailureRegistry.js'
import type { ClientManager } from '@/services/clientManager.js'
import { PromiseModule } from '@/strategies/promiseModule.js'
import type { RunContext } from '@/strategies/types.js'

interface MockLogger {
  child: ReturnType<typeof vi.fn>
  info: ReturnType<typeof vi.fn>
  warn: ReturnType<typeof vi.fn>
  error: ReturnType<typeof vi.fn>
  debug: ReturnType<typeof vi.fn>
}

describe('PromiseModule', () => {
  let module: PromiseModule
  let mockCtx: RunContext
  let mockLogger: MockLogger
  let mockPublicClient: PublicClient
  let mockWalletClient: WalletClient

  const mockPonderPromise: PonderPromise = {
    promiseId: '0x123',
    chainId: 1,
    resolver: '0x1234567890123456789012345678901234567890',
    status: 'pending',
    createdAt: 1700000000,
    createdBlockNumber: 1000,
    createdTransactionHash: '0xabc123',
    resolvedAt: null,
    resolvedBlockNumber: null,
    resolvedTransactionHash: null,
  }

  beforeEach(() => {
    mockLogger = {
      child: vi.fn().mockReturnThis(),
      info: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
      debug: vi.fn(),
    }

    mockPublicClient = {
      readContract: vi.fn(),
      getCode: vi.fn().mockResolvedValue('0x1234'),
      estimateContractGas: vi.fn().mockResolvedValue(100_000n),
    } as unknown as PublicClient

    mockWalletClient = {
      account: { address: '0x1234567890123456789012345678901234567890' },
      writeContract: vi.fn(),
      getAddresses: vi.fn(),
    } as unknown as WalletClient

    mockCtx = {
      ponderClient: {
        getPendingPromises: vi.fn(),
      } as unknown as PonderInteropClient,
      clients: {
        getPublicClient: vi.fn().mockReturnValue(mockPublicClient),
        getWalletClient: vi.fn().mockReturnValue(mockWalletClient),
        resolveAccount: vi.fn().mockResolvedValue(mockWalletClient.account),
      } as unknown as ClientManager,
      log: mockLogger as unknown as RunContext['log'],
      metrics: new RelayerMetrics(new Registry()),
      relayMessage: vi.fn(),
    }

    module = new PromiseModule(new RelayFailureRegistry(':memory:'))
  })

  it('should have name "promise"', () => {
    expect(module.name).toBe('promise')
  })

  it('should fetch pending promises and resolve one successfully', async () => {
    mockCtx.pendingPromises = [mockPonderPromise]
    vi.mocked(mockPublicClient.readContract).mockResolvedValue(true)
    vi.mocked(mockWalletClient.writeContract).mockResolvedValue(
      '0xresolvetxhash',
    )

    const result = await module.run(mockCtx)

    expect(result).toEqual({ relayed: 1, skipped: 0, failed: 0, noMatch: 0 })
    expect(mockLogger.info).toHaveBeenCalledWith('1 pending promises')
  })

  it('should skip promises when canResolve returns false', async () => {
    mockCtx.pendingPromises = [mockPonderPromise]
    vi.mocked(mockPublicClient.readContract).mockResolvedValue(false)

    const result = await module.run(mockCtx)

    expect(result).toEqual({ relayed: 0, skipped: 1, failed: 0, noMatch: 0 })
    expect(mockWalletClient.writeContract).not.toHaveBeenCalled()
    expect(mockLogger.debug).toHaveBeenCalledWith(
      `promise ${mockPonderPromise.promiseId} cannot be resolved yet`,
    )
  })

  it('should skip promises when no client for chain', async () => {
    const promiseWithNoClient = { ...mockPonderPromise, chainId: 999 }
    mockCtx.pendingPromises = [promiseWithNoClient]
    vi.mocked(mockCtx.clients.getPublicClient).mockReturnValue(undefined)

    const result = await module.run(mockCtx)

    expect(result).toEqual({ relayed: 0, skipped: 1, failed: 0, noMatch: 0 })
    expect(mockLogger.warn).toHaveBeenCalledWith(
      'no client for chain, skipping...',
    )
    expect(mockPublicClient.readContract).not.toHaveBeenCalled()
  })

  it('should handle canResolve readContract error gracefully (inner try/catch)', async () => {
    mockCtx.pendingPromises = [mockPonderPromise]
    vi.mocked(mockPublicClient.readContract).mockRejectedValue(
      new Error('Contract call failed'),
    )

    const result = await module.run(mockCtx)

    expect(result).toEqual({ relayed: 0, skipped: 0, failed: 1, noMatch: 0 })
    expect(mockLogger.warn).toHaveBeenCalledWith(
      expect.objectContaining({ err: expect.any(Error) }),
      'failed to check if promise can be resolved',
    )
  })

  it('should handle resolve writeContract error gracefully', async () => {
    mockCtx.pendingPromises = [mockPonderPromise]
    vi.mocked(mockPublicClient.readContract).mockResolvedValue(true)
    vi.mocked(mockWalletClient.writeContract).mockRejectedValue(
      new Error('Transaction failed'),
    )

    const result = await module.run(mockCtx)

    expect(result).toEqual({ relayed: 0, skipped: 0, failed: 1, noMatch: 0 })
    expect(mockLogger.warn).toHaveBeenCalledWith(
      expect.objectContaining({ err: expect.any(Error) }),
      expect.stringContaining('failed to process promise'),
    )
  })

  it('should use sponsored account from resolveAccount when walletClient has no direct account', async () => {
    const sponsoredAccount = {
      address: '0x7777777777777777777777777777777777777777' as const,
      type: 'json-rpc' as const,
    }
    vi.mocked(mockCtx.clients.resolveAccount).mockResolvedValue(
      sponsoredAccount,
    )
    mockCtx.pendingPromises = [mockPonderPromise]
    vi.mocked(mockPublicClient.readContract).mockResolvedValue(true)
    vi.mocked(mockWalletClient.writeContract).mockResolvedValue(
      '0xresolvetxhash',
    )

    const result = await module.run(mockCtx)

    expect(mockCtx.clients.resolveAccount).toHaveBeenCalledWith(
      mockWalletClient,
    )
    expect(mockWalletClient.writeContract).toHaveBeenCalledWith(
      expect.objectContaining({
        account: sponsoredAccount,
      }),
    )
    expect(result).toEqual({ relayed: 1, skipped: 0, failed: 0, noMatch: 0 })
  })

  it('should skip when no accounts found in sponsored mode', async () => {
    vi.mocked(mockCtx.clients.resolveAccount).mockResolvedValue(undefined)
    mockCtx.pendingPromises = [mockPonderPromise]

    const result = await module.run(mockCtx)

    expect(result).toEqual({ relayed: 0, skipped: 1, failed: 0, noMatch: 0 })
    expect(mockLogger.warn).toHaveBeenCalledWith(
      'no accounts found, skipping...',
    )
    // Note: readContract IS called once during the upfront canResolve fan-out
    // (used to compute module_message_backlog). Account resolution happens inside the
    // attempt loop, after fan-out.
    expect(mockWalletClient.writeContract).not.toHaveBeenCalled()
  })

  it('should skip promise when resolver contract has no code (getCode returns 0x)', async () => {
    mockCtx.pendingPromises = [mockPonderPromise]
    vi.mocked(mockPublicClient.getCode).mockResolvedValue('0x')

    const result = await module.run(mockCtx)

    expect(result).toEqual({ relayed: 0, skipped: 1, failed: 0, noMatch: 0 })
    expect(mockPublicClient.getCode).toHaveBeenCalledWith({
      address: mockPonderPromise.resolver,
    })
    expect(mockLogger.warn).toHaveBeenCalledWith(
      'promise creator contract not found, skipping...',
    )
    // Note: readContract IS called once during the upfront canResolve fan-out
    // (used to compute module_message_backlog). The resolver-code check happens
    // inside the attempt loop, after fan-out.
    expect(mockWalletClient.writeContract).not.toHaveBeenCalled()
  })

  it('should call walletClient.writeContract (NOT ctx.relayMessage) for promise resolution', async () => {
    mockCtx.pendingPromises = [mockPonderPromise]
    vi.mocked(mockPublicClient.readContract).mockResolvedValue(true)
    vi.mocked(mockWalletClient.writeContract).mockResolvedValue(
      '0xresolvetxhash',
    )

    await module.run(mockCtx)

    // Promise resolution goes through walletClient.writeContract, NOT ctx.relayMessage
    expect(mockWalletClient.writeContract).toHaveBeenCalledWith({
      address: mockPonderPromise.resolver,
      abi: expect.arrayContaining([
        expect.objectContaining({
          name: 'resolve',
          type: 'function',
        }),
      ]),
      functionName: 'resolve',
      args: [mockPonderPromise.promiseId],
      account: mockWalletClient.account,
      // 100_000 (mocked estimate) padded by 3/2 in the module.
      gas: 150_000n,
      chain: null,
    })
    expect(mockCtx.relayMessage).not.toHaveBeenCalled()
  })

  describe('funnel counters', () => {
    let registry: Registry

    beforeEach(() => {
      registry = new Registry()
      mockCtx.metrics = new RelayerMetrics(registry)
    })

    async function scrape(): Promise<string> {
      return await registry.metrics()
    }

    it('increments failed_total{stage=simulation, reason=resolver_unreachable} on canResolve RPC error', async () => {
      mockCtx.pendingPromises = [mockPonderPromise]
      vi.mocked(mockPublicClient.readContract).mockRejectedValue(
        new Error('RPC flake'),
      )

      await module.run(mockCtx)

      expect(await scrape()).toMatch(
        /relayer_module_relay_attempt_failed_total\{.*module="promise".*stage="simulation".*reason="resolver_unreachable".*\} 1/,
      )
    })

    it('increments skipped_total{reason=promise_not_ready} on clean canResolve=false', async () => {
      mockCtx.pendingPromises = [mockPonderPromise]
      vi.mocked(mockPublicClient.readContract).mockResolvedValue(false)

      await module.run(mockCtx)

      expect(await scrape()).toMatch(
        /relayer_module_message_skipped_total\{.*module="promise".*reason="promise_not_ready".*\} 1/,
      )
    })

    it('increments broadcast_total with intra-chain src==dst on successful resolve', async () => {
      mockCtx.pendingPromises = [mockPonderPromise]
      vi.mocked(mockPublicClient.readContract).mockResolvedValue(true)
      vi.mocked(mockWalletClient.writeContract).mockResolvedValue(
        '0xresolvetxhash',
      )

      await module.run(mockCtx)

      const output = await scrape()
      expect(output).toMatch(
        /relayer_module_relay_tx_broadcast_total\{module="promise",src="1",dst="1",.*\} 1/,
      )
    })
  })
})
