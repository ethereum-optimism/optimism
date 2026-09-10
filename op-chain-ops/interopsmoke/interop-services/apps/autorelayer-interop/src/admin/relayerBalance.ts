import type { Hono } from 'hono'
import type { Logger } from 'pino'
import type { Address } from 'viem'

import type { ClientManager } from '@/services/clientManager.js'

export interface RelayerBalanceRouteDeps {
  clients: ClientManager
  chains: Array<{ id: number; name: string }>
  mode: 'local' | 'sponsored'
  logger: Logger
}

type EoaBalance = {
  chainId: number
  balanceWei: string
}

type EoaEntry = {
  address: Address
  balances: EoaBalance[]
}

type FloorCell = {
  balanceWei: string
  address: Address
  chainId: number
} | null

/**
 * Attaches `GET /relayer-balance` to the supplied admin Hono instance.
 *
 * Returns the full set of (signing EOA × chain) gas balances so a UI can
 * render the operationally-load-bearing floor — the cell that goes dry
 * first stops relays, regardless of how many other cells are flush.
 *
 * All balance reads run in a single Promise.all. If any single getBalance
 * fails the route returns 502 — a partial response would lie about the
 * floor.
 */
export function attachRelayerBalanceRoute(
  api: Hono,
  deps: RelayerBalanceRouteDeps,
): void {
  const { clients, chains, mode, logger } = deps

  api.get('/relayer-balance', async (c) => {
    if (mode === 'sponsored') {
      return c.json({
        mode,
        chains,
        eoas: [],
        summary: { floor: null as FloorCell, totalWei: '0' },
      })
    }

    const cells = clients.listSigningEoas()
    if (cells.length === 0) {
      return c.json({
        mode,
        chains,
        eoas: [],
        summary: { floor: null as FloorCell, totalWei: '0' },
      })
    }

    type Sample = { address: Address; chainId: number; balanceWei: bigint }
    let samples: Sample[]
    try {
      samples = await Promise.all(
        cells.map(async ({ address, chainId }) => {
          const pub = clients.getPublicClient(chainId)
          if (!pub) {
            throw new Error(`no public client for chain ${chainId}`)
          }
          const balanceWei = await pub.getBalance({ address })
          return { address, chainId, balanceWei }
        }),
      )
    } catch (error) {
      logger.warn(
        { err: error },
        'relayer-balance route: getBalance fan-out failed',
      )
      return c.json(
        { error: 'failed to read one or more relayer EOA balances' },
        502,
      )
    }

    // Group by EOA, preserve chain order from `chains` (ascending id).
    const chainOrder = chains.map((cn) => cn.id)
    const byAddr = new Map<Address, Map<number, bigint>>()
    for (const s of samples) {
      let row = byAddr.get(s.address)
      if (!row) {
        row = new Map<number, bigint>()
        byAddr.set(s.address, row)
      }
      row.set(s.chainId, s.balanceWei)
    }

    const eoas: EoaEntry[] = Array.from(byAddr.entries()).map(
      ([address, row]) => ({
        address,
        balances: chainOrder
          .filter((id) => row.has(id))
          .map((id) => ({
            chainId: id,
            balanceWei: (row.get(id) as bigint).toString(),
          })),
      }),
    )

    // Sort EOAs by ascending min-balance (worst first); tie-break by addr
    // for determinism. Empty balance lists go last (shouldn't happen, but
    // belt-and-braces).
    const minOf = (e: EoaEntry): bigint =>
      e.balances.length === 0
        ? 2n ** 256n - 1n
        : e.balances.reduce(
            (m, b) => (BigInt(b.balanceWei) < m ? BigInt(b.balanceWei) : m),
            BigInt(e.balances[0]!.balanceWei),
          )
    eoas.sort((a, b) => {
      const ma = minOf(a)
      const mb = minOf(b)
      if (ma !== mb) return ma < mb ? -1 : 1
      return a.address < b.address ? -1 : a.address > b.address ? 1 : 0
    })

    let floor: FloorCell = null
    let total = 0n
    for (const s of samples) {
      total += s.balanceWei
      if (floor === null || s.balanceWei < BigInt(floor.balanceWei)) {
        floor = {
          balanceWei: s.balanceWei.toString(),
          address: s.address,
          chainId: s.chainId,
        }
      }
    }

    return c.json({
      mode,
      chains,
      eoas,
      summary: { floor, totalWei: total.toString() },
    })
  })
}
