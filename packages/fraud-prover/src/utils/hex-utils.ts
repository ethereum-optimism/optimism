import { BigNumber } from 'ethers'

export const toUnpaddedHexString = (buf: Buffer | string | number): string => {
  // prettier-ignore
  const hex =
    '0x' +
    toHexString(buf)
      .slice(2)
      .replace(/^0+/, '')

  if (hex === '0x') {
    return '0x0'
  } else {
    return hex
  }
}

export const toStrippedHexString = (buf: Buffer | string | number): string => {
  const hex = toUnpaddedHexString(buf).slice(2)

  if (hex === '0') {
    return '0x'
  } else if (hex.length % 2 === 1) {
    return '0x' + '0' + hex
  } else {
    return '0x' + hex
  }
}

export const toBytes32 = (buf: Buffer | string | number): string => {
  return toBytesN(buf, 32)
}

export const toBytesN = (buf: Buffer | string | number, n: number): string => {
  return (
    '0x' +
    toHexString(buf)
      .slice(2)
      .padStart(n * 2, '0')
  )
}

export const toUint256 = (num: number): string => {
  return toUintN(num, 32)
}

export const toUint8 = (num: number): string => {
  return toUintN(num, 1)
}

export const toUintN = (num: number, n: number): string => {
  return (
    '0x' +
    BigNumber.from(num)
      .toHexString()
      .slice(2)
      .padStart(n * 2, '0')
  )
}

export const fromHexString = (buf: Buffer | string): Buffer => {
  if (typeof buf === 'string' && buf.startsWith('0x')) {
    return Buffer.from(buf.slice(2), 'hex')
  }

  return Buffer.from(buf)
}

export const toHexString = (buf: Buffer | string | number | null): string => {
  if (typeof buf === 'number') {
    return BigNumber.from(buf).toHexString()
  } else {
    return '0x' + fromHexString(buf).toString('hex')
  }
}
