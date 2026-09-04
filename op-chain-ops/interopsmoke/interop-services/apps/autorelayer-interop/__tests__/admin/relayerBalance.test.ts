import { Hono } from 'hono'
import type { Logger } from 'pino'
import type { Address } from 'viem'
import { describe, expect, it } from 'vitest'

import { attachRelayerBalanceRoute } from '@/admin/relayerBalance.js'
import type { ClientManager } from '@/services/clientManager.js'

const ADDR_A = '0x1111111111111111111111111111111111111111' as Address
const ADDR_B = '0x2222222222222222222222222222222222222222' as Address
const ADDR_C = '0x3333333333333333333333333333333333333333' as Address

const silentLogger: Logger = {
  warn: () => {},
  info: () => {},
  error: () => {},
  debug: () => {},
  trace: () => {},
  fatal: () => {},
} as unknown as Logger

type BalanceMap = Record<number, Record<Address, bigint | Error>>

function fakeClients(opts: {
  cells: Array<{ address: Address; chainId: number }>
  balances: BalanceMap
}): Pick<ClientManager, 'listSigningEoas' | 'getPublicClient'> {
  return {
    listSigningEoas: () => opts.cells,
    getPublicClient: (chainId: number) =>
      ({
        getBalance: async ({ address }: { address: Address }) => {
          const entry = opts.balances[chainId]?.[address.toLowerCase() as Address]
          if (entry === undefined) {
            throw new Error(`no balance fixture for ${address} on ${chainId}`)
          }
          if (entry instanceof Error) throw entry
          return entry
        },
      }) as unknown as ReturnType<ClientManager['getPublicClient']>,
  }
}

function buildApp(deps: {
  clients: Pick<ClientManager, 'listSigningEoas' | 'getPublicClient'>
  chains: Array<{ id: number; name: string }>
  mode: 'local' | 'sponsored'
}): Hono {
  const api = new Hono()
  attachRelayerBalanceRoute(api, {
    clients: deps.clients as ClientManager,
    chains: deps.chains,
    mode: deps.mode,
    logger: silentLogger,
  })
  return api
}

describe('GET /relayer-balance', () => {
  const chains = [
    { id: 901, name: 'supersim-l2a' },
    { id: 902, name: 'supersim-l2b' },
  ]

  it('returns the local-mode happy path with sorted EOAs and floor', async () => {
    const clients = fakeClients({
      cells: [
        { address: ADDR_A, chainId: 901 },
        { address: ADDR_A, chainId: 902 },
        { address: ADDR_B, chainId: 901 },
        { address: ADDR_B, chainId: 902 },
        { address: ADDR_C, chainId: 901 },
        { address: ADDR_C, chainId: 902 },
      ],
      balances: {
        901: { [ADDR_A]: 5n, [ADDR_B]: 100n, [ADDR_C]: 50n },
        902: { [ADDR_A]: 999n, [ADDR_B]: 200n, [ADDR_C]: 80n },
      },
    })
    const app = buildApp({ clients, chains, mode: 'local' })

    const res = await app.fetch(new Request('http://x/relayer-balance'))
    expect(res.status).toBe(200)
    const body = (await res.json()) as {
      mode: string
      chains: Array<{ id: number; name: string }>
      eoas: Array<{
        address: Address
        balances: Array<{ chainId: number; balanceWei: string }>
      }>
      summary: {
        floor: { balanceWei: string; address: Address; chainId: number } | null
        totalWei: string
      }
    }

    expect(body.mode).toBe('local')
    expect(body.chains).toEqual(chains)
    // ADDR_A has min=5 (worst), ADDR_C has min=50, ADDR_B has min=100.
    expect(body.eoas.map((e) => e.address)).toEqual([ADDR_A, ADDR_C, ADDR_B])
    // Each EOA should have both chains in ascending-id order.
    expect(body.eoas[0]!.balances).toEqual([
      { chainId: 901, balanceWei: '5' },
      { chainId: 902, balanceWei: '999' },
    ])
    expect(body.summary.floor).toEqual({
      balanceWei: '5',
      address: ADDR_A,
      chainId: 901,
    })
    // Sum: 5 + 999 + 100 + 200 + 50 + 80 = 1434
    expect(body.summary.totalWei).toBe('1434')
  })

  it('returns sponsored shape with empty eoas and null floor', async () => {
    const clients = fakeClients({ cells: [], balances: {} })
    const app = buildApp({ clients, chains, mode: 'sponsored' })

    const res = await app.fetch(new Request('http://x/relayer-balance'))
    expect(res.status).toBe(200)
    expect(await res.json()).toEqual({
      mode: 'sponsored',
      chains,
      eoas: [],
      summary: { floor: null, totalWei: '0' },
    })
  })

  it('returns local-mode empty when no signing EOAs are configured', async () => {
    const clients = fakeClients({ cells: [], balances: {} })
    const app = buildApp({ clients, chains, mode: 'local' })

    const res = await app.fetch(new Request('http://x/relayer-balance'))
    expect(res.status).toBe(200)
    const body = (await res.json()) as { eoas: unknown[]; summary: { floor: unknown } }
    expect(body.eoas).toEqual([])
    expect(body.summary.floor).toBeNull()
  })

  it('returns 502 when any per-cell getBalance fails', async () => {
    const clients = fakeClients({
      cells: [
        { address: ADDR_A, chainId: 901 },
        { address: ADDR_A, chainId: 902 },
        { address: ADDR_B, chainId: 901 },
        { address: ADDR_B, chainId: 902 },
      ],
      balances: {
        901: { [ADDR_A]: 100n, [ADDR_B]: new Error('rpc down') },
        902: { [ADDR_A]: 100n, [ADDR_B]: 100n },
      },
    })
    const app = buildApp({ clients, chains, mode: 'local' })

    const res = await app.fetch(new Request('http://x/relayer-balance'))
    expect(res.status).toBe(502)
    expect(await res.json()).toEqual({
      error: 'failed to read one or more relayer EOA balances',
    })
  })

  it('places the floor at the cell with the smallest balance regardless of EOA grouping', async () => {
    const clients = fakeClients({
      cells: [
        { address: ADDR_A, chainId: 901 },
        { address: ADDR_A, chainId: 902 },
        { address: ADDR_B, chainId: 901 },
        { address: ADDR_B, chainId: 902 },
      ],
      balances: {
        // ADDR_A has a high min (10) but ADDR_B has a single low cell (1)
        901: { [ADDR_A]: 10n, [ADDR_B]: 1n },
        902: { [ADDR_A]: 1000n, [ADDR_B]: 1000n },
      },
    })
    const app = buildApp({ clients, chains, mode: 'local' })

    const res = await app.fetch(new Request('http://x/relayer-balance'))
    expect(res.status).toBe(200)
    const body = (await res.json()) as {
      summary: { floor: { balanceWei: string; address: Address; chainId: number } | null }
      eoas: Array<{ address: Address }>
    }
    expect(body.summary.floor).toEqual({
      balanceWei: '1',
      address: ADDR_B,
      chainId: 901,
    })
    // ADDR_B sorts first because its min (1) < ADDR_A's min (10).
    expect(body.eoas[0]!.address).toBe(ADDR_B)
  })
})
