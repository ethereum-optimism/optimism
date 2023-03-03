import { Address } from '@wagmi/core'
import { BigNumber } from 'ethers'
import {
  hexlify,
  isAddress,
  isHexString,
  toUtf8Bytes,
} from 'ethers/lib/utils.js'

import { WagmiBytes } from '../types/WagmiBytes'

/**
 * Turns a value into bytes to make an attestation
 *
 * @example
 * createValue('hello world') // '0x68656c6c6f20776f726c64'
 * createValue(123) // '0x7b'
 * createValue(true) // '0x1'
 * createValue(BigNumber.from(10)) // '0xa'
 */
export const createValue = (
  bytes: WagmiBytes | string | Address | number | boolean | BigNumber
): WagmiBytes => {
  bytes = bytes === '0x' ? '0x0' : bytes
  if (BigNumber.isBigNumber(bytes)) {
    return bytes.toHexString() as WagmiBytes
  }
  if (typeof bytes === 'number') {
    return BigNumber.from(bytes).toHexString() as WagmiBytes
  }
  if (typeof bytes === 'boolean') {
    return bytes ? '0x1' : '0x0'
  }
  if (isAddress(bytes)) {
    return bytes
  }
  if (isHexString(bytes)) {
    return bytes as WagmiBytes
  }
  if (typeof bytes === 'string') {
    return hexlify(toUtf8Bytes(bytes)) as WagmiBytes
  }
  throw new Error(`unrecognized bytes type ${bytes satisfies never}`)
}

/**
 * @deprecated use createValue instead
 * Will be removed in v1.0.0
 */
export const stringifyAttestationBytes: typeof createValue = (bytes) => {
  console.warn(
    'stringifyAttestationBytes is deprecated, use createValue instead'
  )
  return createValue(bytes)
}
