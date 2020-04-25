/* External Imports */
import * as domain from 'domain'

/* Internal Imports */
import { BigNumber } from './number'
import { RLP, hexlify } from 'ethers/utils'

export const NULL_ADDRESS = '0x0000000000000000000000000000000000000000'

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

export interface PrettyPrintable {
  [key: string]: string | number | BigNumber | boolean | any
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
    parsed[key] = BigNumber.isBigNumber(value)
      ? `${value.toString(16)} (${value.toString(10)})`
      : value
  }
  return JSON.stringify(parsed, null, 2)
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
 * Pads the provided string left with the provided string until it is the provided length.
 * Note: This function accounts for hex strings padding inside of the 0x prefix.
 *
 * @param str The string to pad.
 * @param length The desired resulting string length (excluding 0x prefix).
 * @param padString The string to pad with.
 * @returns The padded string.
 */
export const padToLength = (
  str: string,
  length: number,
  padString: string = '0'
): string => {
  const base: string = remove0x(str)
  const repeat: number =
    (length < base.length ? 0 : length - base.length) / padString.length
  const padded = padString.repeat(repeat) + base
  return base === str ? padded : add0x(padded)
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
 * Converts a big number to a hex string.
 * @param bn the big number to be converted.
 * @returns the big number as a string.
 */
export const bnToHexString = (bn: BigNumber): string => {
  return '0x' + bn.toString('hex')
}

/**
 * Converts a JavaScript number to a hex string.
 * @param number the JavaScript number to be converted.
 * @returns the JavaScript number as a string.
 */
export const numberToHexString = (bn: number): string => {
  return add0x(bn.toString(16))
}

/**
 * Converts either a big number or buffer to hex string
 * @param value the big number or buffer to be converted
 * @returns the value as a string.
 */
export const hexStringify = (value: BigNumber | Buffer): string => {
  if (value instanceof BigNumber) {
    return bnToHexString(value)
  } else if (value instanceof Buffer) {
    return bufToHexString(value)
  } else {
    throw new Error("Can't hexStringify--invalid type passed")
  }
}

/**
 * Converts a hex string to a buffer
 * @param hexString the hex string to be converted
 * @returns the hexString as a buffer.
 */
export const hexStrToBuf = (hexString: string): Buffer => {
  if (!/^(0x)?[0-9a-fA-F]*$/.test(hexString)) {
    throw new RangeError(`Invalid hex string [${hexString}]`)
  }
  if (hexString.length % 2 !== 0) {
    throw new RangeError(
      `Invalid hex string -- odd number of characters: [${hexString}]`
    )
  }
  return Buffer.from(remove0x(hexString), 'hex')
}

/**
 * Converts a hex string to a JavaScript Number
 * @param hexString the hex string to be converted
 * @returns the hexString as a JavaScript Number.
 */
export const hexStrToNumber = (hexString: string): number => {
  return parseInt(remove0x(hexString), 16)
}

/**
 * Parses a hex string if one is given otherwise returns the number that was
 * given
 * @param str the String to parse
 * @returns the parsed number.
 */
export const castToNumber = (stringOrNumber: string | number): number => {
  return typeof stringOrNumber === 'number'
    ? stringOrNumber
    : hexStrToNumber(stringOrNumber)
}

/**
 * Converts the provided buffer into a hex string.
 * @param buff The buffer.
 * @param prepend0x Whether or not to prepend '0x' to the resulting string.
 * @returns the hex string.
 */
export const bufToHexString = (
  buff: Buffer,
  prepend0x: boolean = true
): string => {
  const bufStr: string = buff.toString('hex')
  const str: string = bufStr.length % 2 === 0 ? bufStr : `0${bufStr}`
  return prepend0x ? add0x(str) : str
}

/**
 * Converts the provided (UTF-8) string into a hex string.
 * @param str The UTF-8 string.
 * @returns The hex string representation.
 */
export const strToHexStr = (str: string): string => {
  return bufToHexString(Buffer.from(str))
}

/**
 * Converts the provided hex string into the UTF-8 representation of it.
 * @param str The hex string.
 * @returns The UTF-8 representation of the string.
 */
export const hexStrToString = (str: string): string => {
  return hexStrToBuf(str).toString()
}

/**
 * Determine if a hex string is empty or undefined
 * @param str The hex string.
 * @returns boolean `true` if the string is empty or undefined, otherwise `false`
 */
export const isHexStringEmptyOrUndefined = (str: string): boolean => {
  return str === '0x' || str === undefined
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

/**
 * Runs the provided function in the provided domain if one is provided.
 * If a domain is falsy, this will create a domain for this function to run in.
 * @param d The domain in which this function should run.
 * @param func The function to run.
 * @returns The result of the function to be run
 */
export const runInDomain = async (
  d: domain.Domain,
  func: () => any
): Promise<any> => {
  const domainToUse: domain.Domain = !!d ? d : domain.create()
  return domainToUse.run(func)
}

/**
 * Gets the current number of seconds since the epoch.
 *
 * @returns The seconds since epoch.
 */
export const getCurrentTime = (): number => {
  return Math.round(Date.now() / 1000)
}
/**
 * Encodes a transaction in RLP format, using a random signature
 * @param {object} Transaction object
 */
export const rlpEncodeTransactionWithRandomSig = (
  transaction: object
): string => {
  return RLP.encode([
    hexlify(transaction['nonce']),
    hexlify(transaction['gasPrice']),
    hexlify(transaction['gasLimit']),
    hexlify(transaction['to']),
    hexlify(transaction['value']),
    hexlify(transaction['data']),
    '0x11', // v
    '0x' + '11'.repeat(32), // r
    '0x' + '11'.repeat(32), // s
  ])
}
