import { describe, expect, it } from 'vitest'

import { encodeRawKey } from './encodeRawKey'

describe(encodeRawKey.name, () => {
  it('should return just the raw key if it is less than 32 bytes', () => {
    const rawKey = 'I am 32'
    const encodedKey = encodeRawKey(rawKey)
    expect(encodedKey).toMatchInlineSnapshot(
      '"0x4920616d20333200000000000000000000000000000000000000000000000000"'
    )
  })
  it('should return the keccak256 hash of the raw key if it is more than 32 bytes', () => {
    const rawKey = 'I am way more than 32 bytes long I should be hashed'
    const encodedKey = encodeRawKey(rawKey)
    expect(encodedKey).toMatchInlineSnapshot(
      '"0xc9d5d767710cc45f74c3a9a0c53dc44391a7951604c7ea3bd9116ccff406daff"'
    )
  })
})
