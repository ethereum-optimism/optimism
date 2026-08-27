import { describe, it, expect, beforeEach, vi } from 'vitest'

// Mock ponder modules
const mockOnConflictDoUpdate = vi.fn().mockResolvedValue(undefined)
const mockOnConflictDoNothing = vi.fn().mockResolvedValue(undefined)
const mockValues = vi.fn(() => ({
  onConflictDoUpdate: mockOnConflictDoUpdate,
  onConflictDoNothing: mockOnConflictDoNothing,
}))
const mockSet = vi.fn().mockResolvedValue(undefined)

const mockDb = {
  insert: vi.fn(() => ({
    values: mockValues,
  })),
  delete: vi.fn().mockResolvedValue(undefined),
  update: vi.fn(() => ({
    set: mockSet,
  })),
}

const mockContext = {
  db: mockDb,
  chain: { id: 901 },
}

describe('Gas Tank Indexing', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Deposit Event Handler', () => {
    it('should correctly index a gas tank deposit', async () => {
      const mockEvent = {
        args: {
          depositor: '0x1234567890123456789012345678901234567890' as `0x${string}`,
          amount: 1000000n,
        },
        block: mockBlock({
          timestamp: 1700000000n,
        }),
      }

      const { handleGasTankDeposit } = await import('./handlers/gas-tank-handlers')
      await handleGasTankDeposit(mockEvent, mockContext)

      expect(mockDb.insert).toHaveBeenCalledWith('gasTankGasProviders')
      expect(mockValues).toHaveBeenCalledWith({
        chainId: 901n,
        address: '0x1234567890123456789012345678901234567890',
        balance: 1000000n,
        lastUpdatedAt: 1700000000n,
      })
      expect(mockOnConflictDoUpdate).toHaveBeenCalledWith(expect.any(Function))
    })

    it('should handle conflicting deposits by updating balance', async () => {
      const mockEvent = {
        args: {
          depositor: '0x1234567890123456789012345678901234567890' as `0x${string}`,
          amount: 500000n,
        },
        block: mockBlock({
          timestamp: 1700000100n,
        }),
      }

      const { handleGasTankDeposit } = await import('./handlers/gas-tank-handlers')
      await handleGasTankDeposit(mockEvent, mockContext)

      // Verify the onConflictDoUpdate function behavior
      const updateFn = mockOnConflictDoUpdate.mock.calls[0]![0]
      const mockRow = { balance: 1000000n }
      const result = updateFn(mockRow)

      expect(result).toEqual({
        balance: 1500000n, // 1000000n + 500000n
        lastUpdatedAt: 1700000100n,
      })
    })
  })

  describe('WithdrawalInitiated Event Handler', () => {
    it('should correctly index a withdrawal initiation', async () => {
      const mockEvent = {
        args: {
          from: '0x1234567890123456789012345678901234567890' as `0x${string}`,
          amount: 200000n,
        },
        block: mockBlock({
          timestamp: 1700000200n,
        }),
      }

      const { handleGasTankWithdrawalInitiated } = await import('./handlers/gas-tank-handlers')
      await handleGasTankWithdrawalInitiated(mockEvent, mockContext)

      expect(mockDb.insert).toHaveBeenCalledWith('gasTankPendingWithdrawals')
      expect(mockValues).toHaveBeenCalledWith({
        chainId: 901n,
        address: '0x1234567890123456789012345678901234567890',
        amount: 200000n,
        initiatedAt: 1700000200n,
      })
      expect(mockOnConflictDoUpdate).toHaveBeenCalledWith({
        amount: 200000n,
        initiatedAt: 1700000200n,
      })
    })
  })

  describe('WithdrawalFinalized Event Handler', () => {
    it('should correctly process a withdrawal finalization', async () => {
      const mockEvent = {
        args: {
          from: '0x1234567890123456789012345678901234567890' as `0x${string}`,
          amount: 200000n,
        },
        block: mockBlock({
          timestamp: 1700000300n,
        }),
      }

      const { handleGasTankWithdrawalFinalized } = await import('./handlers/gas-tank-handlers')
      await handleGasTankWithdrawalFinalized(mockEvent, mockContext)

      // Should delete the pending withdrawal
      expect(mockDb.delete).toHaveBeenCalledWith('gasTankPendingWithdrawals', {
        chainId: 901n,
        address: '0x1234567890123456789012345678901234567890',
      })

      // Should update the gas provider balance
      expect(mockDb.update).toHaveBeenCalledWith('gasTankGasProviders', {
        chainId: 901n,
        address: '0x1234567890123456789012345678901234567890',
      })
      expect(mockSet).toHaveBeenCalledWith(expect.any(Function))

      // Verify the balance update function
      const updateFn = mockSet.mock.calls[0]![0]
      const mockRow = { balance: 1000000n }
      const result = updateFn(mockRow)

      expect(result).toEqual({
        balance: 800000n, // 1000000n - 200000n
        lastUpdatedAt: 1700000300n,
      })
    })
  })

  describe('Flagged Event Handler', () => {
    it('should correctly index a flagged message', async () => {
      const mockEvent = {
        args: {
          gasProvider: '0x1234567890123456789012345678901234567890' as `0x${string}`,
          originMsgHash: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890' as `0x${string}`,
        },
        block: mockBlock({
          timestamp: 1700000400n,
        }),
      }

      const { handleGasTankFlagged } = await import('./handlers/gas-tank-handlers')
      await handleGasTankFlagged(mockEvent, mockContext)

      expect(mockDb.insert).toHaveBeenCalledWith('gasTankFlaggedMessages')
      expect(mockValues).toHaveBeenCalledWith({
        chainId: 901n,
        gasProvider: '0x1234567890123456789012345678901234567890',
        originMessageHash: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
        flaggedAt: 1700000400n,
      })
      expect(mockOnConflictDoNothing).toHaveBeenCalled()
    })
  })

  describe('Claimed Event Handler', () => {
    it('should correctly process a claimed message', async () => {
      const mockEvent = {
        args: {
          gasProvider: '0x1234567890123456789012345678901234567890' as `0x${string}`,
          relayer: '0x0987654321098765432109876543210987654321' as `0x${string}`,
          originMsgHash: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890' as `0x${string}`,
          amount: 50000n,
        },
        block: mockBlock({
          timestamp: 1700000500n,
        }),
      }

      const { handleGasTankClaimed } = await import('./handlers/gas-tank-handlers')
      await handleGasTankClaimed(mockEvent, mockContext)

      // Should update the gas provider balance
      expect(mockDb.update).toHaveBeenCalledWith('gasTankGasProviders', {
        chainId: 901n,
        address: '0x1234567890123456789012345678901234567890',
      })
      expect(mockSet).toHaveBeenCalledWith(expect.any(Function))

      // Should insert the claimed message record
      expect(mockDb.insert).toHaveBeenCalledWith('gasTankClaimedMessages')
      expect(mockValues).toHaveBeenCalledWith({
        originMessageHash: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
        chainId: 901n,
        relayer: '0x0987654321098765432109876543210987654321',
        gasProvider: '0x1234567890123456789012345678901234567890',
        amountClaimed: 50000n,
        claimedAt: 1700000500n,
      })

      // Verify the balance update function
      const updateFn = mockSet.mock.calls[0]![0]
      const mockRow = { balance: 1000000n }
      const result = updateFn(mockRow)

      expect(result).toEqual({
        balance: 950000n, // 1000000n - 50000n
        lastUpdatedAt: 1700000500n,
      })
    })
  })

  describe('RelayedMessageGasReceipt Event Handler', () => {
    it('should correctly index a gas receipt', async () => {
      const mockEvent = {
        args: {
          originMsgHash: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890' as `0x${string}`,
          relayer: '0x0987654321098765432109876543210987654321' as `0x${string}`,
          gasCost: 75000n,
          destinationMessageHashes: [
            '0x1111111111111111111111111111111111111111111111111111111111111111',
            '0x2222222222222222222222222222222222222222222222222222222222222222',
          ] as `0x${string}`[],
        },
        block: mockBlock({
          timestamp: 1700000600n,
        }),
      }

      const { handleGasTankRelayedMessageGasReceipt } = await import('./handlers/gas-tank-handlers')
      await handleGasTankRelayedMessageGasReceipt(mockEvent, mockContext)

      expect(mockDb.insert).toHaveBeenCalledWith('gasTankRelayedMessageReceipts')
      expect(mockValues).toHaveBeenCalledWith({
        originMessageHash: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
        chainId: 901n,
        relayer: '0x0987654321098765432109876543210987654321',
        gasCost: 75000n,
        destinationMessageHashes: [
          '0x1111111111111111111111111111111111111111111111111111111111111111',
          '0x2222222222222222222222222222222222222222222222222222222222222222',
        ],
        relayedAt: 1700000600n,
      })
    })
  })

  describe('Error Handling', () => {
    it('should handle database errors in deposit handler', async () => {
      const mockEvent = {
        args: {
          depositor: '0x1234567890123456789012345678901234567890' as `0x${string}`,
          amount: 1000000n,
        },
        block: mockBlock(),
      }

      // Mock database error
      const mockOnConflictDoUpdate = vi
        .fn()
        .mockRejectedValue(new Error('Database connection failed'))
      mockValues.mockReturnValue({
        onConflictDoUpdate: mockOnConflictDoUpdate,
        onConflictDoNothing: vi.fn(),
      })

      const { handleGasTankDeposit } = await import('./handlers/gas-tank-handlers')

      await expect(handleGasTankDeposit(mockEvent, mockContext)).rejects.toThrow('Database connection failed')
    })

    it('should handle invalid event args', async () => {
      const mockEvent = {
        args: { depositor: '0x123' }, // Missing amount
        block: mockBlock(),
      }

      const { handleGasTankDeposit } = await import('./handlers/gas-tank-handlers')

      // Should throw a TypeError because amount is undefined
      await expect(handleGasTankDeposit(mockEvent, mockContext)).rejects.toThrowError()
    })
  })
})