import * as fs from 'node:fs'
import * as os from 'node:os'
import * as path from 'node:path'

import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { backoffMs, RelayFailureRegistry } from '@/relay/relayFailureRegistry.js'

const HASH_A = '0xaaaaaaaa'
const HASH_B = '0xbbbbbbbb'
const HASH_C = '0xcccccccc'

describe('backoffMs', () => {
  it('returns 0 for non-positive count', () => {
    expect(backoffMs(0)).toBe(0)
    expect(backoffMs(-1)).toBe(0)
  })

  it('doubles per failure starting at 5s', () => {
    expect(backoffMs(1)).toBe(5_000)
    expect(backoffMs(2)).toBe(10_000)
    expect(backoffMs(3)).toBe(20_000)
    expect(backoffMs(4)).toBe(40_000)
  })

  it('caps at 1h', () => {
    expect(backoffMs(20)).toBe(60 * 60_000)
    expect(backoffMs(100)).toBe(60 * 60_000)
  })
})

describe('RelayFailureRegistry', () => {
  let tmpDir: string
  let registry: RelayFailureRegistry

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'relay-failure-registry-test-'))
    registry = new RelayFailureRegistry(path.join(tmpDir, 'registry.sqlite'))
  })

  afterEach(() => {
    registry.close()
    fs.rmSync(tmpDir, { recursive: true, force: true })
  })

  describe('shouldAttempt', () => {
    it('returns true for an unknown hash', () => {
      expect(registry.shouldAttempt(HASH_A)).toBe(true)
    })

    it('returns false within the backoff window after a failure', () => {
      const now = 1_000_000
      registry.recordFailure(HASH_A, 901, 902, 'unknown', now)
      // 5s backoff after first failure; check at 1s elapsed.
      expect(registry.shouldAttempt(HASH_A, now + 1_000)).toBe(false)
    })

    it('returns true once the backoff window has elapsed', () => {
      const now = 1_000_000
      registry.recordFailure(HASH_A, 901, 902, 'unknown', now)
      expect(registry.shouldAttempt(HASH_A, now + 5_000)).toBe(true)
    })

    it('returns false forever once permanent', () => {
      const now = 1_000_000
      registry.recordFailure(HASH_A, 901, 902, 'rpc_rejected', now)
      // Far past any backoff window — permanent never reattempts.
      expect(registry.shouldAttempt(HASH_A, now + 24 * 60 * 60_000)).toBe(false)
    })
  })

  describe('statusFor', () => {
    it("returns 'ready' for an unknown hash", () => {
      expect(registry.statusFor(HASH_A)).toBe('ready')
    })

    it("returns 'in_backoff' within the backoff window of a transient failure", () => {
      const now = 1_000_000
      registry.recordFailure(HASH_A, 901, 902, 'unknown', now)
      expect(registry.statusFor(HASH_A, now + 1_000)).toBe('in_backoff')
    })

    it("returns 'ready' once the backoff window has elapsed", () => {
      const now = 1_000_000
      registry.recordFailure(HASH_A, 901, 902, 'unknown', now)
      expect(registry.statusFor(HASH_A, now + 5_000)).toBe('ready')
    })

    it("returns 'abandoned' for a permanent entry, regardless of backoff age", () => {
      const now = 1_000_000
      registry.recordFailure(HASH_A, 901, 902, 'rpc_rejected', now)
      expect(registry.statusFor(HASH_A, now + 24 * 60 * 60_000)).toBe(
        'abandoned',
      )
    })
  })

  describe('recordFailure / permanent promotion', () => {
    it('marks permanent on first failure for permanent reasons', () => {
      registry.recordFailure(HASH_A, 901, 902, 'rpc_rejected')
      const perm = registry.getPermanent()
      expect(perm).toHaveLength(1)
      expect(perm[0]?.messageHash).toBe(HASH_A)
      expect(perm[0]?.reason).toBe('rpc_rejected')
      expect(perm[0]?.count).toBe(1)
    })

    it('marks permanent on first failure for `expired`', () => {
      registry.recordFailure(HASH_A, 901, 902, 'expired')
      expect(registry.getPermanent()).toHaveLength(1)
    })

    it('does not mark permanent for transient reasons until threshold', () => {
      const now = 1_000_000
      // 9 transient failures — still not permanent.
      for (let i = 0; i < 9; i++) {
        registry.recordFailure(HASH_A, 901, 902, 'unknown', now + i * 1000)
      }
      expect(registry.getPermanent()).toHaveLength(0)

      // 10th transient failure trips count-based permanent.
      registry.recordFailure(HASH_A, 901, 902, 'unknown', now + 10_000)
      const perm = registry.getPermanent()
      expect(perm).toHaveLength(1)
      expect(perm[0]?.count).toBe(10)
      expect(perm[0]?.reason).toBe('unknown') // last_reason
    })

    it('once permanent, stays permanent even with non-permanent reasons', () => {
      registry.recordFailure(HASH_A, 901, 902, 'rpc_rejected')
      // Unlikely sequence, but guard the invariant.
      registry.recordFailure(HASH_A, 901, 902, 'unknown')
      const perm = registry.getPermanent()
      expect(perm).toHaveLength(1)
      expect(perm[0]?.reason).toBe('rpc_rejected')
    })

    it('lowercases hashes regardless of input case', () => {
      registry.recordFailure(HASH_A.toUpperCase(), 901, 902, 'rpc_rejected')
      const perm = registry.getPermanent()
      expect(perm).toHaveLength(1)
      expect(perm[0]?.messageHash).toBe(HASH_A)
    })
  })

  describe('clearFailure', () => {
    it('removes the entry; subsequent shouldAttempt returns true', () => {
      const now = 1_000_000
      registry.recordFailure(HASH_A, 901, 902, 'unknown', now)
      expect(registry.shouldAttempt(HASH_A, now + 1_000)).toBe(false)
      registry.clearFailure(HASH_A)
      expect(registry.shouldAttempt(HASH_A, now + 1_000)).toBe(true)
    })

    it('clearing a permanent entry brings it back to attemptable', () => {
      registry.recordFailure(HASH_A, 901, 902, 'rpc_rejected')
      expect(registry.shouldAttempt(HASH_A)).toBe(false)
      registry.clearFailure(HASH_A)
      expect(registry.shouldAttempt(HASH_A)).toBe(true)
      expect(registry.getPermanent()).toHaveLength(0)
    })
  })

  describe('gc', () => {
    it('drops entries not in the still-pending set', () => {
      registry.recordFailure(HASH_A, 901, 902, 'unknown')
      registry.recordFailure(HASH_B, 901, 902, 'unknown')
      registry.recordFailure(HASH_C, 901, 902, 'unknown')

      const removed = registry.gc(new Set([HASH_B]))
      expect(removed).toBe(2)
      expect(registry.shouldAttempt(HASH_A)).toBe(true) // gone, attemptable
      expect(registry.shouldAttempt(HASH_C)).toBe(true)
      // HASH_B is still in registry and within its backoff
      expect(registry.shouldAttempt(HASH_B, Date.now())).toBe(false)
    })

    it('empty pending set drops everything', () => {
      registry.recordFailure(HASH_A, 901, 902, 'unknown')
      registry.recordFailure(HASH_B, 901, 902, 'unknown')
      const removed = registry.gc(new Set())
      expect(removed).toBe(2)
    })

    it('drops nothing when every entry is still pending', () => {
      registry.recordFailure(HASH_A, 901, 902, 'unknown')
      registry.recordFailure(HASH_B, 901, 902, 'unknown')
      const removed = registry.gc(new Set([HASH_A, HASH_B]))
      expect(removed).toBe(0)
    })

    it('handles pending sets larger than SQLITE_LIMIT_VARIABLE_NUMBER (~999)', () => {
      // Seed a few failures we want to survive.
      const survivors = new Set<string>()
      for (let i = 0; i < 10; i++) {
        const h = `0x${i.toString(16).padStart(40, '0')}`
        registry.recordFailure(h, 901, 902, 'unknown')
        survivors.add(h)
      }

      // Build a pending set with 5000 hashes including the survivors.
      const pending = new Set<string>(survivors)
      for (let i = 0; i < 5000; i++) {
        pending.add(`0xff${i.toString(16).padStart(38, '0')}`)
      }

      // Add one failure NOT in the pending set so we have something to drop.
      registry.recordFailure(HASH_C, 901, 902, 'unknown')

      const removed = registry.gc(pending)
      expect(removed).toBe(1)
      // Survivors still present.
      expect(registry.shouldAttempt(`0x${'0'.repeat(40)}`, Date.now())).toBe(false)
    })

    it('lowercases pending hashes during gc', () => {
      registry.recordFailure(HASH_A, 901, 902, 'unknown')
      const removed = registry.gc(new Set([HASH_A.toUpperCase()]))
      expect(removed).toBe(0)
    })
  })

  describe('getPermanent', () => {
    it('returns one entry per route+reason for permanent rows only', () => {
      registry.recordFailure(HASH_A, 901, 902, 'rpc_rejected')
      registry.recordFailure(HASH_B, 902, 901, 'expired')
      registry.recordFailure(HASH_C, 901, 902, 'unknown') // transient → not permanent

      const perm = registry.getPermanent()
      expect(perm).toHaveLength(2)
      const byHash = new Map(perm.map((p) => [p.messageHash, p]))
      expect(byHash.get(HASH_A)?.reason).toBe('rpc_rejected')
      expect(byHash.get(HASH_B)?.reason).toBe('expired')
      expect(byHash.get(HASH_A)?.source).toBe(901)
      expect(byHash.get(HASH_A)?.destination).toBe(902)
    })

    it('falls back to last_reason when permanent_reason is null (count-based promotion)', () => {
      const now = 1_000_000
      for (let i = 0; i < 10; i++) {
        registry.recordFailure(HASH_A, 901, 902, 'flaky_rpc', now + i * 1000)
      }
      const perm = registry.getPermanent()
      expect(perm).toHaveLength(1)
      expect(perm[0]?.reason).toBe('flaky_rpc')
    })
  })

  describe('getStats', () => {
    it('groups permanent entries by route × reason with count and oldest timestamp', () => {
      registry.recordFailure(HASH_A, 901, 902, 'rpc_rejected', 1_000_000)
      registry.recordFailure(HASH_B, 901, 902, 'rpc_rejected', 1_500_000)
      registry.recordFailure(HASH_C, 902, 901, 'expired', 2_000_000)

      const stats = registry.getStats()
      expect(stats).toHaveLength(2)
      const byKey = new Map(
        stats.map((s) => [`${s.source}|${s.destination}|${s.reason}`, s]),
      )
      const a = byKey.get('901|902|rpc_rejected')
      expect(a?.count).toBe(2)
      expect(a?.oldestLastFailedAt).toBe(1_000_000)
      const b = byKey.get('902|901|expired')
      expect(b?.count).toBe(1)
      expect(b?.oldestLastFailedAt).toBe(2_000_000)
    })

    it('excludes transient (non-permanent) entries', () => {
      registry.recordFailure(HASH_A, 901, 902, 'unknown') // transient
      expect(registry.getStats()).toHaveLength(0)
    })
  })

  describe('persistence across reopens (crash-resilience)', () => {
    it('preserves state after close + reopen of the same DB file', () => {
      const dbPath = path.join(tmpDir, 'persist.sqlite')
      const a = new RelayFailureRegistry(dbPath)
      a.recordFailure(HASH_A, 901, 902, 'rpc_rejected')
      a.recordFailure(HASH_B, 901, 902, 'unknown', 1_000_000)
      a.close()

      const b = new RelayFailureRegistry(dbPath)
      // HASH_A (permanent) — still permanent.
      expect(b.shouldAttempt(HASH_A)).toBe(false)
      expect(b.getPermanent().map((p) => p.messageHash)).toContain(HASH_A)
      // HASH_B (transient) — still in its backoff window relative to original now.
      expect(b.shouldAttempt(HASH_B, 1_000_000 + 1000)).toBe(false)
      // After enough wall-clock, attemptable again.
      expect(b.shouldAttempt(HASH_B, 1_000_000 + 6_000)).toBe(true)
      b.close()
    })
  })
})
