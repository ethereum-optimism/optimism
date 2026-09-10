import type { PonderInteropClient } from '@eth-optimism/ponder-interop/client'
import {
  relayCrossDomainMessage,
  simulateRelayCrossDomainMessage,
} from '@eth-optimism/viem/actions/interop'
import { Registry } from 'prom-client'
import type { Account, Hex, WalletClient } from 'viem'
import { simulateContract, writeContract } from 'viem/actions'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { RelayError } from '@/errors.js'
import { RelayerMetrics } from '@/metrics.js'
import { RelayFailureRegistry } from '@/relay/relayFailureRegistry.js'
import { Relayer, type RelayerConfig } from '@/relayer.js'
import type { ClientManager } from '@/services/clientManager.js'
import type {
  RelayMessageParams,
  RelayModule,
  RunContext,
  RunResult,
} from '@/strategies/types.js'

// Mock the viem modules (needed for submission helpers in relayer.ts)
vi.mock('@eth-optimism/viem/abis/experimental', () => ({
  gasTankAbi: [],
}))

vi.mock('@eth-optimism/viem/actions/interop', () => ({
  relayCrossDomainMessage: vi.fn(),
  simulateRelayCrossDomainMessage: vi.fn(),
}))

vi.mock('viem/actions', () => ({
  simulateContract: vi.fn(),
  writeContract: vi.fn(),
}))

interface MockLogger {
  child: ReturnType<typeof vi.fn>
  info: ReturnType<typeof vi.fn>
  warn: ReturnType<typeof vi.fn>
  error: ReturnType<typeof vi.fn>
  debug: ReturnType<typeof vi.fn>
}

describe('Relayer', () => {
  let mockLogger: MockLogger
  let mockConfig: RelayerConfig
  let mockCtx: RunContext

  beforeEach(() => {
    mockLogger = {
      child: vi.fn().mockReturnThis(),
      info: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
      debug: vi.fn(),
    }

    mockConfig = {
      ponderInteropApi: 'http://localhost:42069',
      ponderClient: {} as PonderInteropClient,
      clients: {
        getPublicClient: vi.fn(),
        getWalletClient: vi.fn(),
        resolveAccount: vi.fn(),
        listSigningEoas: vi.fn().mockReturnValue([]),
      } as unknown as ClientManager,
      failureRegistry: new RelayFailureRegistry(':memory:'),
      gasTankAddress: undefined,
    }

    mockCtx = {
      ponderClient: {
        getPendingMessages: vi.fn().mockResolvedValue([]),
        getPendingPromises: vi.fn().mockResolvedValue([]),
        getUnsharedResolvedPromises: vi.fn().mockResolvedValue([]),
      } as unknown as PonderInteropClient,
      clients: mockConfig.clients,
      log: mockLogger as unknown as RunContext['log'],
      pendingMessages: [],
      pendingPromises: [],
      unsharedResolvedPromises: [],
      metrics: new RelayerMetrics(new Registry()),
      relayMessage: vi.fn(),
    }

    mockConfig.ponderClient = mockCtx.ponderClient
  })

  describe('run', () => {
    it('should call run() on each registered module', async () => {
      const moduleA: RelayModule = {
        name: 'module-a',
        run: vi
          .fn()
          .mockResolvedValue({ relayed: 1, skipped: 0, failed: 0, noMatch: 0 }),
      }
      const moduleB: RelayModule = {
        name: 'module-b',
        run: vi
          .fn()
          .mockResolvedValue({ relayed: 2, skipped: 1, failed: 0, noMatch: 0 }),
      }

      const relayer = new Relayer(mockLogger as any, mockConfig, [
        moduleA,
        moduleB,
      ])
      relayer.setContext(mockCtx)
      await relayer.run()

      // Modules receive the cycle context: mockCtx plus the per-cycle
      // failedEndpoints set (empty when all fetches succeed).
      expect(moduleA.run).toHaveBeenCalledWith({
        ...mockCtx,
        failedEndpoints: new Set(),
      })
      expect(moduleB.run).toHaveBeenCalledWith({
        ...mockCtx,
        failedEndpoints: new Set(),
      })
    })

    it('should log RunResult after each module completes', async () => {
      const result: RunResult = {
        relayed: 3,
        skipped: 1,
        failed: 0,
        noMatch: 0,
      }
      const module: RelayModule = {
        name: 'test-module',
        run: vi.fn().mockResolvedValue(result),
      }

      const relayer = new Relayer(mockLogger as any, mockConfig, [module])
      relayer.setContext(mockCtx)
      await relayer.run()

      expect(mockLogger.info).toHaveBeenCalledWith(
        {
          module: 'test-module',
          relayed: 3,
          skipped: 1,
          failed: 0,
          noMatch: 0,
        },
        'module completed',
      )
    })

    it('should catch and log module errors without stopping other modules', async () => {
      const failingModule: RelayModule = {
        name: 'failing-module',
        run: vi.fn().mockRejectedValue(new Error('module crashed')),
      }
      const successModule: RelayModule = {
        name: 'success-module',
        run: vi
          .fn()
          .mockResolvedValue({ relayed: 1, skipped: 0, failed: 0, noMatch: 0 }),
      }

      const relayer = new Relayer(mockLogger as any, mockConfig, [
        failingModule,
        successModule,
      ])
      relayer.setContext(mockCtx)
      await relayer.run()

      expect(mockLogger.error).toHaveBeenCalledWith(
        { err: expect.any(Error), module: 'failing-module' },
        'module run failed',
      )
      expect(successModule.run).toHaveBeenCalledWith({
        ...mockCtx,
        failedEndpoints: new Set(),
      })
    })

    it('should run modules in registration order', async () => {
      const order: string[] = []
      const moduleA: RelayModule = {
        name: 'first',
        run: vi.fn().mockImplementation(async () => {
          order.push('first')
          return { relayed: 0, skipped: 0, failed: 0, noMatch: 0 }
        }),
      }
      const moduleB: RelayModule = {
        name: 'second',
        run: vi.fn().mockImplementation(async () => {
          order.push('second')
          return { relayed: 0, skipped: 0, failed: 0, noMatch: 0 }
        }),
      }

      const relayer = new Relayer(mockLogger as any, mockConfig, [
        moduleA,
        moduleB,
      ])
      relayer.setContext(mockCtx)
      await relayer.run()

      expect(order).toEqual(['first', 'second'])
    })

    it('should work with empty module list', async () => {
      const relayer = new Relayer(mockLogger as any, mockConfig, [])
      relayer.setContext(mockCtx)

      await expect(relayer.run()).resolves.toBeUndefined()
    })

    it('should throw if RunContext not set before run()', async () => {
      const relayer = new Relayer(mockLogger as any, mockConfig, [])

      await expect(relayer.run()).rejects.toThrow('RunContext not set')
    })
  })

  describe('submitRelayMessage typed errors', () => {
    const account = {
      address: '0x0000000000000000000000000000000000000001',
    } as Account
    const walletClient = { account } as unknown as WalletClient
    const params: RelayMessageParams = {
      id: {
        origin: '0x0000000000000000000000000000000000000000',
        chainId: 1n,
        logIndex: 0n,
        blockNumber: 0n,
        timestamp: 0n,
      } as RelayMessageParams['id'],
      destinationChainId: 420120000,
      payload: '0x' as Hex,
      account,
      accessList: [],
      chain: null,
      walletClient,
      txOrigin: '0x0000000000000000000000000000000000000002',
      messageHash: '0xdead',
    }

    let relayer: Relayer
    let runCtx: RunContext

    beforeEach(() => {
      vi.mocked(simulateRelayCrossDomainMessage).mockReset()
      vi.mocked(relayCrossDomainMessage).mockReset()
      vi.mocked(simulateContract).mockReset()
      vi.mocked(writeContract).mockReset()

      vi.mocked(mockConfig.clients.getPublicClient).mockReturnValue({} as any)

      relayer = new Relayer(mockLogger as any, mockConfig, [])
      runCtx = { ...mockCtx }
      relayer.setContext(runCtx)
    })

    it('wraps L2ToL2CDM simulation failures as RelayError{stage: simulation}', async () => {
      vi.mocked(simulateRelayCrossDomainMessage).mockRejectedValueOnce(
        new Error('simulation boom'),
      )

      await expect(relayer.submitRelayMessage(params)).rejects.toMatchObject({
        name: 'RelayError',
        stage: 'simulation',
        reason: 'unknown',
      })
    })

    it('wraps L2ToL2CDM broadcast failures as RelayError{stage: broadcast}', async () => {
      vi.mocked(simulateRelayCrossDomainMessage).mockResolvedValueOnce(
        {} as any,
      )
      vi.mocked(relayCrossDomainMessage).mockRejectedValueOnce(
        new Error('broadcast boom'),
      )

      await expect(relayer.submitRelayMessage(params)).rejects.toMatchObject({
        name: 'RelayError',
        stage: 'broadcast',
        reason: 'unknown',
      })
    })

    it('passes caller-provided gas to L2ToL2CDM relay submissions', async () => {
      vi.mocked(simulateRelayCrossDomainMessage).mockResolvedValueOnce(
        {} as any,
      )
      vi.mocked(relayCrossDomainMessage).mockResolvedValueOnce('0xtxhash')

      await relayer.submitRelayMessage({
        ...params,
        estimatedGasCost: 200_000n,
      })

      expect(relayCrossDomainMessage).toHaveBeenCalledWith(
        expect.anything(),
        expect.objectContaining({ gas: 200_000n }),
      )
    })

    it('keeps gas unset for L2ToL2CDM relay submissions without caller-provided gas', async () => {
      vi.mocked(simulateRelayCrossDomainMessage).mockResolvedValueOnce(
        {} as any,
      )
      vi.mocked(relayCrossDomainMessage).mockResolvedValueOnce('0xtxhash')

      await relayer.submitRelayMessage(params)

      expect(relayCrossDomainMessage).toHaveBeenCalledWith(
        expect.anything(),
        expect.objectContaining({ gas: undefined }),
      )
    })

    it('preserves the underlying error as cause', async () => {
      const underlying = new Error('underlying')
      vi.mocked(simulateRelayCrossDomainMessage).mockRejectedValueOnce(
        underlying,
      )

      const caught = await relayer
        .submitRelayMessage(params)
        .catch((e: unknown) => e)

      expect(caught).toBeInstanceOf(RelayError)
      expect((caught as RelayError).cause).toBe(underlying)
    })

    it('wraps GasTank simulation failures as RelayError{stage: simulation}', async () => {
      const gasTankRelayer = new Relayer(
        mockLogger as any,
        {
          ...mockConfig,
          gasTankAddress: '0x000000000000000000000000000000000000dead',
        },
        [],
      )
      gasTankRelayer.setContext(runCtx)
      vi.mocked(simulateContract).mockRejectedValueOnce(
        new Error('gas tank sim boom'),
      )

      await expect(
        gasTankRelayer.submitRelayMessage(params),
      ).rejects.toMatchObject({
        name: 'RelayError',
        stage: 'simulation',
        reason: 'unknown',
      })
    })

    it('wraps GasTank broadcast failures as RelayError{stage: broadcast}', async () => {
      const gasTankRelayer = new Relayer(
        mockLogger as any,
        {
          ...mockConfig,
          gasTankAddress: '0x000000000000000000000000000000000000dead',
        },
        [],
      )
      gasTankRelayer.setContext(runCtx)
      vi.mocked(simulateContract).mockResolvedValueOnce({} as any)
      vi.mocked(writeContract).mockRejectedValueOnce(
        new Error('gas tank write boom'),
      )

      await expect(
        gasTankRelayer.submitRelayMessage(params),
      ).rejects.toMatchObject({
        name: 'RelayError',
        stage: 'broadcast',
        reason: 'unknown',
      })
    })
  })

  describe('cycle-scoped Ponder fetch', () => {
    it('fetches pendingMessages once per cycle', async () => {
      const module: RelayModule = {
        name: 'consumer',
        run: vi
          .fn()
          .mockResolvedValue({ relayed: 0, skipped: 0, failed: 0, noMatch: 0 }),
      }

      const relayer = new Relayer(mockLogger as any, mockConfig, [
        module,
        module,
      ])
      relayer.setContext(mockCtx)
      await relayer.run()

      expect(mockCtx.ponderClient.getPendingMessages).toHaveBeenCalledTimes(1)
    })

    it('passes fetched pendingMessages to modules via RunContext', async () => {
      const sentMessage = { messageHash: '0x1' } as any

      vi.mocked(mockCtx.ponderClient.getPendingMessages)
        .mockResolvedValueOnce([sentMessage])
        .mockResolvedValue([])

      const receivedCtx: RunContext[] = []
      const module: RelayModule = {
        name: 'consumer',
        run: vi.fn().mockImplementation(async (ctx: RunContext) => {
          receivedCtx.push(ctx)
          return { relayed: 0, skipped: 0, failed: 0, noMatch: 0 }
        }),
      }

      const relayer = new Relayer(mockLogger as any, mockConfig, [module])
      relayer.setContext(mockCtx)
      await relayer.run()

      expect(receivedCtx[0]?.pendingMessages).toEqual([sentMessage])
    })

    it('degrades to empty list on Ponder fetch failure and still runs modules', async () => {
      vi.mocked(mockCtx.ponderClient.getPendingMessages).mockRejectedValue(
        new Error('ponder down'),
      )

      const receivedCtx: RunContext[] = []
      const module: RelayModule = {
        name: 'consumer',
        run: vi.fn().mockImplementation(async (ctx: RunContext) => {
          receivedCtx.push(ctx)
          return { relayed: 0, skipped: 0, failed: 0, noMatch: 0 }
        }),
      }

      const relayer = new Relayer(mockLogger as any, mockConfig, [module])
      relayer.setContext(mockCtx)
      await relayer.run()

      expect(receivedCtx[0]?.pendingMessages).toEqual([])
      expect(module.run).toHaveBeenCalled()
      expect(mockLogger.error).toHaveBeenCalledWith(
        expect.objectContaining({ endpoint: 'pending_messages' }),
        expect.stringContaining('ponder fetch failed'),
      )
    })
  })

  describe('top-level metrics', () => {
    let registry: Registry

    beforeEach(() => {
      registry = new Registry()
      mockCtx.metrics = new RelayerMetrics(registry)
    })

    async function scrape(): Promise<string> {
      return await registry.metrics()
    }

    it('increments cycles_total once per run()', async () => {
      const relayer = new Relayer(mockLogger as any, mockConfig, [])
      relayer.setContext(mockCtx)
      await relayer.run()
      await relayer.run()
      await relayer.run()

      expect(await scrape()).toContain('relayer_cycles_total 3')
    })

    it('observes cycle_duration_seconds at end of each run()', async () => {
      const relayer = new Relayer(mockLogger as any, mockConfig, [])
      relayer.setContext(mockCtx)
      await relayer.run()

      const output = await scrape()
      expect(output).toMatch(/relayer_cycle_duration_seconds_count 1/)
    })

    it('buckets messages_from_indexer by src and dst', async () => {
      vi.mocked(mockCtx.ponderClient.getPendingMessages)
        .mockResolvedValueOnce([
          { source: 1, destination: 2 } as any,
          { source: 1, destination: 2 } as any,
          { source: 1, destination: 2 } as any,
          { source: 1, destination: 3 } as any,
        ])
        .mockResolvedValue([])

      const relayer = new Relayer(mockLogger as any, mockConfig, [])
      relayer.setContext(mockCtx)
      await relayer.run()

      const output = await scrape()
      expect(output).toContain(
        'relayer_messages_from_indexer{src="1",dst="2"} 3',
      )
      expect(output).toContain(
        'relayer_messages_from_indexer{src="1",dst="3"} 1',
      )
    })

    it('resets messages_from_indexer each cycle so stale buckets do not linger', async () => {
      vi.mocked(mockCtx.ponderClient.getPendingMessages)
        .mockResolvedValueOnce([{ source: 1, destination: 2 } as any])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([{ source: 9, destination: 9 } as any])
        .mockResolvedValue([])

      const relayer = new Relayer(mockLogger as any, mockConfig, [])
      relayer.setContext(mockCtx)
      await relayer.run()
      await relayer.run()

      const output = await scrape()
      expect(output).not.toContain('src="1",dst="2"')
      expect(output).toContain(
        'relayer_messages_from_indexer{src="9",dst="9"} 1',
      )
    })

    it('observes ponder_request_duration_seconds for pending_messages on success', async () => {
      const relayer = new Relayer(mockLogger as any, mockConfig, [])
      relayer.setContext(mockCtx)
      await relayer.run()

      const output = await scrape()
      expect(output).toMatch(
        /relayer_ponder_request_duration_seconds_count\{endpoint="pending_messages"\} 1/,
      )
    })

    it('sets ponder_last_success_timestamp on successful fetch', async () => {
      const relayer = new Relayer(mockLogger as any, mockConfig, [])
      relayer.setContext(mockCtx)
      await relayer.run()

      const output = await scrape()
      expect(output).toMatch(
        /relayer_ponder_last_success_timestamp\{endpoint="pending_messages"\} \d+/,
      )
    })

    it('increments ponder_errors_total on fetch failure', async () => {
      vi.mocked(mockCtx.ponderClient.getPendingMessages).mockRejectedValue(
        new Error('HTTP 502: Bad Gateway'),
      )

      const relayer = new Relayer(mockLogger as any, mockConfig, [])
      relayer.setContext(mockCtx)
      await relayer.run()

      const output = await scrape()
      expect(output).toContain(
        'relayer_ponder_errors_total{endpoint="pending_messages",kind="http_5xx"} 1',
      )
    })

    it('classifies ponder errors by kind', async () => {
      const cases: Array<{ err: Error; kind: string }> = [
        { err: new Error('HTTP 404: Not Found'), kind: 'http_4xx' },
        { err: new Error('HTTP 503: Unavailable'), kind: 'http_5xx' },
        {
          err: new Error('API response validation failed for /x: ...'),
          kind: 'parse',
        },
        { err: new Error('connection refused'), kind: 'network' },
        {
          err: Object.assign(new Error('aborted'), { name: 'AbortError' }),
          kind: 'timeout',
        },
      ]

      for (const { err, kind } of cases) {
        const localRegistry = new Registry()
        mockCtx.metrics = new RelayerMetrics(localRegistry)
        vi.mocked(
          mockCtx.ponderClient.getPendingMessages,
        ).mockRejectedValueOnce(err)

        const relayer = new Relayer(mockLogger as any, mockConfig, [])
        relayer.setContext(mockCtx)
        await relayer.run()

        const output = await localRegistry.metrics()
        expect(output).toContain(
          `relayer_ponder_errors_total{endpoint="pending_messages",kind="${kind}"} 1`,
        )
      }
    })

    it('does not update last_success_timestamp when a fetch fails', async () => {
      vi.mocked(mockCtx.ponderClient.getPendingMessages).mockRejectedValue(
        new Error('HTTP 500'),
      )

      const relayer = new Relayer(mockLogger as any, mockConfig, [])
      relayer.setContext(mockCtx)
      await relayer.run()

      const output = await scrape()
      expect(output).not.toMatch(
        /relayer_ponder_last_success_timestamp\{endpoint="pending_messages"\}/,
      )
    })
  })

  describe('needs-gated endpoint fetching', () => {
    it('fetches promise lists only when a module needs them', async () => {
      const promiseModule: RelayModule = {
        name: 'promise',
        needs: ['pendingPromises'],
        run: vi
          .fn()
          .mockResolvedValue({ relayed: 0, skipped: 0, failed: 0, noMatch: 0 }),
      }

      const relayer = new Relayer(mockLogger as any, mockConfig, [
        promiseModule,
      ])
      relayer.setContext(mockCtx)
      await relayer.run()

      // pendingMessages is always fetched (baseline); pendingPromises fetched
      // because a module needs it; unsharedResolvedPromises NOT fetched.
      expect(mockCtx.ponderClient.getPendingMessages).toHaveBeenCalled()
      expect(mockCtx.ponderClient.getPendingPromises).toHaveBeenCalled()
      expect(
        mockCtx.ponderClient.getUnsharedResolvedPromises,
      ).not.toHaveBeenCalled()
    })

    it('does not fetch promise lists for an eth-bridge-only relayer', async () => {
      const bridgeModule: RelayModule = {
        name: 'eth-bridge',
        needs: ['pendingMessages'],
        run: vi
          .fn()
          .mockResolvedValue({ relayed: 0, skipped: 0, failed: 0, noMatch: 0 }),
      }

      const relayer = new Relayer(mockLogger as any, mockConfig, [bridgeModule])
      relayer.setContext(mockCtx)
      await relayer.run()

      expect(mockCtx.ponderClient.getPendingMessages).toHaveBeenCalled()
      expect(mockCtx.ponderClient.getPendingPromises).not.toHaveBeenCalled()
      expect(
        mockCtx.ponderClient.getUnsharedResolvedPromises,
      ).not.toHaveBeenCalled()
    })
  })

  describe('pagination starvation (R11 / permastuck)', () => {
    /** Build a minimal pending message with a deterministic hash. */
    function makePending(i: number, hash: string) {
      return {
        messageHash: hash,
        source: 1,
        destination: 901,
        sender: '0x9999999999999999999999999999999999999999',
        messageIdentifierHash: `0xid${i}`,
        nonce: i,
        target: '0x2222222222222222222222222222222222222222',
        message: '0x',
        logIndex: i,
        logPayload: '0x',
        timestamp: 1000 + i,
        blockNumber: i,
        transactionHash: `0xtx${i}`,
        txOrigin: '0x3333333333333333333333333333333333333333',
      }
    }

    /**
     * Ed's permastuck scenario at unit scale: 25 permanently-failed
     * (abandoned) messages at the head of the oldest-first pending list, one
     * fresh message behind them, and a per-cycle budget of 20. With the old
     * single-window fetch the fresh message never enters the window; with
     * budget-aware paging it must reach the modules.
     */
    it('a fresh message behind a head of abandoned messages still reaches modules', async () => {
      const failureRegistry = new RelayFailureRegistry(':memory:')
      mockConfig.failureRegistry = failureRegistry
      mockConfig.pagination = {
        pageLimit: 10,
        maxAttemptablePerCycle: 20,
        maxScanPerCycle: 100,
      }

      const stuck = Array.from({ length: 25 }, (_, i) =>
        makePending(i, `0xstuck${String(i).padStart(4, '0')}`),
      )
      for (const m of stuck) {
        // rpc_rejected → permanent on first failure.
        failureRegistry.recordFailure(m.messageHash, 1, 901, 'rpc_rejected')
      }
      const fresh = makePending(9999, '0xfresh')
      const allPending = [...stuck, fresh]

      // Fake server: honors limit/offset over the oldest-first list, exactly
      // like /messages/pending.
      vi.mocked(mockCtx.ponderClient.getPendingMessages).mockImplementation(
        (async (params?: { limit?: number; offset?: number }) => {
          const offset = params?.offset ?? 0
          const limit = params?.limit ?? 100
          return allPending.slice(offset, offset + limit)
        }) as any,
      )

      const module: RelayModule = {
        name: 'observer',
        needs: ['pendingMessages'],
        run: vi
          .fn()
          .mockResolvedValue({ relayed: 0, skipped: 0, failed: 0, noMatch: 0 }),
      }
      const relayer = new Relayer(mockLogger as any, mockConfig, [module])
      relayer.setContext(mockCtx)
      await relayer.run()

      const ctxArg = vi.mocked(module.run).mock.calls[0]![0] as RunContext
      const hashes = ctxArg.pendingMessages.map((m) => m.messageHash)
      expect(hashes).toContain('0xfresh')
      // Abandoned rows must survive GC even though they were fetched —
      // they are still pending server-side.
      expect(failureRegistry.hasFailed('0xstuck0000')).toBe(true)
    })

    it('does not GC registry rows for keys beyond a truncated fetch window', async () => {
      const failureRegistry = new RelayFailureRegistry(':memory:')
      mockConfig.failureRegistry = failureRegistry
      mockConfig.pagination = {
        pageLimit: 5,
        maxAttemptablePerCycle: 5,
        maxScanPerCycle: 5,
      }

      // 10 pending; only the first 5 fit the window. A failure row for a
      // message beyond the window must survive the cycle.
      const allPending = Array.from({ length: 10 }, (_, i) =>
        makePending(i, `0xmsg${String(i).padStart(4, '0')}`),
      )
      failureRegistry.recordFailure('0xmsg0009', 1, 901, 'unknown', Date.now())

      vi.mocked(mockCtx.ponderClient.getPendingMessages).mockImplementation(
        (async (params?: { limit?: number; offset?: number }) => {
          const offset = params?.offset ?? 0
          const limit = params?.limit ?? 100
          return allPending.slice(offset, offset + limit)
        }) as any,
      )

      const module: RelayModule = {
        name: 'observer',
        needs: ['pendingMessages'],
        run: vi
          .fn()
          .mockResolvedValue({ relayed: 0, skipped: 0, failed: 0, noMatch: 0 }),
      }
      const relayer = new Relayer(mockLogger as any, mockConfig, [module])
      relayer.setContext(mockCtx)
      await relayer.run()

      expect(failureRegistry.hasFailed('0xmsg0009')).toBe(true)
    })
  })

  describe('unowned-message observability (R1)', () => {
    const unownedMessage = {
      messageHash: '0xunowned',
      source: 1,
      destination: 901,
      sender: '0x9999999999999999999999999999999999999999',
      messageIdentifierHash: '0x1',
      nonce: 1,
      target: '0x2222222222222222222222222222222222222222',
      message: '0x',
      logIndex: 0,
      logPayload: '0x',
      timestamp: 1,
      blockNumber: 1,
      transactionHash: '0x2',
      txOrigin: '0x3333333333333333333333333333333333333333',
    }

    it('emits relayer_messages_unowned when no enabled module owns a pending message', async () => {
      const registry = new Registry()
      mockCtx.metrics = new RelayerMetrics(registry)
      vi.mocked(mockCtx.ponderClient.getPendingMessages).mockResolvedValue([
        unownedMessage,
      ] as any)

      // A specialized module that owns a different sender, and no catch-all.
      const module: RelayModule = {
        name: 'specialized',
        needs: ['pendingMessages'],
        ownedSenders: ['0x1111111111111111111111111111111111111111'],
        run: vi
          .fn()
          .mockResolvedValue({ relayed: 0, skipped: 0, failed: 0, noMatch: 1 }),
      }
      const relayer = new Relayer(mockLogger as any, mockConfig, [module])
      relayer.setContext(mockCtx)
      await relayer.run()

      const output = await registry.metrics()
      expect(output).toMatch(
        /relayer_messages_unowned\{src="1",dst="901"\} 1/,
      )
    })

    it('reports zero unowned messages when a catch-all module is enabled', async () => {
      const registry = new Registry()
      mockCtx.metrics = new RelayerMetrics(registry)
      vi.mocked(mockCtx.ponderClient.getPendingMessages).mockResolvedValue([
        unownedMessage,
      ] as any)

      const catchAll: RelayModule = {
        name: 'general-relay',
        needs: ['pendingMessages'],
        catchAll: true,
        run: vi
          .fn()
          .mockResolvedValue({ relayed: 1, skipped: 0, failed: 0, noMatch: 0 }),
      }
      const relayer = new Relayer(mockLogger as any, mockConfig, [catchAll])
      relayer.setContext(mockCtx)
      await relayer.run()

      const output = await registry.metrics()
      expect(output).not.toMatch(/relayer_messages_unowned\{.*\} [1-9]/)
    })
  })

  describe('shared failure-registry GC across modules (R8)', () => {
    it('one module cycle must not wipe another module keyspace from the shared registry', async () => {
      const { RelayFailureRegistry } = await import(
        '@/relay/relayFailureRegistry.js'
      )
      const { EthBridgeModule } = await import(
        '@/strategies/ethBridgeModule.js'
      )
      const { PromiseModule } = await import('@/strategies/promiseModule.js')
      const { contracts } = await import('@eth-optimism/viem')

      const failureRegistry = new RelayFailureRegistry(':memory:')
      mockConfig.failureRegistry = failureRegistry

      const MSG_HASH = '0xaaaa000000000000000000000000000000000000000000000001'
      const PROMISE_ID =
        '0xbbbb000000000000000000000000000000000000000000000002'

      // Prior cycles left failure history for one key in each keyspace.
      // Both are still pending per Ponder, so both rows must survive GC.
      failureRegistry.recordFailure(MSG_HASH, 1, 901, 'unknown', Date.now())
      failureRegistry.recordFailure(PROMISE_ID, 901, 901, 'unknown', Date.now())

      const pendingMessage = {
        messageIdentifierHash: '0xabcd',
        messageHash: MSG_HASH,
        source: 1,
        destination: 901,
        nonce: 1,
        sender: contracts.superchainETHBridge.address,
        target: '0x2222222222222222222222222222222222222222',
        message: '0x',
        logIndex: 5,
        logPayload: '0x5678',
        timestamp: 1234567890,
        blockNumber: 100,
        transactionHash: '0x9999',
        txOrigin: '0x3333333333333333333333333333333333333333',
      }
      const pendingPromise = {
        promiseId: PROMISE_ID,
        chainId: 901,
        resolver: '0x4444444444444444444444444444444444444444',
        creator: '0x5555555555555555555555555555555555555555',
        createdAt: 1234567890,
        transactionHash: '0x8888',
      }

      vi.mocked(mockCtx.ponderClient.getPendingMessages).mockResolvedValue([
        pendingMessage,
      ] as any)
      vi.mocked(mockCtx.ponderClient.getPendingPromises).mockResolvedValue([
        pendingPromise,
      ] as any)
      ;(mockCtx.ponderClient as any).getDepositBalance = vi
        .fn()
        .mockResolvedValue({ totalBalance: '0' })

      const bridgeModule = new EthBridgeModule(
        {
          recordConsumption: vi.fn(),
          hasEnoughBudget: vi.fn().mockReturnValue(false),
          markBlocked: vi.fn(),
          clearBlocked: vi.fn(),
        } as any,
        failureRegistry,
      )
      const promiseModule = new PromiseModule(failureRegistry)

      // No clients configured: both modules skip all attempts. The bug under
      // test is purely in the GC path, which runs regardless of attempts.
      const relayer = new Relayer(mockLogger as any, mockConfig, [
        bridgeModule,
        promiseModule,
      ])
      relayer.setContext(mockCtx)
      await relayer.run()

      // On main, EthBridgeModule GCs against only its own pending set (the
      // message hash), deleting the promise row — and PromiseModule then GCs
      // against only the promise id, deleting the message row. Both keys are
      // still pending, so both failure rows must survive the cycle.
      expect(failureRegistry.hasFailed(MSG_HASH)).toBe(true)
      expect(failureRegistry.hasFailed(PROMISE_ID)).toBe(true)
    })

    it('GCs rows for keys that are no longer pending anywhere', async () => {
      const { RelayFailureRegistry } = await import(
        '@/relay/relayFailureRegistry.js'
      )
      const failureRegistry = new RelayFailureRegistry(':memory:')
      failureRegistry.recordFailure('0xgone', 1, 901, 'unknown', Date.now())
      mockConfig.failureRegistry = failureRegistry

      const module: RelayModule = {
        name: 'test-module',
        needs: ['pendingMessages'],
        run: vi
          .fn()
          .mockResolvedValue({ relayed: 0, skipped: 0, failed: 0, noMatch: 0 }),
      }
      const relayer = new Relayer(mockLogger as any, mockConfig, [module])
      relayer.setContext(mockCtx)
      await relayer.run()

      expect(failureRegistry.hasFailed('0xgone')).toBe(false)
    })

    it('skips GC when a fetch failed, even for other endpoints', async () => {
      const { RelayFailureRegistry } = await import(
        '@/relay/relayFailureRegistry.js'
      )
      const failureRegistry = new RelayFailureRegistry(':memory:')
      failureRegistry.recordFailure('0xkeep', 1, 901, 'unknown', Date.now())
      mockConfig.failureRegistry = failureRegistry

      vi.mocked(mockCtx.ponderClient.getPendingMessages).mockRejectedValue(
        new Error('ponder down'),
      )
      const module: RelayModule = {
        name: 'test-module',
        needs: ['pendingMessages'],
        run: vi
          .fn()
          .mockResolvedValue({ relayed: 0, skipped: 0, failed: 0, noMatch: 0 }),
      }
      const relayer = new Relayer(mockLogger as any, mockConfig, [module])
      relayer.setContext(mockCtx)
      await relayer.run()

      expect(failureRegistry.hasFailed('0xkeep')).toBe(true)
    })
  })

  describe('fetch-failure signalling (R12)', () => {
    it('flags a failed endpoint in the module RunContext instead of silently passing an empty list', async () => {
      vi.mocked(mockCtx.ponderClient.getPendingMessages).mockRejectedValue(
        new Error('connect ECONNREFUSED 127.0.0.1:42069'),
      )
      const module: RelayModule = {
        name: 'test-module',
        needs: ['pendingMessages'],
        run: vi
          .fn()
          .mockResolvedValue({ relayed: 0, skipped: 0, failed: 0, noMatch: 0 }),
      }

      const relayer = new Relayer(mockLogger as any, mockConfig, [module])
      relayer.setContext(mockCtx)
      await relayer.run()

      const ctxArg = vi.mocked(module.run).mock.calls[0]![0] as RunContext
      expect(ctxArg.pendingMessages).toEqual([])
      expect(ctxArg.failedEndpoints?.has('pendingMessages')).toBe(true)
    })

    it('reports no failed endpoints when all fetches succeed', async () => {
      const module: RelayModule = {
        name: 'test-module',
        needs: ['pendingMessages'],
        run: vi
          .fn()
          .mockResolvedValue({ relayed: 0, skipped: 0, failed: 0, noMatch: 0 }),
      }

      const relayer = new Relayer(mockLogger as any, mockConfig, [module])
      relayer.setContext(mockCtx)
      await relayer.run()

      const ctxArg = vi.mocked(module.run).mock.calls[0]![0] as RunContext
      expect(ctxArg.failedEndpoints?.size ?? 0).toBe(0)
    })
  })
})
