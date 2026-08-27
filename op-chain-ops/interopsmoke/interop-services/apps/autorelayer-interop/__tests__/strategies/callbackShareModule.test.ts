import type {
  PonderInteropClient,
  UnsharedResolvedPromise,
} from '@eth-optimism/ponder-interop/client'
import { Registry } from 'prom-client'
import type { PublicClient, WalletClient } from 'viem'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { RelayerMetrics } from '@/metrics.js'
import { RelayFailureRegistry } from '@/relay/relayFailureRegistry.js'
import type { ClientManager } from '@/services/clientManager.js'
import { CallbackShareModule } from '@/strategies/callbackShareModule.js'
import type { RunContext } from '@/strategies/types.js'

interface MockLogger {
  child: ReturnType<typeof vi.fn>
  info: ReturnType<typeof vi.fn>
  warn: ReturnType<typeof vi.fn>
  error: ReturnType<typeof vi.fn>
  debug: ReturnType<typeof vi.fn>
}

const PROMISE_ADDRESS = '0xf94c51c00a72a92e8f31ee08e3b93cab24fdf304'

describe('CallbackShareModule', () => {
  let module: CallbackShareModule
  let mockCtx: RunContext
  let mockLogger: MockLogger
  let mockPublicClient: PublicClient
  let mockWalletClient: WalletClient

  const resolvedPromise: UnsharedResolvedPromise = {
    promiseId:
      '0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
    chainId: 901,
    resolver: '0x0000000000000000000000000000000000000000',
    status: 'resolved',
    createdAt: 1700000000,
    createdBlockNumber: 1000,
    createdTransactionHash: '0xabc',
    transferredAt: null,
    transferredBlockNumber: null,
    transferredTransactionHash: null,
    resolvedAt: 1700000100,
    resolvedBlockNumber: 1100,
    resolvedTransactionHash: '0xdef',
    pendingChainIds: [902, 903],
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
      waitForTransactionReceipt: vi
        .fn()
        .mockResolvedValue({ status: 'success' }),
    } as unknown as PublicClient

    mockWalletClient = {
      account: { address: '0x1234567890123456789012345678901234567890' },
      writeContract: vi.fn().mockResolvedValue('0xsharetx'),
      getAddresses: vi.fn(),
    } as unknown as WalletClient

    mockCtx = {
      ponderClient: {
        getUnsharedResolvedPromises: vi.fn(),
      } as unknown as PonderInteropClient,
      clients: {
        getPublicClient: vi.fn().mockReturnValue(mockPublicClient),
        getWalletClient: vi.fn().mockReturnValue(mockWalletClient),
        resolveAccount: vi.fn().mockResolvedValue(mockWalletClient.account),
      } as unknown as ClientManager,
      log: mockLogger as unknown as RunContext['log'],
      pendingMessages: [],
      pendingPromises: [],
      unsharedResolvedPromises: [],
      metrics: new RelayerMetrics(new Registry()),
      relayMessage: vi.fn(),
    }

    module = new CallbackShareModule(
      PROMISE_ADDRESS,
      new RelayFailureRegistry(':memory:'),
    )
  })

  it('has name "callback-share" and needs unsharedResolvedPromises', () => {
    expect(module.name).toBe('callback-share')
    expect(module.needs).toEqual(['unsharedResolvedPromises'])
  })

  it('returns zero result when nothing to share', async () => {
    mockCtx.unsharedResolvedPromises = []
    const result = await module.run(mockCtx)
    expect(result).toEqual({ relayed: 0, skipped: 0, failed: 0, noMatch: 0 })
    expect(mockWalletClient.writeContract).not.toHaveBeenCalled()
  })

  it('shares a resolved promise to each pending chain', async () => {
    mockCtx.unsharedResolvedPromises = [resolvedPromise]

    const result = await module.run(mockCtx)

    expect(result.relayed).toBe(2) // one per pendingChainId
    expect(mockWalletClient.writeContract).toHaveBeenCalledTimes(2)
    // Shares are sent from the chain where the promise is resolved.
    expect(mockCtx.clients.getWalletClient).toHaveBeenCalledWith(901)
    const firstCall = vi.mocked(mockWalletClient.writeContract).mock.calls[0][0]
    expect(firstCall).toMatchObject({
      address: PROMISE_ADDRESS,
      functionName: 'shareResolvedPromise',
      args: [902n, resolvedPromise.promiseId],
    })
  })

  it('skips when there is no client for the resolved chain', async () => {
    mockCtx.unsharedResolvedPromises = [resolvedPromise]
    vi.mocked(mockCtx.clients.getPublicClient).mockReturnValue(undefined)

    const result = await module.run(mockCtx)

    expect(result.skipped).toBe(1)
    expect(mockWalletClient.writeContract).not.toHaveBeenCalled()
  })

  it('counts a failed share without throwing', async () => {
    mockCtx.unsharedResolvedPromises = [
      { ...resolvedPromise, pendingChainIds: [902] },
    ]
    vi.mocked(mockWalletClient.writeContract).mockRejectedValueOnce(
      new Error('boom'),
    )

    const result = await module.run(mockCtx)

    expect(result.failed).toBe(1)
    expect(result.relayed).toBe(0)
  })
})
