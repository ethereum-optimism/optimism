/* External Imports */
import { BigNumber } from 'ethers'

export const getLen = (pos: { start; end }) => (pos.end - pos.start) * 2

export const encodeHex = (val: any, len: number) =>
  remove0x(BigNumber.from(val).toHexString()).padStart(len, '0')

export const toVerifiedBytes = (val: string, len: number) => {
  val = remove0x(val)
  if (val.length !== len) {
    throw new Error('Invalid length!')
  }
  return val
}

export const remove0x = (str: string): string => {
  if (str.startsWith('0x')) {
    return str.slice(2)
  } else {
    return str
  }
}
