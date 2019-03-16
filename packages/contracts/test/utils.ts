import BigNum from 'bn.js'

/* tslint:disable:no-bitwise */
/**
 * Converts a string to a UTF8 byte array.
 * Modified from: https://github.com/google/closure-library/blob/e877b1eac410c0d842bcda118689759512e0e26f/closure/goog/crypt/crypt.js
 * @param str String to convert.
 * @returns the string as a UTF8 byte array.
 */
export const stringToUtf8ByteArray = (str: string): number[] => {
  const out = []
  let p = 0
  for (let i = 0; i < str.length; i++) {
    let c = str.charCodeAt(i)
    if (c < 128) {
      out[p++] = c
    } else if (c < 2048) {
      out[p++] = (c >> 6) | 192
      out[p++] = (c & 63) | 128
    } else if (
      (c & 0xfc00) === 0xd800 &&
      i + 1 < str.length &&
      (str.charCodeAt(i + 1) & 0xfc00) === 0xdc00
    ) {
      // Surrogate Pair
      c = 0x10000 + ((c & 0x03ff) << 10) + (str.charCodeAt(++i) & 0x03ff)
      out[p++] = (c >> 18) | 240
      out[p++] = ((c >> 12) & 63) | 128
      out[p++] = ((c >> 6) & 63) | 128
      out[p++] = (c & 63) | 128
    } else {
      out[p++] = (c >> 12) | 224
      out[p++] = ((c >> 6) & 63) | 128
      out[p++] = (c & 63) | 128
    }
  }
  return out
}
/* tslint:enable:no-bitwise */

/**
 * Converts a string to a Bytes32 value by padding it
 * to the right.
 * @param str String to convert.
 * @returns the padded string.
 */
export const stringToBytes32 = (str: string): string => {
  const utf8 = stringToUtf8ByteArray(str)
  const padding = new Array(32 - utf8.length).fill(0)
  return '0x' + new BigNum(utf8.concat(padding)).toString('hex', 64)
}
