import * as fs from 'node:fs'
import * as os from 'node:os'
import * as path from 'node:path'

import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { RelayFundsDeposited } from '@/deposit/relayFundsDeposited.js'

const ADDR_A = '0x1111111111111111111111111111111111111111'
const ADDR_B = '0x2222222222222222222222222222222222222222'
const HASH_1 = '0xaaaaaaaa'
const HASH_2 = '0xbbbbbbbb'

describe('RelayFundsDeposited', () => {
  let tmpDir: string
  let dbPath: string
  let store: RelayFundsDeposited

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'relay-funds-test-'))
    dbPath = path.join(tmpDir, 'test.sqlite')
    store = new RelayFundsDeposited(dbPath)
  })

  afterEach(() => {
    store.close()
    fs.rmSync(tmpDir, { recursive: true, force: true })
  })

  describe('consumption', () => {
    it('returns 0n for unknown address', () => {
      expect(store.getConsumed(ADDR_A)).toBe(0n)
    })

    it('accumulates consumption across calls for the same address', () => {
      store.recordConsumption(ADDR_A, 200n)
      store.recordConsumption(ADDR_A, 300n)
      expect(store.getConsumed(ADDR_A)).toBe(500n)
    })

    it('tracks multiple addresses independently', () => {
      store.recordConsumption(ADDR_A, 100n)
      store.recordConsumption(ADDR_B, 200n)
      expect(store.getConsumed(ADDR_A)).toBe(100n)
      expect(store.getConsumed(ADDR_B)).toBe(200n)
    })

    it('lowercases address keys regardless of input case', () => {
      store.recordConsumption(ADDR_A.toUpperCase(), 100n)
      expect(store.getConsumed(ADDR_A)).toBe(100n)
      expect(store.getConsumed(ADDR_A.toUpperCase())).toBe(100n)
    })

    it('returns totalDeposits when nothing consumed', () => {
      expect(store.getRemainingBudget(ADDR_A, 1000n)).toBe(1000n)
    })

    it('returns totalDeposits minus consumed', () => {
      store.recordConsumption(ADDR_A, 300n)
      expect(store.getRemainingBudget(ADDR_A, 1000n)).toBe(700n)
    })

    it('clamps remaining at 0n when consumed exceeds totalDeposits', () => {
      store.recordConsumption(ADDR_A, 1500n)
      expect(store.getRemainingBudget(ADDR_A, 1000n)).toBe(0n)
    })

    it('hasEnoughBudget true when budget >= estimatedCost', () => {
      expect(store.hasEnoughBudget(ADDR_A, 1000n, 500n)).toBe(true)
    })

    it('hasEnoughBudget false when budget < estimatedCost', () => {
      store.recordConsumption(ADDR_A, 900n)
      expect(store.hasEnoughBudget(ADDR_A, 1000n, 200n)).toBe(false)
    })

    it('hasEnoughBudget true at exact boundary (remaining == estimatedCost)', () => {
      store.recordConsumption(ADDR_A, 500n)
      expect(store.hasEnoughBudget(ADDR_A, 1000n, 500n)).toBe(true)
    })

    it('persists across new instance pointing at same DB file', () => {
      store.recordConsumption(ADDR_A, 500n)
      store.close()

      const reopened = new RelayFundsDeposited(dbPath)
      try {
        expect(reopened.getConsumed(ADDR_A)).toBe(500n)
      } finally {
        reopened.close()
      }

      // re-open original handle for afterEach close
      store = new RelayFundsDeposited(dbPath)
    })
  })

  describe('blocked-message state', () => {
    it('markBlocked + getBlockedForOrigin returns the hash', () => {
      store.markBlocked(HASH_1, ADDR_A)
      expect(store.getBlockedForOrigin(ADDR_A)).toEqual([HASH_1])
    })

    it('markBlocked is idempotent for the same hash', () => {
      store.markBlocked(HASH_1, ADDR_A)
      store.markBlocked(HASH_1, ADDR_A)
      expect(store.getBlockedForOrigin(ADDR_A)).toEqual([HASH_1])
    })

    it('returns empty array for origin with no blocked messages', () => {
      expect(store.getBlockedForOrigin(ADDR_A)).toEqual([])
    })

    it('groups multiple hashes under the same origin', () => {
      store.markBlocked(HASH_1, ADDR_A)
      store.markBlocked(HASH_2, ADDR_A)
      const hashes = store.getBlockedForOrigin(ADDR_A)
      expect(hashes).toHaveLength(2)
      expect(hashes).toContain(HASH_1)
      expect(hashes).toContain(HASH_2)
    })

    it('separates blocked messages by origin', () => {
      store.markBlocked(HASH_1, ADDR_A)
      store.markBlocked(HASH_2, ADDR_B)
      expect(store.getBlockedForOrigin(ADDR_A)).toEqual([HASH_1])
      expect(store.getBlockedForOrigin(ADDR_B)).toEqual([HASH_2])
    })

    it('clearBlocked removes the hash', () => {
      store.markBlocked(HASH_1, ADDR_A)
      store.clearBlocked(HASH_1)
      expect(store.getBlockedForOrigin(ADDR_A)).toEqual([])
    })

    it('clearBlocked is a no-op for an unknown hash', () => {
      expect(() => store.clearBlocked(HASH_1)).not.toThrow()
    })

    it('lowercases hashes and origins regardless of input case', () => {
      store.markBlocked(HASH_1.toUpperCase(), ADDR_A.toUpperCase())
      expect(store.getBlockedForOrigin(ADDR_A)).toEqual([HASH_1])
      expect(store.getBlockedForOrigin(ADDR_A.toUpperCase())).toEqual([HASH_1])
      store.clearBlocked(HASH_1.toUpperCase())
      expect(store.getBlockedForOrigin(ADDR_A)).toEqual([])
    })
  })
})
