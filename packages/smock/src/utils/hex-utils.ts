/* External Imports */
import { BigNumber } from 'ethers'
import { remove0x } from '@eth-optimism/core-utils'

export const toHexString32 = (
  value: string | number | BigNumber | boolean
): string => {
  if (typeof value === 'string' && value.startsWith('0x')) {
    // Known bug here is that bytes20 and address are indistinguishable but have to be treated
    // differently. Address gets padded on the right, bytes20 gets padded on the left. Address is
    // way more common so I'm going with the strategy of treating all bytes20 like addresses.
    // Sorry to anyone who wants to smodify bytes20 values :-/ requires a bit of rewrite to fix.
    if (value.length === 42) {
      return '0x' + remove0x(value).padStart(64, '0').toLowerCase()
    } else {
      return '0x' + remove0x(value).padEnd(64, '0').toLowerCase()
    }
  } else if (typeof value === 'boolean') {
    return '0x' + `${value ? 1 : 0}`.padStart(64, '0')
  } else {
    return (
      '0x' +
      remove0x(BigNumber.from(value).toHexString())
        .padStart(64, '0')
        .toLowerCase()
    )
  }
}
