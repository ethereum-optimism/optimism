import { chainById } from '@eth-optimism/viem/chains'
import { Hono } from 'hono'
import {
  and,
  count,
  desc,
  eq,
  gt,
  inArray,
  isNull,
  replaceBigInts,
  sql,
} from 'ponder'
import { db, publicClients } from 'ponder:api'
import schema from 'ponder:schema'
import { isAddress } from 'viem'

import { expiryCutoffSeconds } from './expiry.js'

const DEFAULT_LIMIT = 100
const MAX_LIMIT = 10000

function parsePagination(c: {
  req: { query: (key: string) => string | undefined }
}): { limit: number; offset: number } {
  const parsedLimit = parseInt(c.req.query('limit') ?? '', 10)
  const parsedOffset = parseInt(c.req.query('offset') ?? '', 10)
  const limit =
    Number.isFinite(parsedLimit) && parsedLimit >= 1
      ? Math.min(parsedLimit, MAX_LIMIT)
      : DEFAULT_LIMIT
  const offset =
    Number.isFinite(parsedOffset) && parsedOffset >= 0 ? parsedOffset : 0
  return { limit, offset }
}

const app = new Hono()

interface ChainInfo {
  id: number
  name: string
  url: string
}

// The chains ponder is actually indexing — one publicClient per configured
// chain — not every chain in viem's registry. Keeps /chains (and everything that
// derives its chain set from it: the relayer, the viewer) scoped to the active
// network. ponder 0.16's publicClients use a custom transport with no URL and no
// attached chain, so we read the id from each client (eth_chainId) and resolve
// name/RPC URL from viem's registry. Cached after the first resolve.
let chainsCache: ChainInfo[] | null = null
async function getIndexedChains(): Promise<ChainInfo[]> {
  if (chainsCache) return chainsCache
  const clients = Object.values(publicClients) as Array<{
    getChainId: () => Promise<number>
  }>
  const ids = await Promise.all(clients.map((client) => client.getChainId()))
  chainsCache = ids.map((id) => {
    const vc = (
      chainById as Record<
        number,
        { name: string; rpcUrls: { default: { http: readonly string[] } } }
      >
    )[id]
    return {
      id,
      name: vc?.name ?? `chain-${id}`,
      url: vc?.rpcUrls.default.http[0] ?? '',
    }
  })
  return chainsCache
}

// List of interoperable chains (only those being indexed).
app.get('/chains', async (c) => {
  return c.json(await getIndexedChains())
})

app.get('/schema', async (c) => {
  return c.json(process.env.DATABASE_SCHEMA)
})

// Count of all messages (sent, relayed, pending)
app.get('/messages/count', async (c) => {
  const chainIds = (await getIndexedChains()).map((ch) => BigInt(ch.id))
  const sent = await db
    .select({ count: count() })
    .from(schema.sentMessages)
    .where(inArray(schema.sentMessages.destination, chainIds))

  const relayed = await db
    .select({ count: count() })
    .from(schema.relayedMessages)

  let pending = undefined
  if (sent.length > 0 && sent.length === relayed.length) {
    pending = sent[0]!.count - relayed[0]!.count
  }

  return c.json({ sent, relayed, pending })
})

// List of pending messages (oldest first). Excludes messages past the interop
// expiry window — see EXPIRY_WINDOW_SECONDS / expiryCutoffSeconds().
app.get('/messages/pending', async (c) => {
  const { limit, offset } = parsePagination(c)
  const result = await db
    .select()
    .from(schema.sentMessages)
    .limit(limit)
    .offset(offset)
    .leftJoin(
      schema.relayedMessages,
      eq(schema.sentMessages.messageHash, schema.relayedMessages.messageHash),
    )
    .where(
      and(
        isNull(schema.relayedMessages.messageHash),
        gt(schema.sentMessages.timestamp, expiryCutoffSeconds()),
      ),
    )
    .orderBy(schema.sentMessages.timestamp)

  const messages = result
    .map((m) => m.l2_to_l2_cdm_sent_messages)
    .map((m) => replaceBigInts(m, (x) => Number(x)))

  return c.json(messages)
})

// List of pending messages for an account
app.get('/messages/:account/pending', async (c) => {
  const account = c.req.param('account').toLowerCase()
  if (!account || !isAddress(account)) {
    return c.json({ error: 'Invalid account' }, 400)
  }

  const { limit, offset } = parsePagination(c)
  const result = await db
    .select()
    .from(schema.sentMessages)
    .limit(limit)
    .offset(offset)
    .leftJoin(
      schema.relayedMessages,
      eq(schema.sentMessages.messageHash, schema.relayedMessages.messageHash),
    )
    .where(
      and(
        isNull(schema.relayedMessages.messageHash),
        eq(schema.sentMessages.sender, account),
        gt(schema.sentMessages.timestamp, expiryCutoffSeconds()),
      ),
    )
    .orderBy(desc(schema.sentMessages.timestamp))

  const messages = result
    .map((m) => m.l2_to_l2_cdm_sent_messages)
    .map((m) => replaceBigInts(m, (x) => Number(x)))

  return c.json(messages)
})

// Deposit endpoints

// Single depositor balance (aggregated across all chains)
app.get('/deposits/:address', async (c) => {
  const address = c.req.param('address').toLowerCase() as `0x${string}`
  if (!address || !isAddress(address)) {
    return c.json({ error: 'Invalid address' }, 400)
  }

  const result = await db
    .select({
      depositor: schema.deposits.depositor,
      totalBalance: sql<bigint>`sum(${schema.deposits.balance})`,
    })
    .from(schema.deposits)
    .where(eq(schema.deposits.depositor, address))
    .groupBy(schema.deposits.depositor)

  if (result.length === 0) {
    return c.json({ depositor: address, totalBalance: '0', eligible: false })
  }

  const row = result[0]!
  return c.json(
    replaceBigInts(
      {
        depositor: row.depositor,
        totalBalance: row.totalBalance,
        eligible: row.totalBalance > 0n,
      },
      (x) => String(x),
    ),
  )
})

// List all depositors with aggregate balances
app.get('/deposits', async (c) => {
  const { limit, offset } = parsePagination(c)
  const result = await db
    .select({
      depositor: schema.deposits.depositor,
      totalBalance: sql<bigint>`sum(${schema.deposits.balance})`,
    })
    .from(schema.deposits)
    .groupBy(schema.deposits.depositor)
    .limit(limit)
    .offset(offset)

  return c.json(result.map((r) => replaceBigInts(r, (x) => String(x))))
})

// Promise endpoints

// List of pending promises (consumed by the relayer's PromiseModule).
app.get('/promises/pending', async (c) => {
  const { limit, offset } = parsePagination(c)
  const result = await db
    .select()
    .from(schema.promises)
    .where(eq(schema.promises.status, 'pending'))
    .limit(limit)
    .offset(offset)
    .orderBy(desc(schema.promises.createdAt))

  return c.json(result.map((p) => replaceBigInts(p, (x) => Number(x))))
})

// Resolved promises that have callbacks on chains where the promise is NOT yet
// resolved. Server-side filter so the relayer only sees work that still needs
// doing. Strategy: start from callbacks, find parents resolved somewhere but
// not on the callback's chain (a resolved row appears on a destination once
// receiveSharedPromise → PromiseResolved is indexed there).
app.get('/promises/unshared-resolved', async (c) => {
  const allCallbacks = await db
    .select({
      parentPromiseId: schema.callbacks.parentPromiseId,
      callbackChainId: schema.callbacks.chainId,
    })
    .from(schema.callbacks)

  if (allCallbacks.length === 0) return c.json([])

  // Group callback chains by parent promise.
  const callbacksByParent = new Map<string, bigint[]>()
  for (const cb of allCallbacks) {
    const existing = callbacksByParent.get(cb.parentPromiseId) ?? []
    existing.push(cb.callbackChainId)
    callbacksByParent.set(cb.parentPromiseId, existing)
  }

  const unshared = []
  for (const [parentPromiseId, callbackChainIds] of callbacksByParent) {
    const promiseRows = await db
      .select()
      .from(schema.promises)
      .where(eq(schema.promises.promiseId, parentPromiseId as `0x${string}`))

    // Need at least one resolved row to share from.
    const resolvedRow = promiseRows.find((r) => r.status === 'resolved')
    if (!resolvedRow) continue

    // Chains where this promise is already resolved.
    const resolvedChainIds = new Set(
      promiseRows.filter((r) => r.status === 'resolved').map((r) => r.chainId),
    )

    // Callback chains that don't yet have a resolved row.
    const pendingChainIds = [...new Set(callbackChainIds)].filter(
      (chainId) => !resolvedChainIds.has(chainId),
    )

    if (pendingChainIds.length > 0) {
      unshared.push({
        ...replaceBigInts(resolvedRow, (x) => Number(x)),
        pendingChainIds: pendingChainIds.map(Number),
      })
    }
  }

  return c.json(unshared)
})

// ---------------------------------------------------------------------------
// Promise-chain views (consumed by the promise-viz viewer). A "chain" is the
// tree of callbacks rooted at a promise, flattened with depth + parent links,
// cross-chain home/resolution rollups, and decoded callback targets.
// ---------------------------------------------------------------------------

// Dedup callbacks that were indexed on multiple chains (a cross-chain callback
// registration lands on both origin and destination). Prefer the home chain.
function deduplicateCallbacks<
  T extends { callbackPromiseId: string; chainId: bigint },
>(allCallbacks: T[], promiseHomeChains: Map<string, number>): T[] {
  const seen = new Map<string, T>()
  for (const cb of allCallbacks) {
    const existing = seen.get(cb.callbackPromiseId)
    if (!existing) {
      seen.set(cb.callbackPromiseId, cb)
    } else {
      const home = promiseHomeChains.get(cb.callbackPromiseId)
      if (home !== undefined && Number(cb.chainId) === home) {
        seen.set(cb.callbackPromiseId, cb)
      }
    }
  }
  return Array.from(seen.values())
}

// "Home" chain = where the promise actually runs. Transferred promises live on
// their transfer destination; otherwise the chain with the earliest createdAt
// (the original PromiseCreated precedes any later receiveSharedPromise row).
function buildPromiseHomeChains(
  allPromises: Array<{ promiseId: string; chainId: bigint; createdAt: bigint }>,
  resolutionTransfers: Array<{
    promiseId: string
    destinationChainId: bigint
  }>,
): Map<string, number> {
  const homeChains = new Map<string, number>()
  for (const rt of resolutionTransfers) {
    homeChains.set(rt.promiseId, Number(rt.destinationChainId))
  }
  const earliest = new Map<string, { chainId: number; createdAt: bigint }>()
  for (const p of allPromises) {
    if (homeChains.has(p.promiseId)) continue // set by transfer
    const existing = earliest.get(p.promiseId)
    if (!existing || p.createdAt < existing.createdAt) {
      earliest.set(p.promiseId, {
        chainId: Number(p.chainId),
        createdAt: p.createdAt,
      })
    }
  }
  for (const [pid, info] of earliest) homeChains.set(pid, info.chainId)
  return homeChains
}

// Best status for a promise across all chains (resolved/rejected win over
// transferred/pending).
function buildPromiseStatusMap(
  allPromises: Array<{
    promiseId: string
    status: string
    resolvedAt: bigint | null
    createdAt: bigint
    returnData: string | null
    resolvedBy: string | null
    resolver: string
  }>,
): Map<
  string,
  {
    status: string
    resolvedAt: number | null
    createdAt: number
    returnData: string | null
    resolvedBy: string | null
    resolver: string
  }
> {
  const statusMap = new Map<
    string,
    {
      status: string
      resolvedAt: number | null
      createdAt: number
      returnData: string | null
      resolvedBy: string | null
      resolver: string
    }
  >()
  for (const p of allPromises) {
    const existing = statusMap.get(p.promiseId)
    const isBetter =
      !existing ||
      p.status === 'resolved' ||
      p.status === 'rejected' ||
      (existing.status === 'transferred' && p.status === 'pending')
    if (isBetter) {
      statusMap.set(p.promiseId, {
        status: p.status,
        resolvedAt: p.resolvedAt ? Number(p.resolvedAt) : null,
        createdAt: Number(p.createdAt),
        returnData: p.returnData,
        resolvedBy: p.resolvedBy,
        resolver: p.resolver,
      })
    }
  }
  return statusMap
}

// Which chains a promise is resolved on (a resolved row per chain).
function buildResolvedOnChainsMap<
  T extends { promiseId: string; chainId: bigint; status: string },
>(allPromises: T[]): Map<string, number[]> {
  const map = new Map<string, number[]>()
  for (const p of allPromises) {
    if (p.status === 'resolved') {
      const chains = map.get(p.promiseId) || []
      chains.push(Number(p.chainId))
      map.set(p.promiseId, chains)
    }
  }
  return map
}

// A single promise chain, given any promiseId in it. Walks up to the root then
// flattens the callback tree depth-first.
app.get('/promise-chain/:promiseId', async (c) => {
  const promiseId = c.req.param('promiseId') as `0x${string}`

  const allCallbacks = await db.select().from(schema.callbacks)
  const allPromises = await db.select().from(schema.promises)
  const allTransfers = await db.select().from(schema.ResolutionTransferred)

  const promiseHomeChains = buildPromiseHomeChains(allPromises, allTransfers)
  const promiseStatusMap = buildPromiseStatusMap(allPromises)
  const resolvedOnChainsMap = buildResolvedOnChainsMap(allPromises)
  const dedupedCallbacks = deduplicateCallbacks(allCallbacks, promiseHomeChains)

  const childToParent = new Map<string, string>()
  const parentToChildren = new Map<string, typeof dedupedCallbacks>()
  const callbackByPromiseId = new Map<
    string,
    (typeof dedupedCallbacks)[number]
  >()
  for (const cb of dedupedCallbacks) {
    childToParent.set(cb.callbackPromiseId, cb.parentPromiseId)
    const children = parentToChildren.get(cb.parentPromiseId) || []
    children.push(cb)
    parentToChildren.set(cb.parentPromiseId, children)
    callbackByPromiseId.set(cb.callbackPromiseId, cb)
  }

  // Walk up to the root.
  let root: string = promiseId
  while (childToParent.has(root)) root = childToParent.get(root)!

  const chain: Array<Record<string, unknown>> = []
  function walk(
    pid: string,
    depth: number,
    parentId: string | null,
    cbType: number | null,
  ) {
    const info = promiseStatusMap.get(pid)
    const cb = callbackByPromiseId.get(pid)
    chain.push({
      promiseId: pid,
      homeChainId: promiseHomeChains.get(pid) || 0,
      status: info?.status || 'unknown',
      depth,
      parentPromiseId: parentId,
      callbackType: cbType,
      callbackTarget: cb?.target || null,
      callbackSelector: cb?.selector || null,
      realTarget: cb?.realTarget || null,
      realSelector: cb?.realSelector || null,
      isDelegateScript: cb?.isDelegateScript ?? null,
      resolvedAt: info?.resolvedAt || null,
      createdAt: info?.createdAt || null,
      resolvedOnChains: resolvedOnChainsMap.get(pid) || [],
      returnData: info?.returnData || null,
      resolvedBy: info?.resolvedBy || null,
      resolver: info?.resolver || null,
    })
    for (const child of parentToChildren.get(pid) || []) {
      walk(child.callbackPromiseId, depth + 1, pid, child.callbackType)
    }
  }
  walk(root, 0, null, null)

  const chainIds = [...new Set(chain.map((n) => n.homeChainId))].sort()
  return c.json({ root, chainIds, promises: chain })
})

// All promise chains grouped by root, with resolved/pending counts.
app.get('/promise-chains', async (c) => {
  const allCallbacks = await db.select().from(schema.callbacks)
  const allPromises = await db.select().from(schema.promises)
  const allTransfers = await db.select().from(schema.ResolutionTransferred)

  const promiseHomeChains = buildPromiseHomeChains(allPromises, allTransfers)
  const promiseStatusMap = buildPromiseStatusMap(allPromises)
  const dedupedCallbacks = deduplicateCallbacks(allCallbacks, promiseHomeChains)

  const childToParent = new Map<string, string>()
  const parentToChildren = new Map<string, typeof dedupedCallbacks>()
  for (const cb of dedupedCallbacks) {
    childToParent.set(cb.callbackPromiseId, cb.parentPromiseId)
    const children = parentToChildren.get(cb.parentPromiseId) || []
    children.push(cb)
    parentToChildren.set(cb.parentPromiseId, children)
  }

  // Roots = parents that are never a child.
  const roots = new Set<string>()
  for (const cb of dedupedCallbacks) {
    if (!childToParent.has(cb.parentPromiseId)) roots.add(cb.parentPromiseId)
  }
  // Plus standalone promises (no callbacks at all).
  const childIds = new Set(dedupedCallbacks.map((cb) => cb.callbackPromiseId))
  const parentIds = new Set(dedupedCallbacks.map((cb) => cb.parentPromiseId))
  for (const p of allPromises) {
    if (!childIds.has(p.promiseId) && !parentIds.has(p.promiseId)) {
      roots.add(p.promiseId)
    }
  }

  function countChain(pid: string): {
    total: number
    resolved: number
    rejected: number
    promiseIds: string[]
  } {
    const status = promiseStatusMap.get(pid)?.status || 'unknown'
    let total = 1
    let resolved = status === 'resolved' ? 1 : 0
    // A `.catch()` on a successful chain settles as 'rejected'. Report it
    // separately so the UI can count settled (resolved + rejected) promises —
    // error handlers stay part of the chain's total, not dropped.
    let rejected = status === 'rejected' ? 1 : 0
    const promiseIds = [pid]
    for (const child of parentToChildren.get(pid) || []) {
      const sub = countChain(child.callbackPromiseId)
      total += sub.total
      resolved += sub.resolved
      rejected += sub.rejected
      promiseIds.push(...sub.promiseIds)
    }
    return { total, resolved, rejected, promiseIds }
  }

  const chains = Array.from(roots).map((root) => {
    const { total, resolved, rejected, promiseIds } = countChain(root)
    return {
      root,
      length: total,
      resolved,
      rejected,
      pending: total - resolved - rejected,
      // Root's creation time — lets the UI identify the most recent chain.
      createdAt: promiseStatusMap.get(root)?.createdAt ?? 0,
      promiseIds,
    }
  })

  return c.json(chains)
})

export default app
