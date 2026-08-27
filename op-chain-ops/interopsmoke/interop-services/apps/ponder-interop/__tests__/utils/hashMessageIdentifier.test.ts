import { describe, it, expect } from 'vitest'
import { hashMessageIdentifier } from '../../src/utils/hashMessageIdentifier'
import type { MessageIdentifier } from '@eth-optimism/viem'
import { encodeAbiParameters, keccak256, parseAbiParameters } from 'viem'

describe('hashMessageIdentifier', () => {
  const sampleMessageIdentifier: MessageIdentifier = {
    origin: '0x1234567890123456789012345678901234567890',
    chainId: 901n,
    logIndex: 5n,
    blockNumber: 1000n,
    timestamp: 1700000000n,
  }

  describe('Basic Functionality', () => {
    it('should hash a message identifier correctly', () => {
      const result = hashMessageIdentifier(sampleMessageIdentifier)

      expect(result).toMatch(/^0x[a-fA-F0-9]{64}$/) // Should be a 32-byte hex string
      expect(typeof result).toBe('string')
    })

    it('should produce consistent hashes for same input', () => {
      const hash1 = hashMessageIdentifier(sampleMessageIdentifier)
      const hash2 = hashMessageIdentifier(sampleMessageIdentifier)

      expect(hash1).toBe(hash2)
    })

    it('should produce different hashes for different inputs', () => {
      const differentIdentifier: MessageIdentifier = {
        ...sampleMessageIdentifier,
        chainId: 902n, // Different chain ID
      }

      const hash1 = hashMessageIdentifier(sampleMessageIdentifier)
      const hash2 = hashMessageIdentifier(differentIdentifier)

      expect(hash1).not.toBe(hash2)
    })
  })

  describe('Field Sensitivity', () => {
    it('should produce different hashes when origin changes', () => {
      const modifiedIdentifier: MessageIdentifier = {
        ...sampleMessageIdentifier,
        origin: '0x0987654321098765432109876543210987654321',
      }

      const originalHash = hashMessageIdentifier(sampleMessageIdentifier)
      const modifiedHash = hashMessageIdentifier(modifiedIdentifier)

      expect(originalHash).not.toBe(modifiedHash)
    })

    it('should produce different hashes when chainId changes', () => {
      const modifiedIdentifier: MessageIdentifier = {
        ...sampleMessageIdentifier,
        chainId: 999n,
      }

      const originalHash = hashMessageIdentifier(sampleMessageIdentifier)
      const modifiedHash = hashMessageIdentifier(modifiedIdentifier)

      expect(originalHash).not.toBe(modifiedHash)
    })

    it('should produce different hashes when logIndex changes', () => {
      const modifiedIdentifier: MessageIdentifier = {
        ...sampleMessageIdentifier,
        logIndex: 10n,
      }

      const originalHash = hashMessageIdentifier(sampleMessageIdentifier)
      const modifiedHash = hashMessageIdentifier(modifiedIdentifier)

      expect(originalHash).not.toBe(modifiedHash)
    })

    it('should produce different hashes when blockNumber changes', () => {
      const modifiedIdentifier: MessageIdentifier = {
        ...sampleMessageIdentifier,
        blockNumber: 2000n,
      }

      const originalHash = hashMessageIdentifier(sampleMessageIdentifier)
      const modifiedHash = hashMessageIdentifier(modifiedIdentifier)

      expect(originalHash).not.toBe(modifiedHash)
    })

    it('should produce different hashes when timestamp changes', () => {
      const modifiedIdentifier: MessageIdentifier = {
        ...sampleMessageIdentifier,
        timestamp: 1700001000n,
      }

      const originalHash = hashMessageIdentifier(sampleMessageIdentifier)
      const modifiedHash = hashMessageIdentifier(modifiedIdentifier)

      expect(originalHash).not.toBe(modifiedHash)
    })
  })

  describe('Edge Cases', () => {
    it('should handle zero values', () => {
      const zeroIdentifier: MessageIdentifier = {
        origin: '0x0000000000000000000000000000000000000000',
        chainId: 0n,
        logIndex: 0n,
        blockNumber: 0n,
        timestamp: 0n,
      }

      const result = hashMessageIdentifier(zeroIdentifier)

      expect(result).toMatch(/^0x[a-fA-F0-9]{64}$/)
      expect(typeof result).toBe('string')
    })

    it('should handle maximum values', () => {
      const maxIdentifier: MessageIdentifier = {
        origin: '0xffffffffffffffffffffffffffffffffffffffff',
        chainId: 2n ** 256n - 1n, // Max uint256
        logIndex: 2n ** 256n - 1n,
        blockNumber: 2n ** 256n - 1n,
        timestamp: 2n ** 256n - 1n,
      }

      const result = hashMessageIdentifier(maxIdentifier)

      expect(result).toMatch(/^0x[a-fA-F0-9]{64}$/)
      expect(typeof result).toBe('string')
    })
  })

  describe('Implementation Correctness', () => {
    it('should match manual encoding and hashing', () => {
      const expectedEncoded = encodeAbiParameters(
        parseAbiParameters('address,uint256,uint256,uint256,uint256'),
        [
          sampleMessageIdentifier.origin,
          sampleMessageIdentifier.blockNumber,
          sampleMessageIdentifier.logIndex,
          sampleMessageIdentifier.timestamp,
          sampleMessageIdentifier.chainId,
        ]
      )

      const expectedHash = keccak256(expectedEncoded)
      const actualHash = hashMessageIdentifier(sampleMessageIdentifier)

      expect(actualHash).toBe(expectedHash)
    })

    it('should use correct parameter order', () => {
      // The function should encode parameters in the order:
      // origin, blockNumber, logIndex, timestamp, chainId

      const testIdentifier: MessageIdentifier = {
        origin: '0x1111111111111111111111111111111111111111',
        chainId: 1n,
        logIndex: 2n,
        blockNumber: 3n,
        timestamp: 4n,
      }

      const manualEncoded = encodeAbiParameters(
        parseAbiParameters('address,uint256,uint256,uint256,uint256'),
        [
          testIdentifier.origin,      // address
          testIdentifier.blockNumber, // uint256
          testIdentifier.logIndex,    // uint256
          testIdentifier.timestamp,   // uint256
          testIdentifier.chainId,     // uint256
        ]
      )

      const manualHash = keccak256(manualEncoded)
      const functionHash = hashMessageIdentifier(testIdentifier)

      expect(functionHash).toBe(manualHash)
    })
  })

  describe('Type Safety', () => {
    it('should accept valid MessageIdentifier', () => {
      const validIdentifier: MessageIdentifier = {
        origin: '0x1234567890123456789012345678901234567890',
        chainId: 1n,
        logIndex: 1n,
        blockNumber: 1n,
        timestamp: 1n,
      }

      expect(() => hashMessageIdentifier(validIdentifier)).not.toThrow()
    })

    it('should handle different numeric types correctly', () => {
      const identifierWithNumbers: MessageIdentifier = {
        origin: '0x1234567890123456789012345678901234567890',
        chainId: BigInt(901),
        logIndex: BigInt('5'),
        blockNumber: BigInt(1000),
        timestamp: BigInt(1700000000),
      }

      const result = hashMessageIdentifier(identifierWithNumbers)

      expect(result).toMatch(/^0x[a-fA-F0-9]{64}$/)
    })
  })

  describe('Real-world Scenarios', () => {
    it('should handle typical mainnet values', () => {
      const mainnetIdentifier: MessageIdentifier = {
        origin: '0x4200000000000000000000000000000000000023', // Typical L2ToL2CrossDomainMessenger
        chainId: 1n, // Ethereum mainnet
        logIndex: 123n,
        blockNumber: 18500000n, // Recent mainnet block
        timestamp: 1700000000n, // Recent timestamp
      }

      const result = hashMessageIdentifier(mainnetIdentifier)

      expect(result).toMatch(/^0x[a-fA-F0-9]{64}$/)
      expect(result).not.toBe('0x0000000000000000000000000000000000000000000000000000000000000000')
    })

    it('should handle L2 chain identifiers', () => {
      const l2Identifiers = [
        { chainId: 10n }, // Optimism
        { chainId: 8453n }, // Base
        { chainId: 7777777n }, // Zora
      ].map(partial => ({
        origin: '0x4200000000000000000000000000000000000023' as `0x${string}`,
        logIndex: 1n,
        blockNumber: 1000n,
        timestamp: 1700000000n,
        ...partial,
      }))

      const hashes = l2Identifiers.map(hashMessageIdentifier)

      // All should be different
      expect(new Set(hashes).size).toBe(hashes.length)

      // All should be valid hashes
      hashes.forEach(hash => {
        expect(hash).toMatch(/^0x[a-fA-F0-9]{64}$/)
      })
    })
  })

  describe('Collision Resistance', () => {
    it('should produce different hashes for similar inputs', () => {
      const baseIdentifier: MessageIdentifier = {
        origin: '0x1234567890123456789012345678901234567890',
        chainId: 1n,
        logIndex: 1n,
        blockNumber: 1n,
        timestamp: 1n,
      }

      // Create variations by incrementing each field
      const variations = [
        { ...baseIdentifier, chainId: 2n },
        { ...baseIdentifier, logIndex: 2n },
        { ...baseIdentifier, blockNumber: 2n },
        { ...baseIdentifier, timestamp: 2n },
      ]

      const hashes = [baseIdentifier, ...variations].map(hashMessageIdentifier)

      // All hashes should be unique
      expect(new Set(hashes).size).toBe(hashes.length)
    })
  })
})