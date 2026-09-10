import * as fs from 'node:fs'
import * as os from 'node:os'
import * as path from 'node:path'

import type { PonderInteropClient } from '@eth-optimism/ponder-interop/client'
import { Hono } from 'hono'
import type { Logger } from 'pino'
import { getAddress } from 'viem'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { attachBudgetRoute } from '@/admin/budget.js'
import { RelayFundsDeposited } from '@/deposit/relayFundsDeposited.js'

const ADDR = '0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48'
const ADDR_CHECKSUM = getAddress(ADDR) // 0xA0b8...eB48 — exercises the lowercase path

const silentLogger: Logger = {
  warn: () => {},
  info: () => {},
  error: () => {},
  debug: () => {},
  trace: () => {},
  fatal: () => {},
} as unknown as Logger

function fakePonderClient(
  totalBalance: string | Error,
): Pick<PonderInteropClient, 'getDepositBalance'> {
  return {
    getDepositBalance: async (address: string) => {
      if (totalBalance instanceof Error) throw totalBalance
      return { depositor: address, totalBalance, eligible: totalBalance !== '0' }
    },
  }
}

function buildApp(deps: {
  relayFunds: RelayFundsDeposited
  ponderClient: Pick<PonderInteropClient, 'getDepositBalance'>
}): Hono {
  const api = new Hono()
  attachBudgetRoute(api, {
    relayFunds: deps.relayFunds,
    ponderClient: deps.ponderClient as PonderInteropClient,
    logger: silentLogger,
  })
  return api
}

describe('GET /budget/:address', () => {
  let tmpDir: string
  let store: RelayFundsDeposited

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'budget-route-test-'))
    store = new RelayFundsDeposited(path.join(tmpDir, 'rf.sqlite'))
  })

  afterEach(() => {
    store.close()
    fs.rmSync(tmpDir, { recursive: true, force: true })
  })

  it('returns deposited / consumed / remaining as decimal strings', async () => {
    store.recordConsumption(ADDR, 1234n)
    const app = buildApp({
      relayFunds: store,
      ponderClient: fakePonderClient('5000'),
    })

    const res = await app.fetch(new Request(`http://x/budget/${ADDR}`))
    expect(res.status).toBe(200)
    expect(await res.json()).toEqual({
      depositor: ADDR,
      totalDepositedWei: '5000',
      consumedWei: '1234',
      remainingWei: '3766',
    })
  })

  it('clamps remaining at 0 when consumed exceeds deposited', async () => {
    store.recordConsumption(ADDR, 9000n)
    const app = buildApp({
      relayFunds: store,
      ponderClient: fakePonderClient('5000'),
    })

    const res = await app.fetch(new Request(`http://x/budget/${ADDR}`))
    expect(res.status).toBe(200)
    const body = (await res.json()) as { remainingWei: string }
    expect(body.remainingWei).toBe('0')
  })

  it('returns all-zeros for a depositor with no deposits and no consumption', async () => {
    const app = buildApp({
      relayFunds: store,
      ponderClient: fakePonderClient('0'),
    })

    const res = await app.fetch(new Request(`http://x/budget/${ADDR}`))
    expect(res.status).toBe(200)
    expect(await res.json()).toEqual({
      depositor: ADDR,
      totalDepositedWei: '0',
      consumedWei: '0',
      remainingWei: '0',
    })
  })

  it('lowercases checksum-cased input before lookup', async () => {
    store.recordConsumption(ADDR, 100n)
    const app = buildApp({
      relayFunds: store,
      ponderClient: fakePonderClient('500'),
    })

    const res = await app.fetch(new Request(`http://x/budget/${ADDR_CHECKSUM}`))
    expect(res.status).toBe(200)
    const body = (await res.json()) as { depositor: string; consumedWei: string }
    expect(body.depositor).toBe(ADDR)
    expect(body.consumedWei).toBe('100')
  })

  it('returns 400 on a malformed address', async () => {
    const app = buildApp({
      relayFunds: store,
      ponderClient: fakePonderClient('0'),
    })

    const res = await app.fetch(new Request('http://x/budget/0xnotanaddress'))
    expect(res.status).toBe(400)
    expect(await res.json()).toEqual({ error: 'invalid address' })
  })

  it('returns 502 when ponder lookup fails', async () => {
    const app = buildApp({
      relayFunds: store,
      ponderClient: fakePonderClient(new Error('ponder down')),
    })

    const res = await app.fetch(new Request(`http://x/budget/${ADDR}`))
    expect(res.status).toBe(502)
    expect(await res.json()).toEqual({ error: 'failed to fetch on-chain balance' })
  })
})
