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
