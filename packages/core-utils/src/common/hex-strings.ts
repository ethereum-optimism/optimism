/* Imports: External */
import { BigNumber } from 'ethers'

/**
 * Removes "0x" from start of a string if it exists.
 * @param str String to modify.
 * @returns the string without "0x".
 */
export const remove0x = (str: string): string => {
  if (str === undefined) {
    return str
  }
  return str.startsWith('0x') ? str.slice(2) : str
}

/**
 * Adds "0x" to the start of a string if necessary.
 * @param str String to modify.
 * @returns the string with "0x".
 */
export const add0x = (str: string): string => {
  if (str === undefined) {
    return str
  }
  return str.startsWith('0x') ? str : '0x' + str
}

/**
 * Returns whether or not the provided string is a hex string.
 * @param str The string to test.
 * @returns True if the provided string is a hex string, false otherwise.
 */
export const isHexString = (inp: any): boolean => {
  return typeof inp === 'string' && inp.startsWith('0x')
}

/**
 * Casts a hex string to a buffer.
 * @param inp Input to cast to a buffer.
 * @return Input cast as a buffer.
 */
export const fromHexString = (inp: Buffer | string): Buffer => {
  if (typeof inp === 'string' && inp.startsWith('0x')) {
    return Buffer.from(inp.slice(2), 'hex')
  }

  return Buffer.from(inp)
}

/**
 * Casts an input to a hex string.
 * @param inp Input to cast to a hex string.
 * @return Input cast as a hex string.
 */
export const toHexString = (inp: Buffer | string | number | null): string => {
  if (typeof inp === 'number') {
    return BigNumber.from(inp).toHexString()
  } else {
    return '0x' + fromHexString(inp).toString('hex')
  }
}

export const toRpcHexString = (n: number | BigNumber): string => {
  let num
  if (typeof n === 'number') {
    num = '0x' + n.toString(16)
  } else {
    num = n.toHexString()
  }

  if (num === '0x0') {
    return num
  } else {
    return num.replace(/^0x0/, '0x')
  }
}

export const padHexString = (str: string, length: number): string => {
  if (str.length === 2 + length * 2) {
    return str
  } else {
    return '0x' + str.slice(2).padStart(length * 2, '0')
  }
}

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

/**
 * @param byteLength The length of the hex string in bytes
 * @returns a random hex string of the specified byteLength (string length will be byteLength*2)
 */
export const getRandomHexString = (byteLength: number): string => {
  return (
    '0x' +
    [...Array(byteLength * 2)]
      .map(() => {
        return Math.floor(Math.random() * 16).toString(16)
      })
      .join('')
  )
}
