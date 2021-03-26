/* External Imports */
import { BigNumber } from 'ethers'
import { remove0x } from '@eth-optimism/core-utils'

export const toHexString32 = (
  value: string | number | BigNumber | boolean
): string => {
  if (typeof value === 'string' && value.startsWith('0x')) {
    return '0x' + remove0x(value).padStart(64, '0').toLowerCase()
  } else if (typeof value === 'boolean') {
    return toHexString32(value ? 1 : 0)
  } else {
    return toHexString32(BigNumber.from(value).toHexString())
  }
}
