import { z } from 'zod'

export interface PaginationParams {
  limit?: number
  offset?: number
}

// Zod schemas for API response validation
const ChainSchema = z.object({
  id: z.number(),
  name: z.string(),
  url: z.string(),
})

const ChainsSchema = z.array(ChainSchema)

const MessageCountSchema = z.object({
  sent: z.array(z.object({ count: z.number() })),
  relayed: z.array(z.object({ count: z.number() })),
  pending: z.number().optional(),
})

const SentMessageSchema = z.object({
  messageIdentifierHash: z.string(),
  messageHash: z.string(),
  source: z.number(),
  destination: z.number(),
  nonce: z.number(),
  sender: z.string(),
  target: z.string(),
  message: z.string(),
  logIndex: z.number(),
  logPayload: z.string(),
  timestamp: z.number(),
  blockNumber: z.number(),
  transactionHash: z.string(),
  txOrigin: z.string(),
})

const SentMessagesSchema = z.array(SentMessageSchema)

const SchemaResponseSchema = z.string()

const DepositBalanceSchema = z.object({
  depositor: z.string(),
  totalBalance: z.string(),
  eligible: z.boolean(),
})

const DepositsListSchema = z.array(
  z.object({
    depositor: z.string(),
    totalBalance: z.string(),
  }),
)

const PonderPromiseSchema = z.object({
  promiseId: z.string(),
  chainId: z.number(),
  resolver: z.string(),
  status: z.enum(['pending', 'resolved', 'rejected', 'transferred']),
  createdAt: z.number(),
  createdBlockNumber: z.number(),
  createdTransactionHash: z.string(),
  transferredAt: z.number().nullable(),
  transferredBlockNumber: z.number().nullable(),
  transferredTransactionHash: z.string().nullable(),
  resolvedAt: z.number().nullable(),
  resolvedBlockNumber: z.number().nullable(),
  resolvedTransactionHash: z.string().nullable(),
})

const PonderPromisesSchema = z.array(PonderPromiseSchema)

// A resolved promise plus the chains that have a waiting callback but no
// resolved row yet — i.e. the destinations that still need the resolution
// shared to them.
const UnsharedResolvedPromiseSchema = PonderPromiseSchema.extend({
  pendingChainIds: z.array(z.number()),
})

const UnsharedResolvedPromisesSchema = z.array(UnsharedResolvedPromiseSchema)

// Types for API responses (derived from schemas)
export type Chain = z.infer<typeof ChainSchema>
export type MessageCount = z.infer<typeof MessageCountSchema>
export type SentMessage = z.infer<typeof SentMessageSchema>
export type DepositBalance = z.infer<typeof DepositBalanceSchema>
export type DepositListItem = z.infer<typeof DepositsListSchema>[number]
export type PonderPromise = z.infer<typeof PonderPromiseSchema>
export type UnsharedResolvedPromise = z.infer<
  typeof UnsharedResolvedPromiseSchema
>

export interface ApiError {
  error: string
}

/**
 * Client for the ponder-interop API
 */
export class PonderInteropClient {
  private readonly baseUrl: string

  constructor(baseUrl: string) {
    // Remove trailing slash if present
    this.baseUrl = baseUrl.replace(/\/$/, '')
  }

  private buildUrl(endpoint: string, params?: PaginationParams): string {
    const url = `${this.baseUrl}${endpoint}`
    if (!params?.limit && !params?.offset) return url
    const searchParams = new URLSearchParams()
    if (params.limit !== undefined)
      searchParams.set('limit', String(params.limit))
    if (params.offset !== undefined)
      searchParams.set('offset', String(params.offset))
    return `${url}?${searchParams.toString()}`
  }

  /**
   * Generic fetch method with error handling and validation
   */
  private async fetch<T>(
    endpoint: string,
    schema: z.ZodSchema<T>,
    params?: PaginationParams,
  ): Promise<T> {
    const url = this.buildUrl(endpoint, params)

    try {
      const response = await fetch(url)

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(
          `HTTP ${response.status}: ${
            (errorData as ApiError)?.error || response.statusText
          }`,
        )
      }

      const data = await response.json()
      const result = schema.safeParse(data)

      if (!result.success) {
        throw new Error(
          `API response validation failed for ${endpoint}: ${result.error.errors
            .map((e) => `{${e.path.join('.')}: ${e.message}}`)
            .join(', ')}`,
        )
      }

      return result.data
    } catch (error) {
      if (error instanceof Error) {
        throw error
      }
      throw new Error(`Failed to fetch ${endpoint}: ${String(error)}`)
    }
  }

  /**
   * Get list of all interoperable chains
   */
  async getChains(): Promise<Chain[]> {
    return this.fetch('/chains', ChainsSchema)
  }

  /**
   * Get database schema information
   */
  async getSchema(): Promise<string> {
    return this.fetch('/schema', SchemaResponseSchema)
  }

  /**
   * Get count of all messages (sent, relayed, pending)
   */
  async getMessageCount(): Promise<MessageCount> {
    return this.fetch('/messages/count', MessageCountSchema)
  }

  /**
   * Get list of pending messages
   */
  async getPendingMessages(params?: PaginationParams): Promise<SentMessage[]> {
    return this.fetch('/messages/pending', SentMessagesSchema, params)
  }

  /**
   * Get list of pending messages for a specific account
   */
  async getPendingMessagesForAccount(
    account: string,
    params?: PaginationParams,
  ): Promise<SentMessage[]> {
    if (!account || !/^0x[a-fA-F0-9]{40}$/.test(account)) {
      throw new Error('Invalid account address')
    }
    return this.fetch(
      `/messages/${account}/pending`,
      SentMessagesSchema,
      params,
    )
  }

  /**
   * Get deposit balance for a specific address (aggregated across all chains)
   */
  async getDepositBalance(address: string): Promise<DepositBalance> {
    if (!address || !/^0x[a-fA-F0-9]{40}$/.test(address)) {
      throw new Error('Invalid address')
    }
    return this.fetch(`/deposits/${address}`, DepositBalanceSchema)
  }

  /**
   * Get list of all depositors with aggregate balances
   */
  async getDeposits(): Promise<DepositListItem[]> {
    return this.fetch('/deposits', DepositsListSchema)
  }

  /**
   * Get list of pending promises (consumed by the relayer's PromiseModule).
   */
  async getPendingPromises(
    params?: PaginationParams,
  ): Promise<PonderPromise[]> {
    return this.fetch('/promises/pending', PonderPromisesSchema, params)
  }

  /**
   * Get resolved promises that have callbacks on chains where the promise is
   * not yet resolved — i.e. promises that still need cross-chain sharing.
   * Consumed by the relayer's CallbackShareModule.
   */
  async getUnsharedResolvedPromises(): Promise<UnsharedResolvedPromise[]> {
    return this.fetch(
      '/promises/unshared-resolved',
      UnsharedResolvedPromisesSchema,
    )
  }
}

/**
 * Create a new PonderInteropClient instance
 */
export function createPonderInteropClient(
  baseUrl: string,
): PonderInteropClient {
  return new PonderInteropClient(baseUrl)
}
