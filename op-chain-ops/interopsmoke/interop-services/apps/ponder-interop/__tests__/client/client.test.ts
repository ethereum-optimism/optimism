import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { PonderInteropClient, createPonderInteropClient } from '../../src/client/index'

// Mock fetch globally
const mockFetch = vi.fn()
global.fetch = mockFetch

describe('PonderInteropClient', () => {
  let client: PonderInteropClient
  const baseUrl = 'http://localhost:3000'

  beforeEach(() => {
    client = new PonderInteropClient(baseUrl)
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('Constructor', () => {
    it('should create client with base URL', () => {
      expect(client).toBeInstanceOf(PonderInteropClient)
    })

    it('should remove trailing slash from base URL', () => {
      const clientWithSlash = new PonderInteropClient('http://localhost:3000/')
      expect(clientWithSlash).toBeInstanceOf(PonderInteropClient)
    })
  })

  describe('Factory Function', () => {
    it('should create client using factory function', () => {
      const factoryClient = createPonderInteropClient(baseUrl)
      expect(factoryClient).toBeInstanceOf(PonderInteropClient)
    })
  })

  describe('getChains', () => {
    it('should fetch and return chains', async () => {
      const mockChains = [
        { id: 901, name: 'Chain 1', url: 'http://localhost:8545' },
        { id: 902, name: 'Chain 2', url: 'http://localhost:8546' },
      ]

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockChains,
      })

      const result = await client.getChains()

      expect(mockFetch).toHaveBeenCalledWith(`${baseUrl}/chains`)
      expect(result).toEqual(mockChains)
    })

    it('should handle fetch errors', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'))

      await expect(client.getChains()).rejects.toThrow('Network error')
    })

    it('should handle HTTP errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        json: async () => ({ error: 'Database connection failed' }),
      })

      await expect(client.getChains()).rejects.toThrow('HTTP 500: Database connection failed')
    })

    it('should handle validation errors', async () => {
      const invalidData = [{ id: 'invalid', name: 'Chain 1' }] // id should be number

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => invalidData,
      })

      await expect(client.getChains()).rejects.toThrow('API response validation failed')
    })
  })

  describe('getSchema', () => {
    it('should fetch and return schema', async () => {
      const mockSchema = 'public'

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockSchema,
      })

      const result = await client.getSchema()

      expect(mockFetch).toHaveBeenCalledWith(`${baseUrl}/schema`)
      expect(result).toBe(mockSchema)
    })
  })

  describe('getMessageCount', () => {
    it('should fetch and return message counts', async () => {
      const mockCount = {
        sent: [{ count: 100 }],
        relayed: [{ count: 75 }],
        pending: 25,
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockCount,
      })

      const result = await client.getMessageCount()

      expect(mockFetch).toHaveBeenCalledWith(`${baseUrl}/messages/count`)
      expect(result).toEqual(mockCount)
    })
  })

  describe('getPendingMessages', () => {
    it('should fetch and return pending messages', async () => {
      const mockMessages = [
        {
          messageIdentifierHash: '0x123',
          messageHash: '0x456',
          source: 901,
          destination: 902,
          nonce: 1,
          sender: '0x1234567890123456789012345678901234567890',
          target: '0x0987654321098765432109876543210987654321',
          message: '0xabcdef',
          logIndex: 1,
          logPayload: '0xdata',
          timestamp: 1700000000,
          blockNumber: 1000,
          transactionHash: '0xabc123',
          txOrigin: '0x1234567890123456789012345678901234567890',
        },
      ]

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockMessages,
      })

      const result = await client.getPendingMessages()

      expect(mockFetch).toHaveBeenCalledWith(`${baseUrl}/messages/pending`)
      expect(result).toEqual(mockMessages)
    })
  })

  describe('getPendingMessagesForAccount', () => {
    it('should fetch pending messages for valid account', async () => {
      const account = '0x1234567890123456789012345678901234567890'
      const mockMessages = [
        {
          messageIdentifierHash: '0x123',
          messageHash: '0x456',
          source: 901,
          destination: 902,
          nonce: 1,
          sender: account,
          target: '0x0987654321098765432109876543210987654321',
          message: '0xabcdef',
          logIndex: 1,
          logPayload: '0xdata',
          timestamp: 1700000000,
          blockNumber: 1000,
          transactionHash: '0xabc123',
          txOrigin: account,
        },
      ]

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockMessages,
      })

      const result = await client.getPendingMessagesForAccount(account)

      expect(mockFetch).toHaveBeenCalledWith(`${baseUrl}/messages/${account}/pending`)
      expect(result).toEqual(mockMessages)
    })

    it('should reject invalid account addresses', async () => {
      const invalidAccount = 'invalid-address'

      await expect(client.getPendingMessagesForAccount(invalidAccount)).rejects.toThrow('Invalid account address')
    })

    it('should reject empty account addresses', async () => {
      await expect(client.getPendingMessagesForAccount('')).rejects.toThrow('Invalid account address')
    })
  })

  describe('Error Handling', () => {
    it('should handle network timeouts', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Request timeout'))

      await expect(client.getChains()).rejects.toThrow('Request timeout')
    })

    it('should handle malformed JSON responses', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => {
          throw new Error('Invalid JSON')
        },
      })

      await expect(client.getChains()).rejects.toThrow('Invalid JSON')
    })

    it('should handle HTTP errors without error message', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        json: async () => ({}), // No error field
      })

      await expect(client.getChains()).rejects.toThrow('HTTP 404: Not Found')
    })
  })

  describe('Response Validation', () => {
    it('should validate chain response schema', async () => {
      const invalidChainData = [
        { id: 901, name: 'Chain 1' }, // missing url
      ]

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => invalidChainData,
      })

      await expect(client.getChains()).rejects.toThrow('API response validation failed')
    })

    it('should validate message response schema', async () => {
      const invalidMessageData = [
        { messageHash: '0x123' }, // missing required fields
      ]

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => invalidMessageData,
      })

      await expect(client.getPendingMessages()).rejects.toThrow('API response validation failed')
    })
  })

  describe('getPendingPromises', () => {
    it('should fetch and return pending promises', async () => {
      const mockPromises = [
        {
          promiseId: '0xaaaa',
          chainId: 901,
          resolver: '0x2222222222222222222222222222222222222222',
          status: 'pending',
          createdAt: 1700000000,
          createdBlockNumber: 5,
          createdTransactionHash: '0xcreate',
          transferredAt: null,
          transferredBlockNumber: null,
          transferredTransactionHash: null,
          resolvedAt: null,
          resolvedBlockNumber: null,
          resolvedTransactionHash: null,
        },
      ]

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockPromises,
      })

      const result = await client.getPendingPromises()

      expect(mockFetch).toHaveBeenCalledWith(`${baseUrl}/promises/pending`)
      expect(result).toEqual(mockPromises)
    })
  })

  describe('getUnsharedResolvedPromises', () => {
    it('should fetch resolved promises with their pending chain ids', async () => {
      const mockUnshared = [
        {
          promiseId: '0xaaaa',
          chainId: 901,
          resolver: '0x0000000000000000000000000000000000000000',
          status: 'resolved',
          createdAt: 1700000000,
          createdBlockNumber: 5,
          createdTransactionHash: '0xcreate',
          transferredAt: null,
          transferredBlockNumber: null,
          transferredTransactionHash: null,
          resolvedAt: 1700000100,
          resolvedBlockNumber: 20,
          resolvedTransactionHash: '0xresolve',
          pendingChainIds: [902],
        },
      ]

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockUnshared,
      })

      const result = await client.getUnsharedResolvedPromises()

      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/promises/unshared-resolved`,
      )
      expect(result).toEqual(mockUnshared)
    })

    it('should reject responses missing pendingChainIds', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => [{ promiseId: '0xaaaa', chainId: 901 }],
      })

      await expect(client.getUnsharedResolvedPromises()).rejects.toThrow(
        'API response validation failed',
      )
    })
  })
})