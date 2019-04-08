/* External Imports */
import BigNum = require('bn.js')

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
 * Determines the less of two BigNums.
 * @param a First BigNum.
 * @param b Second BigNum.
 * @returns the lesser of the two.
 */
export const bnMin = (a: BigNum, b: BigNum) => {
  return a.lt(b) ? a : b
}

/**
 * Determines the greater of two BigNums.
 * @param a First BigNum.
 * @param b Second BigNum.
 * @returns the greater of the two.
 */
export const bnMax = (a: BigNum, b: BigNum) => {
  return a.gt(b) ? a : b
}
