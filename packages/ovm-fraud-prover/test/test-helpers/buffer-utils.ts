/**
 * Utility; converts a buffer or string into a '0x'-prefixed string.
 * @param buf Element to convert.
 * @returns Converted element.
 */
export const toHexString = (buf: Buffer | string | null): string => {
  return '0x' + toHexBuffer(buf).toString('hex')
}

export const toHexBuffer = (buf: Buffer | string): Buffer => {
  if (typeof buf === 'string' && buf.startsWith('0x')) {
    return Buffer.from(buf.slice(2), 'hex')
  }

  return Buffer.from(buf)
}