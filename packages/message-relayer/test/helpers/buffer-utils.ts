export const toHexAddress = (buf: any): string => {
  return '0x' + buf.toString('hex').padStart(40, '0')
}

export const toHexString = (buf: any): string => {
  return '0x' + buf.toString('hex')
}

export const fromHexString = (str: string): Buffer => {
  return Buffer.from(str.slice(2), 'hex')
}

export const toAddressBuf = (address: string | Buffer): Buffer => {
  return typeof address === 'string' ? fromHexString(address) : address
}
