import { BigNumber } from 'ethers'
import { toUtf8Bytes } from 'ethers/lib/utils.js'
import { expect, describe, it } from 'vitest'

import { WagmiBytes } from '../types/WagmiBytes'
import { parseAttestationBytes } from './parseAttestationBytes'

describe(parseAttestationBytes.name, () => {
  it('works for strings', () => {
    const str = 'Hello World'
    const bytes = BigNumber.from(toUtf8Bytes(str)).toHexString() as WagmiBytes
    expect(parseAttestationBytes(bytes, 'string')).toBe(str)
  })

  it('works for numbers', () => {
    const num = 123
    const bytes = BigNumber.from(num).toHexString() as WagmiBytes
    expect(parseAttestationBytes(bytes, 'number')).toBe(num.toString())
  })

  it('works for addresses', () => {
    const addr = '0x1234567890123456789012345678901234567890'
    const bytes = BigNumber.from(addr).toHexString() as WagmiBytes
    expect(parseAttestationBytes(bytes, 'address')).toBe(addr)
  })

  it('works for booleans', () => {
    const bytes = BigNumber.from(1).toHexString() as WagmiBytes
    expect(parseAttestationBytes(bytes, 'bool')).toBe('true')
  })

  it('should work for raw bytes', () => {
    const bytes = '0x420'
    expect(parseAttestationBytes(bytes, 'bytes')).toBe(bytes)
  })

  it('should return raw bytes for invalid type', () => {
    const bytes = '0x420'
    // @ts-expect-error - this is a test for an error case
    expect(parseAttestationBytes(bytes, 'foo')).toBe(bytes)
  })
})
