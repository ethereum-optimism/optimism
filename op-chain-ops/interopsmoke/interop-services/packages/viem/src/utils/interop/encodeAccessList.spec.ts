import { maxUint64 } from 'viem'
import { describe, expect, it } from 'vitest'

import { contracts } from '@/contracts.js'
import type { MessageIdentifier } from '@/types/interop/executingMessage.js'

import { encodeAccessList } from './encodeAccessList.js'

describe('encodeAccessList', () => {
  const baseIdentifier: MessageIdentifier = {
    origin: '0x4200000000000000000000000000000000000023',
    blockNumber: 14n,
    logIndex: 2n,
    timestamp: 1743801675n,
    chainId: 901n,
  }

  const messagePayload =
    '0x382409ac69001e11931a28435afef442cbfd20d9891907e8fa373ba7d351f3200000000000000000000000000000000000000000000000000000000000000386000000000000000000000000420000000000000000000000000000000000002800000000000000000000000000000000000000000000000000000000000000000000000000000000000000004200000000000000000000000000000000000028000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000847cfd6dbc000000000000000000000000420beef000000000000000000000000000000001000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb9226600000000000000000000000000000000000000000000000000000000000003e800000000000000000000000000000000000000000000000000000000'

  it('should encode access list with chain ID that fits in uint64', () => {
    const result = encodeAccessList(baseIdentifier, messagePayload)

    expect(result).toEqual([
      {
        address: contracts.crossL2Inbox.address,
        storageKeys: [
          '0x010000000000000000000385000000000000000e0000000067f04d4b00000002',
          '0x03c6d2648cef120ce1d7ccf9f8d4042d6b25ff30a02e22d9ea2a47d2677ccb8d',
        ],
      },
    ])
  })

  it('should encode access list with chain ID greater than maxUint64', () => {
    // Create an identifier with a chain ID greater than maxUint64
    const largeChainIdIdentifier: MessageIdentifier = {
      ...baseIdentifier,
      chainId: maxUint64 + 20n,
    }

    const result = encodeAccessList(largeChainIdIdentifier, messagePayload)

    expect(result).toEqual([
      {
        address: contracts.crossL2Inbox.address,
        storageKeys: [
          '0x010000000000000000000013000000000000000e0000000067f04d4b00000002',
          '0x0200000000000000000000000000000000000000000000000000000000000001',
          '0x03ffd9bcc97d1c56ae1e5727287611dc92241680d1e32e590d919b5b458c9204',
        ],
      },
    ])
  })
})
