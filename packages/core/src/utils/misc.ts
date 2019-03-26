/* External Imports */
import BigNum = require('bn.js')

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

interface PrettyPrintable {
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
