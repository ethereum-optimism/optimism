/* Imports: External */
import { BigNumber } from '@ethersproject/bignumber'
import { isHexString, hexZeroPad } from '@ethersproject/bytes'

/**
 * Removes "0x" from start of a string if it exists.
 *
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
 *
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
 * Casts a hex string to a buffer.
 *
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
 *
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

/**
 * Casts a number to a hex string without zero padding.
 *
 * @param n Number to cast to a hex string.
 * @return Number cast as a hex string.
 */
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
    // BigNumber pads a single 0 to keep hex length even
    return num.replace(/^0x0/, '0x')
  }
}

/**
 * Zero pads a hex string if str.length !== 2 + length * 2. Pads to length * 2.
 *
 * @param str Hex string to pad
 * @param length Half the length of the desired padded hex string
 * @return Hex string with length of 2 + length * 2
 */
export const padHexString = (str: string, length: number): string => {
  if (str.length === 2 + length * 2) {
    return str
  } else {
    return '0x' + str.slice(2).padStart(length * 2, '0')
  }
}

/**
 * Casts an input to hex string without '0x' prefix with conditional padding.
 * Hex string will always start with a 0.
 *
 * @param val Input to cast to a hex string.
 * @param len Desired length to pad hex string. Ignored if less than hex string length.
 * @return Hex string with '0' prefix
 */
export const encodeHex = (val: any, len: number): string =>
  remove0x(BigNumber.from(val).toHexString()).padStart(len, '0')

/**
 * Case insensitive hex string equality check
 *
 * @param stringA Hex string A
 * @param stringB Hex string B
 * @throws {Error} Inputs must be valid hex strings
 * @return True if equal
 */
export const hexStringEquals = (stringA: string, stringB: string): boolean => {
  if (!isHexString(stringA)) {
    throw new Error(`input is not a hex string: ${stringA}`)
  }

  if (!isHexString(stringB)) {
    throw new Error(`input is not a hex string: ${stringB}`)
  }

  return stringA.toLowerCase() === stringB.toLowerCase()
}

/**
 * Casts a number to a 32-byte, zero padded hex string.
 *
 * @param value Number to cast to a hex string.
 * @return Number cast as a hex string.
 */
export const bytes32ify = (value: number | BigNumber): string => {
  return hexZeroPad(BigNumber.from(value).toHexString(), 32)
}
