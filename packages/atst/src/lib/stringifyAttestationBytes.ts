import { Address } from '@wagmi/core'
import { BigNumber } from 'ethers'
import {
  hexlify,
  isAddress,
  isHexString,
  toUtf8Bytes,
} from 'ethers/lib/utils.js'

import { WagmiBytes } from '../types/WagmiBytes'

export const stringifyAttestationBytes = (
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
