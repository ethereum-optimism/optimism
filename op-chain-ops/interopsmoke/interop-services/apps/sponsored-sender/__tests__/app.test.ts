import { EventEmitter } from 'node:events'

import type { ServerType } from '@hono/node-server'
import type { Hono } from 'hono'
import type { PublicClient } from 'viem'
import { privateKeyToAccount } from 'viem/accounts'
import { afterEach, describe, expect, it, vi } from 'vitest'

const { serveMock } = vi.hoisted(() => ({ serveMock: vi.fn() }))

vi.mock('@hono/node-server', () => ({ serve: serveMock }))

import { SponsoredSenderApp } from '@/app.js'

const PRIVATE_KEY =
  '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80'

type ServeOptions = {
  fetch: (request: Request) => Response | Promise<Response>
}

const createServer = (): ServerType => {
  const server = new EventEmitter() as ServerType
  server.close = vi.fn((callback) => {
    callback?.()
    return server
  })
  return server
}

class TestSponsoredSenderApp extends SponsoredSenderApp {
  protected override async preMain(): Promise<void> {
    this.sender = privateKeyToAccount(PRIVATE_KEY)
    this.clients[9545] = {
      getChainId: vi.fn().mockResolvedValue(9545),
    } as unknown as PublicClient
  }

  public initializeAdminApiForTest(): Hono {
    return this.initializeAdminApi()
  }
}

describe('SponsoredSenderApp runtime', () => {
  const originalArgv = process.argv
  const originalSigintListeners = process.listeners('SIGINT')
  const originalSigtermListeners = process.listeners('SIGTERM')

  afterEach(() => {
    process.argv = originalArgv
    for (const listener of process.listeners('SIGINT')) {
      if (!originalSigintListeners.includes(listener)) {
        process.removeListener('SIGINT', listener)
      }
    }
    for (const listener of process.listeners('SIGTERM')) {
      if (!originalSigtermListeners.includes(listener)) {
        process.removeListener('SIGTERM', listener)
      }
    }
    serveMock.mockReset()
  })

  it('keeps running after both HTTP servers become ready', async () => {
    const apiServer = createServer()
    serveMock.mockReturnValueOnce(apiServer)

    process.argv = [
      'node',
      'sponsored-sender',
      '--admin-enabled',
      '--admin-port',
      '19001',
      '--port',
      '19000',
    ]

    const app = new TestSponsoredSenderApp()
    let adminApi: Hono | undefined
    const startAdminServer = vi.fn(async () => {
      adminApi = app.initializeAdminApiForTest()
    })
    Object.defineProperty(app, '__startAdminApiServer', {
      value: startAdminServer,
    })

    const runPromise = app.run()
    await vi.waitFor(() => expect(serveMock).toHaveBeenCalledOnce())
    expect(startAdminServer).toHaveBeenCalledWith('19001')

    const apiOptions = serveMock.mock.calls[0][0] as ServeOptions
    expect(apiOptions.fetch).toBeTypeOf('function')

    const readyResponse = await adminApi!.fetch(
      new Request('http://localhost/readyz'),
    )
    expect(await readyResponse.text()).toBe('OK')

    const chainResponse = await apiOptions.fetch(
      new Request('http://localhost/9545', {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          jsonrpc: '2.0',
          id: 1,
          method: 'eth_chainId',
          params: [],
        }),
      }),
    )
    expect(await chainResponse.json()).toMatchObject({ result: '0x2549' })

    let settled = false
    void runPromise.finally(() => {
      settled = true
    })
    await new Promise((resolve) => setImmediate(resolve))
    expect(settled).toBe(false)

    apiServer.emit('close')
    await runPromise
  })
})
