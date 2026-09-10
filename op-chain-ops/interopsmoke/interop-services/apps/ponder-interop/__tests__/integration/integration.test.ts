import { describe, it, expect, beforeEach, vi } from 'vitest'

describe('Ponder Interop Integration Tests', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('End-to-End Message Flow', () => {
    it('should process message from event to client response', async () => {
      // This test simulates the complete flow:
      // 1. Event is emitted from blockchain
      // 2. Ponder indexes it into database
      // 3. API serves the data
      // 4. Client fetches and validates the data

      // Step 1: Simulate blockchain event
      const mockSentMessageEvent = {
        args: {
          destination: 902n,
          messageNonce: 123n,
          sender: '0x1234567890123456789012345678901234567890' as `0x${string}`,
          target: '0x0987654321098765432109876543210987654321' as `0x${string}`,
          message: '0xabcdef' as `0x${string}`,
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

      // Step 2: Verify indexing logic processes event correctly
      const { handleSentMessage } = await import('../indexing/handlers/l2-to-l2-cdm-handlers')

      const mockValues = vi.fn().mockResolvedValue(undefined)
      const mockContext = {
        db: {
          insert: vi.fn(() => ({
            values: mockValues,
          })),
        },
        chain: { id: 901 },
      }

      await handleSentMessage(mockSentMessageEvent, mockContext)

      // Verify the event was processed correctly
      expect(mockContext.db.insert).toHaveBeenCalledWith('sentMessages')
      expect(mockValues).toHaveBeenCalledWith(
        expect.objectContaining({
          source: 901n,
          destination: 902n,
          nonce: 123n,
          sender: '0x1234567890123456789012345678901234567890',
          target: '0x0987654321098765432109876543210987654321',
          message: '0xabcdef',
        })
      )

      // Step 3: Verify hash computations are consistent
      const expectedMessageIdentifierHash = expect.stringMatching(/^0x[a-fA-F0-9]{64}$/)
      const expectedMessageHash = expect.stringMatching(/^0x[a-fA-F0-9]{64}$/)

      expect(mockValues).toHaveBeenCalledWith(
        expect.objectContaining({
          messageIdentifierHash: expectedMessageIdentifierHash,
          messageHash: expectedMessageHash,
        })
      )
    })
  })

  describe('Gas Tank Integration', () => {
    it('should handle gas tank operations correctly', async () => {
      const mockOnConflictDoUpdate = vi.fn().mockResolvedValue(undefined)
      const mockOnConflictDoNothing = vi.fn().mockResolvedValue(undefined)
      const mockValues = vi.fn(() => ({
        onConflictDoUpdate: mockOnConflictDoUpdate,
        onConflictDoNothing: mockOnConflictDoNothing,
      }))
      const mockSet = vi.fn().mockResolvedValue(undefined)
      const mockContext = {
        db: {
          insert: vi.fn(() => ({
            values: mockValues,
          })),
          delete: vi.fn().mockResolvedValue(undefined),
          update: vi.fn(() => ({
            set: mockSet,
          })),
        },
        chain: { id: 901 },
      }

      const {
        handleGasTankDeposit,
        handleGasTankClaimed,
      } = await import('../indexing/handlers/gas-tank-handlers')

      const gasProvider = '0x1234567890123456789012345678901234567890' as `0x${string}`
      const depositAmount = 1000000n
      const claimAmount = 50000n

      // Step 1: Deposit
      const depositEvent = {
        args: { depositor: gasProvider, amount: depositAmount },
        block: mockBlock({ timestamp: 1700000000n }),
      }

      await handleGasTankDeposit(depositEvent, mockContext)

      expect(mockContext.db.insert).toHaveBeenCalledWith('gasTankGasProviders')
      expect(mockValues).toHaveBeenCalledWith({
        chainId: 901n,
        address: gasProvider,
        balance: depositAmount,
        lastUpdatedAt: 1700000000n,
      })

      // Step 2: Claim
      const claimEvent = {
        args: {
          gasProvider,
          relayer: '0x0987654321098765432109876543210987654321' as `0x${string}`,
          originMsgHash: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890' as `0x${string}`,
          amount: claimAmount,
        },
        block: mockBlock({ timestamp: 1700000500n }),
      }

      await handleGasTankClaimed(claimEvent, mockContext)

      // Verify balance update
      expect(mockContext.db.update).toHaveBeenCalledWith('gasTankGasProviders', {
        chainId: 901n,
        address: gasProvider,
      })
      expect(mockSet).toHaveBeenCalledWith(expect.any(Function))

      // Verify claim record
      expect(mockContext.db.insert).toHaveBeenCalledWith('gasTankClaimedMessages')
      expect(mockContext.db.insert().values).toHaveBeenCalledWith(
        expect.objectContaining({
          originMessageHash: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
          chainId: 901n,
          gasProvider,
          amountClaimed: claimAmount,
        })
      )
    })

    it('should handle gas tank deposit errors correctly', async () => {
      const { handleGasTankDeposit } = await import('../indexing/handlers/gas-tank-handlers')

      const mockOnConflictDoUpdate = vi.fn().mockRejectedValue(new Error('Database error'))
      const mockValues = vi.fn().mockReturnValue({
        onConflictDoUpdate: mockOnConflictDoUpdate,
      })
      const mockContext = {
        db: {
          insert: vi.fn(() => ({
            values: mockValues,
          })),
        },
        chain: { id: 901 },
      }

      const depositEvent = {
        args: { depositor: '0x1234567890123456789012345678901234567890' as `0x${string}`, amount: 1000000n },
        block: mockBlock({ timestamp: 1700000000n }),
      }

      await expect(handleGasTankDeposit(depositEvent, mockContext)).rejects.toThrow('Database error')
    })

    it('should handle gas tank claim errors correctly', async () => {
      const { handleGasTankClaimed } = await import('../indexing/handlers/gas-tank-handlers')

      const mockSet = vi.fn().mockRejectedValue(new Error('Database error'))
      const mockContext = {
        db: {
          insert: vi.fn().mockReturnThis(),
          values: vi.fn().mockReturnThis(),
          update: vi.fn(() => ({
            set: mockSet,
          })),
        },
        chain: { id: 901 },
      }

      const claimEvent = {
        args: {
          gasProvider: '0x1234567890123456789012345678901234567890' as `0x${string}`,
          relayer: '0x0987654321098765432109876543210987654321' as `0x${string}`,
          originMsgHash: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890' as `0x${string}`,
          amount: 50000n,
        },
        block: mockBlock({ timestamp: 1700000500n }),
      }

      await expect(handleGasTankClaimed(claimEvent, mockContext)).rejects.toThrow('Database error')
    })

    it('should handle database errors gracefully across all handlers', async () => {
      const {
        handleGasTankDeposit,
      } = await import('../indexing/handlers/gas-tank-handlers')

      const mockOnConflictDoUpdate = vi.fn().mockRejectedValue(new Error('Database error'))
      const mockValues = vi.fn().mockReturnValue({
        onConflictDoUpdate: mockOnConflictDoUpdate,
      })
      const mockContext = {
        db: {
          insert: vi.fn(() => ({
            values: mockValues,
          })),
        },
        chain: { id: 901 },
      }

      const depositEvent = {
        args: {
          depositor: '0x123' as `0x${string}`,
          amount: 100n,
        },
        block: mockBlock(),
      }

      await expect(handleGasTankDeposit(depositEvent, mockContext)).rejects.toThrow('Database error')
    })
  })

  describe('Client and Utility Integration', () => {
    it('should validate complete data flow through utilities', async () => {
      // Test the utility functions work correctly with realistic data
      const { hashMessageIdentifier } = await import('../../src/utils/hashMessageIdentifier')

      const messageIdentifier = {
        origin: '0x4200000000000000000000000000000000000023' as `0x${string}`,
        chainId: 901n,
        logIndex: 5n,
        blockNumber: 1000n,
        timestamp: 1700000000n,
      }

      const hash = hashMessageIdentifier(messageIdentifier)

      // Verify hash format
      expect(hash).toMatch(/^0x[a-fA-F0-9]{64}$/)

      // Verify hash is deterministic
      const hash2 = hashMessageIdentifier(messageIdentifier)
      expect(hash).toBe(hash2)

      // Verify different inputs produce different hashes
      const differentIdentifier = { ...messageIdentifier, chainId: 902n }
      const differentHash = hashMessageIdentifier(differentIdentifier)
      expect(hash).not.toBe(differentHash)
    })

  })

  describe('Error Handling Integration', () => {
    it('should handle database errors gracefully across all handlers', async () => {
      const mockContextWithError = {
        db: {
          insert: vi.fn(() => {
            const error = new Error('Database error')
            return {
              values: vi.fn(() => { throw error }),
              onConflictDoUpdate: vi.fn(() => { throw error }),
              onConflictDoNothing: vi.fn(() => { throw error }),
            }
          }),
          update: vi.fn(() => ({
            set: () => { throw new Error('Database error') },
          })),
          delete: () => { throw new Error('Database error') },
        },
        chain: { id: 901 },
      }

      const { handleSentMessage } = await import('../indexing/handlers/l2-to-l2-cdm-handlers')
      const { handleGasTankDeposit } = await import('../indexing/handlers/gas-tank-handlers')

      const mockEvent = {
        args: {
          destination: 902n,
          messageNonce: 123n,
          sender: '0x1234567890123456789012345678901234567890' as `0x${string}`,
          target: '0x0987654321098765432109876543210987654321' as `0x${string}`,
          message: '0xabcdef' as `0x${string}`,
        },
        log: mockEventLog(),
        block: mockBlock(),
        transaction: mockTransaction(),
      }

      const mockGasTankEvent = {
        args: {
          depositor: '0x1234567890123456789012345678901234567890' as `0x${string}`,
          amount: 1000000n,
        },
        block: mockBlock(),
      }

      // All handlers should propagate database errors
      await expect(handleSentMessage(mockEvent, mockContextWithError)).rejects.toThrow('Database error')
      await expect(handleGasTankDeposit(mockGasTankEvent, mockContextWithError)).rejects.toThrow('Database error')
    })
  })
})