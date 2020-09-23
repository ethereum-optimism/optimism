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
