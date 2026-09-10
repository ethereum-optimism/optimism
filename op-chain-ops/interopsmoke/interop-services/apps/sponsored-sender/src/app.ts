import { App } from '@eth-optimism/utils-app'
import { serve } from '@hono/node-server'
import { Option } from 'commander'
import type { Account, Hex, PublicClient } from 'viem'
import { createPublicClient, http, isHex } from 'viem'
import { privateKeyToAccount } from 'viem/accounts'
import { z } from 'zod'

import { createApiRouter } from '@/api.js'
import type { JsonRpcHandler } from '@/jsonrpc/handler.js'
import { senderJsonRpcHandler } from '@/sender/senderJsonRpcHandler.js'

const ConfigSchema = z.object({
  port: z.number().min(1024).max(65535),
  endpoints: z.array(z.string().url()).min(1),
  senderPrivateKey: z.string().refine((key) => {
    if (!key) return true
    return isHex(key) && privateKeyToAccount(key) !== undefined
  }, 'private key must be a valid hex string'),
})

export class SponsoredSenderApp extends App {
  protected clients: Record<number, PublicClient> = {}
  protected sender!: Account

  // Must be set on main()
  private server!: ReturnType<typeof serve>

  constructor() {
    super({
      name: 'sponsored-sender',
      version: '0.0.1',
      description: 'A single endpoint for sending sponsored transactions',
    })
  }

  protected additionalOptions(): Option[] {
    return [
      new Option('--port <port>', 'port to listen on').default(3000),
      new Option(
        '--endpoints <endpoints>',
        'comma separated list of endpoints to send transactions to',
      ).default(['http://127.0.0.1:9545', 'http://127.0.0.1:9546']),
      new Option(
        '--sender-private-key <key>',
        'local private key to use for the relayer',
      ).default(
        '0x2a871d0798f97d79848a013d4936a73bf4cc922c825d33c1cf7073dff6d409c6',
      ),
    ]
  }

  protected async preMain(): Promise<void> {
    const { data: config, error } = ConfigSchema.safeParse(this.options)
    if (error) {
      throw new Error(`invalid configuration: ${error}`)
    }

    this.sender = privateKeyToAccount(config.senderPrivateKey as Hex)
    this.logger.debug(`configured sender ${this.sender.address}`)

    // Create clients
    for (const url of config.endpoints) {
      try {
        const client = createPublicClient({ transport: http(url) })
        const chainId = await client.getChainId()
        const balance = await client.getBalance({
          address: this.sender.address,
        })

        if (balance == 0n) {
          throw Error(`zero sender balance: ${chainId}`)
        }

        this.clients[chainId] = client
        this.logger.debug(`configured chain ${chainId}`)
      } catch (err) {
        throw Error(`failed endpoint ${url} check: ${err}`)
      }
    }
  }

  protected async main(): Promise<void> {
    const handlers: Record<number, JsonRpcHandler> = {}
    for (const [chainId, client] of Object.entries(this.clients)) {
      handlers[Number(chainId)] = senderJsonRpcHandler(this.sender, client)
    }

    const api = createApiRouter(this.logger, handlers)

    // serve the api
    this.logger.info(`starting server on port :${this.options.port}`)
    this.server = serve({ fetch: api.fetch, port: this.options.port })

    // unsatisfied promise as long as the server is running
    return new Promise((resolve, reject) => {
      this.server.on('close', resolve)
      this.server.on('error', reject)
    })
  }

  protected async shutdown(): Promise<void> {
    return new Promise((resolve, reject) => {
      this.server.close((error) => {
        if (error) {
          this.logger.error({ error }, 'error closing server')
          reject(error)
        } else {
          resolve()
        }
      })
    })
  }
}
