/* External Imports */
import BigNum = require('bn.js')

/* Internal Imports */
import { Transaction } from '../../../interfaces'

/**
 * JSON-stringifies a value if it's not already a string.
 * @param value Value to stringify.
 * @returns the stringified value.
 */
export const stringify = (value: any): string => {
  if (!(typeof value === 'string')) {
    value = JSON.stringify(value)
  }
  return value as string
}

/**
 * JSON-parses a value if it's not already an object.
 * @param value Value to parse.
 * @returns the parsed value.
 */
export const jsonify = (value: any): {} => {
  return isJson(value) ? JSON.parse(value) : value
}

/**
 * Checks whether something is a JSON string.
 * @param value Value to check.
 * @returns `true` if it's a JSON string, `false` otherwise.
 */
export const isJson = (value: string): boolean => {
  try {
    JSON.parse(value)
  } catch (err) {
    return false
  }
  return true
}

/**
 * Determines the lesser of two BigNums.
 * @param a First BigNum.
 * @param b Second BigNum.
 * @returns the lesser of the two.
 */
export const bnMin = (a: BigNum, b: BigNum): BigNum => {
  return a.lt(b) ? a : b
}

/**
 * Determines the greater of two BigNums.
 * @param a First BigNum.
 * @param b Second BigNum.
 * @returns the greater of the two.
 */
export const bnMax = (a: BigNum, b: BigNum): BigNum => {
  return a.gt(b) ? a : b
}

export interface PrettyPrintable {
  [key: string]: string | number | BigNum | boolean | any
}

/**
 * Converts an object to a pretty JSON string.
 * @param obj Object to convert.
 * @returns the object as a pretty JSON string.
 */
export const prettify = (obj: PrettyPrintable): string => {
  const parsed: PrettyPrintable = {}
  for (const key of Object.keys(obj)) {
    const value = obj[key]
    parsed[key] = BigNum.isBN(value)
      ? `${value.toString(16)} (${value.toString(10)})`
      : value
  }
  return JSON.stringify(parsed, null, 2)
}

/**
 * Converts a BigNum to a Uint256 Buffer.
 * @param bn BigNum to convert.
 * @returns the Uint256 Buffer.
 */
export const bnToUint256 = (bn: BigNum): Buffer => {
  return bn.toBuffer('be', 32)
}

/**
 * Gets the end of a transaction range as a Uint256 Buffer.
 * @param transaction Transaction to query.
 * @returns the transacted range's end as a Uint256 Buffer.
 */
export const getTransactionRangeEnd = (transaction: Transaction): Buffer => {
  const end = transaction.range.end
  return bnToUint256(end)
}

/**
 * Sleeps for a number of milliseconds.
 * @param ms Number of ms to sleep.
 * @returns a promise that resolves after the number of ms.
 */
export const sleep = (ms: number): Promise<void> => {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}

/**
 * Removes "0x" from start of a string if it exists.
 * @param str String to modify.
 * @returns the string without "0x".
 */
export const remove0x = (str: string): string => {
  return str.startsWith('0x') ? str.slice(2) : str
}

/**
 * Adds "0x" to the start of a string if necessary.
 * @param str String to modify.
 * @returns the string with "0x".
 */
export const add0x = (str: string): string => {
  return str.startsWith('0x') ? str : '0x' + str
}

/**
 * Checks if something is an Object
 * @param obj Thing that might be an Object.
 * @returns `true` if the thing is a Object, `false` otherwise.
 */
export const isObject = (obj: any): boolean => {
  return typeof obj === 'object' && obj !== null
}

/**
 * Creates a hex string with a certain number of zeroes.
 * @param n Number of zeroes.
 * @returns the hex string.
 */
export const getNullString = (n: number): string => {
  return '0x' + '0'.repeat(n)
}

/**
 * Reverses a string in place.
 * @param str String to reverse.
 * @returns the reversed string.
 */
export const reverse = (str: string): string => {
  return Array.from(str)
    .reverse()
    .join('')
}

/**
 * Converts a buffer to a hex string.
 * @param buf the buffer to be converted.
 * @returns the buffer as a string.
 */
export const bufToHexString = (buf: Buffer): string => {
  return '0x' + buf.toString('hex')
}

/**
 * Converts a big number to a hex string.
 * @param bn the big number to be converted.
 * @returns the big number as a string.
 */
export const bnToHexString = (bn: BigNum): string => {
  return '0x' + bn.toString('hex')
}

/**
 * Converts either a big number or buffer to hex string
 * @param value the big number or buffer to be converted
 * @returns the value as a string.
 */
export const hexStringify = (value: BigNum | Buffer): string => {
  if (value instanceof BigNum) {
    return bnToHexString(value)
  } else {
    return bufToHexString(value)
  }
}

/**
 * Converts a hex string to a buffer
 * @param hexString the hex string to be converted
 * @returns the hexString as a buffer.
 */
export const hexStrToBuf = (hexString: string): Buffer => {
  return Buffer.from(hexString.slice(2), 'hex')
}


/**
 * Creates a new version of a list with all instances of a specific element
 * removed.
 * @param list List to remove elements from.
 * @param element Element to remove from the list.
 * @returns the list without the given element.
 */
export const except = <T>(list: T[], element: T): T[] => {
  return list.filter((item) => {
    return item !== element
  })
}
