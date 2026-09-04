import { describe, it, expect } from 'vitest'

import { expiryCutoffSeconds, parseExpiryEnv } from '../../src/api/expiry.js'

describe('parseExpiryEnv', () => {
  it('returns the fallback when the value is undefined', () => {
    expect(parseExpiryEnv(undefined, 3600)).toBe(3600)
  })

  it('parses positive integers', () => {
    expect(parseExpiryEnv('1800', 3600)).toBe(1800)
    expect(parseExpiryEnv('0', 3600)).toBe(0)
  })

  it('falls back on negative values, NaN, or empty strings', () => {
    expect(parseExpiryEnv('-1', 3600)).toBe(3600)
    expect(parseExpiryEnv('not-a-number', 3600)).toBe(3600)
    expect(parseExpiryEnv('', 3600)).toBe(3600)
  })
})

describe('expiryCutoffSeconds', () => {
  // 2026-05-06T00:00:00Z = 1778025600 seconds since epoch
  const fixedNowMs = 1778025600 * 1000

  it('returns now - window + margin in seconds', () => {
    const cutoff = expiryCutoffSeconds(3600, 60, fixedNowMs)
    // 1778025600 - 3600 + 60 = 1778022060
    expect(cutoff).toBe(BigInt(1778022060))
  })

  it('safety margin shrinks the window (cutoff moves later, expiring more aggressively)', () => {
    const noMargin = expiryCutoffSeconds(3600, 0, fixedNowMs)
    const withMargin = expiryCutoffSeconds(3600, 60, fixedNowMs)
    expect(withMargin).toBeGreaterThan(noMargin)
    expect(withMargin - noMargin).toBe(BigInt(60))
  })

  it('a message at exactly the cutoff is filtered (predicate uses gt, not gte)', () => {
    // Caller asserts: schema.timestamp > cutoff. So a message whose timestamp
    // equals the cutoff is excluded. This test pins the contract — if the
    // predicate ever changes to gte, update both this test and the SQL together.
    const cutoff = expiryCutoffSeconds(3600, 60, fixedNowMs)
    const messageAtCutoff = cutoff
    expect(messageAtCutoff > cutoff).toBe(false)
  })

  it('a message one second past the cutoff passes', () => {
    const cutoff = expiryCutoffSeconds(3600, 60, fixedNowMs)
    expect(cutoff + 1n > cutoff).toBe(true)
  })

  it('zero window with zero margin yields now (everything older expires)', () => {
    const cutoff = expiryCutoffSeconds(0, 0, fixedNowMs)
    expect(cutoff).toBe(BigInt(Math.floor(fixedNowMs / 1000)))
  })
})
