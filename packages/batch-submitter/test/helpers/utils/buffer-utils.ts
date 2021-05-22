/* External Imports */
import { BigNumber } from 'ethers'

/**
 * Converts a string or buffer to a '0x'-prefixed hex string.
 * @param buf String or buffer to convert.
 * @returns '0x'-prefixed string.
 */
export const toHexString = (buf: Buffer | string): string => {
  return '0x' + fromHexString(buf).toString('hex')
}

/**
 * Converts a '0x'-prefixed string to a buffer.
 * @param str '0x'-prefixed string to convert.
 * @returns Hex buffer.
 */
export const fromHexString = (str: string | Buffer): Buffer => {
  if (typeof str === 'string' && str.startsWith('0x')) {
    return Buffer.from(str.slice(2), 'hex')
  }

  return Buffer.from(str)
}

export const toHexString32 = (
  input: Buffer | string | number,
  padRight = false
): string => {
  if (typeof input === 'number') {
    input = BigNumber.from(input).toHexString()
  }

  input = toHexString(input).slice(2)
  return '0x' + (padRight ? input.padEnd(64, '0') : input.padStart(64, '0'))
}

export const getHexSlice = (
  input: Buffer | string,
  start: number,
  length: number
): string => {
  return toHexString(fromHexString(input).slice(start, start + length))
}
