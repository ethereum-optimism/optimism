export const makeHexString = (byte: string, len: number): string => {
  return '0x' + byte.repeat(len)
}

export const makeAddress = (byte: string): string => {
  return makeHexString(byte, 20)
}

export const remove0x = (str: string): string => {
  if (str.startsWith('0x')) {
    return str.slice(2)
  } else {
    return str
  }
}
