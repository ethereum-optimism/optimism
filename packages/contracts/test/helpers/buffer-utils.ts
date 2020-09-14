export const toHexString = (buf: Buffer | string): string => {
  return '0x' + fromHexString(buf).toString('hex')
}

export const fromHexString = (str: string | Buffer): Buffer => {
  if (typeof str === 'string' && str.startsWith('0x')) {
    return Buffer.from(str.slice(2), 'hex')
  }

  return Buffer.from(str)
}
