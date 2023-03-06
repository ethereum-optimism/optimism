import { BigNumber } from 'ethers'
import { toUtf8Bytes } from 'ethers/lib/utils.js'
import { expect, describe, it } from 'vitest'

import { WagmiBytes } from '../types/WagmiBytes'
import {
  parseNumber,
  parseAddress,
  parseBool,
  parseString,
  parseAttestationBytes,
} from './parseAttestationBytes'

describe(parseAttestationBytes.name, () => {
  it('works for strings', () => {
    const str = 'Hello World'
    const bytes = BigNumber.from(toUtf8Bytes(str)).toHexString() as WagmiBytes
    expect(parseAttestationBytes(bytes, 'string')).toBe(str)
  })

  it('works for numbers', () => {
    const num = 123
    const bytes = BigNumber.from(num).toHexString() as WagmiBytes
    expect(parseAttestationBytes(bytes, 'number')).toMatchInlineSnapshot(`
      {
        "hex": "0x7b",
        "type": "BigNumber",
      }
    `)
  })

  it('works for addresses', () => {
    const addr = '0x1234567890123456789012345678901234567890'
    const bytes = BigNumber.from(addr).toHexString() as WagmiBytes
    expect(parseAttestationBytes(bytes, 'address')).toBe(addr)
  })

  it('works for booleans', () => {
    const bytes = BigNumber.from(1).toHexString() as WagmiBytes
    expect(parseAttestationBytes(bytes, 'bool')).toBe(true)
  })

  it('should work for raw bytes', () => {
    expect(parseAttestationBytes('0x420', 'bytes')).toMatchInlineSnapshot(
      '"0x420"'
    )
    expect(parseAttestationBytes('0x', 'string')).toMatchInlineSnapshot('""')
    expect(parseAttestationBytes('0x0', 'string')).toMatchInlineSnapshot('""')
  })

  it('should return raw bytes for invalid type', () => {
    const bytes = '0x420'
    // @ts-expect-error - this is a test for an error case
    expect(parseAttestationBytes(bytes, 'foo')).toBe(bytes)
  })
})

describe('parseFoo', () => {
  it('works for strings', () => {
    const str = 'Hello World'
    const bytes = BigNumber.from(toUtf8Bytes(str)).toHexString() as WagmiBytes
    expect(parseString(bytes)).toBe(str)
    expect(parseString('0x')).toMatchInlineSnapshot('""')
    expect(parseString('0x0')).toMatchInlineSnapshot('""')
    expect(parseString('0x0')).toMatchInlineSnapshot('""')
  })

  it('works for numbers', () => {
    const num = 123
    const bytes = BigNumber.from(num).toHexString() as WagmiBytes
    expect(parseNumber(bytes)).toEqual(BigNumber.from(num))
    expect(parseNumber('0x')).toEqual(BigNumber.from(0))
  })

  it('works for addresses', () => {
    const addr = '0x1234567890123456789012345678901234567890'
    const bytes = BigNumber.from(addr).toHexString() as WagmiBytes
    expect(parseAddress(bytes)).toBe(addr)
  })

  it('works for booleans', () => {
    const bytes = BigNumber.from(1).toHexString() as WagmiBytes
    expect(parseBool(bytes)).toBe(true)
    expect(parseBool('0x')).toBe(false)
    expect(parseBool('0x0')).toBe(false)
    expect(parseBool('0x00000')).toBe(false)
  })
})
