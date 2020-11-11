const hexRegex = /^(0x)?[0-9a-fA-F]*$/

/**
 * Generates a hex string of repeated bytes.
 * @param byte Byte to repeat.
 * @param len Number of times to repeat the byte.
 * @return '0x'-prefixed hex string filled with the provided byte.
 */
export const makeHexString = (byte: string, len: number): string => {
  return '0x' + byte.repeat(len)
}

/**
 * Genereates an address with a repeated byte.
 * @param byte Byte to repeat in the address.
 * @return Address filled with the repeated byte.
 */
export const makeAddress = (byte: string): string => {
  return makeHexString(byte, 20)
}

/**
 * Removes '0x' from a hex string.
 * @param str Hex string to remove '0x' from.
 * @returns String without the '0x' prefix.
 */
export const remove0x = (str: string): string => {
  if (str.startsWith('0x')) {
    return str.slice(2)
  } else {
    return str
  }
}

/**
 * Returns whether or not the provided string is a hex string.
 *
 * @param str The string to test.
 * @returns True if the provided string is a hex string, false otherwise.
 */
export const isHexString = (str: string): boolean => {
  return hexRegex.test(str)
}

/**
 * Converts a hex string to a buffer
 * @param hexString the hex string to be converted
 * @returns the hexString as a buffer.
 */
export const hexStrToBuf = (hexString: string): Buffer => {
  if (!isHexString(hexString)) {
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
 * Converts a JavaScript number to a big-endian hex string.
 * @param number the JavaScript number to be converted.
 * @param padToBytes the number of numeric bytes the resulting string should be, -1 if no padding should be done.
 * @returns the JavaScript number as a string.
 */
export const numberToHexString = (
  number: number,
  padToBytes: number = -1
): string => {
  let str = number.toString(16)
  if (padToBytes > 0 || str.length < padToBytes * 2) {
    str = `${'0'.repeat(padToBytes * 2 - str.length)}${str}`
  }
  return add0x(str)
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
