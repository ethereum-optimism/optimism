import { BigNumber } from 'ethers'
import { toUtf8String } from 'ethers/lib/utils.js'

import type { DataTypeOption } from '../types/DataTypeOption'
import type { WagmiBytes } from '../types/WagmiBytes'

export const parseAttestationBytes = (
  attestationBytes: WagmiBytes,
  dataType: DataTypeOption
) => {
  if (dataType === 'bytes') {
    return attestationBytes
  }
  if (dataType === 'number') {
    return BigNumber.from(attestationBytes).toString()
  }
  if (dataType === 'address') {
    return BigNumber.from(attestationBytes).toHexString()
  }
  if (dataType === 'bool') {
    return BigNumber.from(attestationBytes).gt(0) ? 'true' : 'false'
  }
  if (dataType === 'string') {
    return attestationBytes && toUtf8String(attestationBytes)
  }
  console.warn(`unrecognized dataType ${dataType satisfies never}`)
  return attestationBytes
}
