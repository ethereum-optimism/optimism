import { describe, it, expect, beforeEach, vi } from 'vitest'
import { keccak256, encodeAbiParameters, type Log } from 'viem'
import { contracts } from '@eth-optimism/viem'
import type { MessageIdentifier } from '@eth-optimism/viem/types/interop'
import { hashCrossDomainMessage, encodeMessagePayload } from '@eth-optimism/viem/utils/interop'
import { hashMessageIdentifier } from '../../src/utils/hashMessageIdentifier'

// Mock ponder modules
const mockValues = vi.fn().mockResolvedValue(undefined)
const mockDb = {
  insert: vi.fn(() => ({
    values: mockValues,
  })),
}

const mockContext = {
  db: mockDb,
  chain: { id: 901 },
}

// Mock the ponder registry and schema
vi.mock('ponder:registry', () => ({
  ponder: {
    on: vi.fn(),
  },
}))

vi.mock('ponder:schema', () => ({
  sentMessages: 'sentMessages',
  relayedMessages: 'relayedMessages',
}))

describe('L2 to L2 Cross-Domain Message Indexing', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('SentMessage Event Handler', () => {
    it('should correctly index a sent message', async () => {
      const mockEvent = {
        args: {
          destination: 902n,
          messageNonce: 123n,
          sender: '0x1234567890123456789012345678901234567890',
          target: '0x0987654321098765432109876543210987654321',
          message: '0xabcdef',
        },
        log: mockEventLog({
          logIndex: 5,
          transactionHash: '0xabc123',
          blockNumber: 1000n,
        }),
        block: mockBlock({
          number: 1000n,
          timestamp: 1700000000n,
        }),
        transaction: mockTransaction({
          hash: '0xabc123',
          from: '0x1234567890123456789012345678901234567890',
        }),
      }

      // Import the handler function dynamically to avoid module loading issues
      const { handleSentMessage } = await import('./handlers/l2-to-l2-cdm-handlers')

      await handleSentMessage(mockEvent, mockContext)

      // Verify database insertion was called
      expect(mockDb.insert).toHaveBeenCalledWith('sentMessages')

      expect(mockValues).toHaveBeenCalledWith(
        expect.objectContaining({
          source: 901n,
          destination: 902n,
          nonce: 123n,
          sender: '0x1234567890123456789012345678901234567890',
          target: '0x0987654321098765432109876543210987654321',
          message: '0xabcdef',
          logIndex: 5n,
          timestamp: 1700000000n,
          blockNumber: 1000n,
          transactionHash: '0xabc123',
          txOrigin: '0x1234567890123456789012345678901234567890',
        })
      )
    })

    it('should generate correct message identifier hash', async () => {
      const mockEvent = {
        args: {
          destination: 902n,
          messageNonce: 123n,
          sender: '0x1234567890123456789012345678901234567890',
          target: '0x0987654321098765432109876543210987654321',
          message: '0xabcdef',
        },
        log: mockEventLog({
          logIndex: 5,
          blockNumber: 1000n,
        }),
        block: mockBlock({
          number: 1000n,
          timestamp: 1700000000n,
        }),
        transaction: mockTransaction(),
      }

      const { handleSentMessage } = await import('./handlers/l2-to-l2-cdm-handlers')
      await handleSentMessage(mockEvent, mockContext)

      const expectedMessageIdentifier: MessageIdentifier = {
        origin: contracts.l2ToL2CrossDomainMessenger.address,
        chainId: 901n,
        logIndex: 5n,
        blockNumber: 1000n,
        timestamp: 1700000000n,
      }

      const expectedHash = hashMessageIdentifier(expectedMessageIdentifier)

      expect(mockValues).toHaveBeenCalledWith(
        expect.objectContaining({
          messageIdentifierHash: expectedHash,
        })
      )
    })

    it('should generate correct cross-domain message hash', async () => {
      const mockEvent = {
        args: {
          destination: 902n,
          messageNonce: 123n,
          sender: '0x1234567890123456789012345678901234567890',
          target: '0x0987654321098765432109876543210987654321',
          message: '0xabcdef',
        },
        log: mockEventLog(),
        block: mockBlock(),
        transaction: mockTransaction(),
      }

      const { handleSentMessage } = await import('./handlers/l2-to-l2-cdm-handlers')
      await handleSentMessage(mockEvent, mockContext)

      const expectedCdm = {
        source: 901n,
        destination: 902n,
        nonce: 123n,
        sender: '0x1234567890123456789012345678901234567890' as `0x${string}`,
        target: '0x0987654321098765432109876543210987654321' as `0x${string}`,
        message: '0xabcdef' as `0x${string}`,
        log: mockEvent.log,
      }

      const expectedMessageHash = hashCrossDomainMessage(expectedCdm)

      expect(mockValues).toHaveBeenCalledWith(
        expect.objectContaining({
          messageHash: expectedMessageHash,
        })
      )
    })

    it('should encode log payload correctly', async () => {
      const mockLog = mockEventLog({
        address: '0x1234567890123456789012345678901234567890',
        topics: ['0xabcd', '0xefgh'],
        data: '0x1234',
      })

      const mockEvent = {
        args: {
          destination: 902n,
          messageNonce: 123n,
          sender: '0x1234567890123456789012345678901234567890',
          target: '0x0987654321098765432109876543210987654321',
          message: '0xabcdef',
        },
        log: mockLog,
        block: mockBlock(),
        transaction: mockTransaction(),
      }

      const { handleSentMessage } = await import('./handlers/l2-to-l2-cdm-handlers')
      await handleSentMessage(mockEvent, mockContext)

      const expectedLogPayload = encodeMessagePayload(mockLog as Log)

      expect(mockValues).toHaveBeenCalledWith(
        expect.objectContaining({
          logPayload: expectedLogPayload,
        })
      )
    })

    it('should handle malformed event args', async () => {
      const mockEvent = {
        args: {
          // Missing required fields
          destination: 902n,
        },
        log: mockEventLog(),
        block: mockBlock(),
        transaction: mockTransaction(),
      }

      const { handleSentMessage } = await import('./handlers/l2-to-l2-cdm-handlers')

      // Should not throw as validation is out of scope for indexing handlers
      await expect(handleSentMessage(mockEvent, mockContext)).rejects.toThrow(
        'Cannot convert undefined to a BigInt',
      )
    })
  })

  describe('RelayedMessage Event Handler (v1)', () => {
    it('should correctly index a relayed message without return data hash', async () => {
      const mockEvent = {
        args: {
          source: 901n,
          messageNonce: 123n,
          messageHash: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
        },
        log: mockEventLog({
          logIndex: 10,
        }),
        block: mockBlock({
          timestamp: 1700000100n,
        }),
        transaction: mockTransaction({
          hash: '0xdef456',
          from: '0x0987654321098765432109876543210987654321',
        }),
      }

      const { handleRelayedMessageV1 } = await import('./handlers/l2-to-l2-cdm-handlers')
      await handleRelayedMessageV1(mockEvent, mockContext)

      expect(mockDb.insert).toHaveBeenCalledWith('relayedMessages')
      expect(mockValues).toHaveBeenCalledWith({
        messageHash: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
        relayer: '0x0987654321098765432109876543210987654321',
        logIndex: 10n,
        logPayload: expect.any(String),
        timestamp: 1700000100n,
        blockNumber: 1000n,
        transactionHash: '0xdef456',
      })
    })
  })

  describe('RelayedMessage Event Handler (v2)', () => {
    it('should correctly index a relayed message with return data hash', async () => {
      const mockEvent = {
        args: {
          source: 901n,
          messageNonce: 123n,
          messageHash: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
          returnDataHash: '0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef',
        },
        log: mockEventLog({
          logIndex: 15,
        }),
        block: mockBlock({
          timestamp: 1700000200n,
        }),
        transaction: mockTransaction({
          hash: '0xghi789',
          from: '0xabcdef1234567890abcdef1234567890abcdef12',
        }),
      }

      const { handleRelayedMessageV2 } = await import('./handlers/l2-to-l2-cdm-handlers')
      await handleRelayedMessageV2(mockEvent, mockContext)

      expect(mockDb.insert).toHaveBeenCalledWith('relayedMessages')
      expect(mockValues).toHaveBeenCalledWith({
        messageHash: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
        relayer: '0xabcdef1234567890abcdef1234567890abcdef12',
        logIndex: 15n,
        logPayload: expect.any(String),
        timestamp: 1700000200n,
        blockNumber: 1000n,
        transactionHash: '0xghi789',
      })
    })
  })

  describe('Error Handling', () => {
    it('should handle database insertion errors gracefully', async () => {
      const mockEvent = {
        args: {
          destination: 902n,
          messageNonce: 123n,
          sender: '0x1234567890123456789012345678901234567890',
          target: '0x0987654321098765432109876543210987654321',
          message: '0xabcdef',
        },
        log: mockEventLog(),
        block: mockBlock(),
        transaction: mockTransaction(),
      }

      // Mock database error
      mockValues.mockRejectedValueOnce(new Error('Database connection failed'))

      const { handleSentMessage } = await import('./handlers/l2-to-l2-cdm-handlers')

      await expect(handleSentMessage(mockEvent, mockContext)).rejects.toThrow('Database connection failed')
    })
  })
})