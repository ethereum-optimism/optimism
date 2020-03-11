import { bufToHexString, remove0x } from './misc'

/**
 * Checks if buf1 is less than or equal to buf2
 * @param buf1 the first Buffer
 * @param buf2 the second Buffer
 * @returns boolean result of evaluating buf1 <= buf2
 */
const lte = (buf1: Buffer, buf2: Buffer): boolean => {
  return Buffer.compare(buf1, buf2) <= 0
}

/**
 * Checks if buf1 is strictly less than buf2
 * @param buf1 the first Buffer
 * @param buf2 the second Buffer
 * @returns boolean result of evaluating buf1 < buf2
 */
const lt = (buf1: Buffer, buf2: Buffer): boolean => {
  return Buffer.compare(buf1, buf2) < 0
}

/**
 * Checks if buf1 is strictly greater than buf2
 * @param buf1 the first Buffer
 * @param buf2 the second Buffer
 * @returns boolean result of evaluating buf1 > buf2
 */
const gt = (buf1: Buffer, buf2: Buffer): boolean => {
  return Buffer.compare(buf1, buf2) > 0
}

/**
 * Checks if buf1 is greater than or equal to buf2
 * @param buf1 the first Buffer
 * @param buf2 the second Buffer
 * @returns boolean result of evaluating buf1 >= buf2
 */
const gte = (buf1: Buffer, buf2: Buffer): boolean => {
  return Buffer.compare(buf1, buf2) >= 0
}

/**
 * Compare two buffers, returning the maximum of the two.
 * @param buf1 the first Buffer
 * @param buf2 the second Buffer
 * @returns Buffer the larger Buffer
 */
const max = (buf1: Buffer, buf2: Buffer): Buffer => {
  return gte(buf1, buf2) ? buf1 : buf2
}

/**
 * Compare two buffers, returning the minimum of the two.
 * @param buf1 the first Buffer
 * @param buf2 the second Buffer
 * @returns Buffer the smaller Buffer
 */
const min = (buf1: Buffer, buf2: Buffer): Buffer => {
  return lte(buf1, buf2) ? buf1 : buf2
}

/**
 * Pad Buffer with zeros to the left
 * @param buf the Buffer we want to pad
 * @param totalWidth the total number of bytes the Buffer should be after being padded
 * @returns Buffer the original buffer padded with zeros
 */
const padLeft = (buf: Buffer, totalWidth: number): Buffer => {
  if (buf.length > totalWidth) {
    throw new Error('Attempting to pad a buffer which is too large')
  }
  const newBuf = Buffer.alloc(totalWidth)
  newBuf.fill(buf, totalWidth - buf.length, totalWidth)
  return newBuf
}

/**
 * Pad Buffer with zeros to the right
 * @param buf the Buffer we want to pad
 * @param totalWidth the total number of bytes the Buffer should be after being padded
 * @returns Buffer the original buffer padded with zeros
 */
const padRight = (buf: Buffer, totalWidth: number): Buffer => {
  if (buf.length > totalWidth) {
    throw new Error('Attempting to pad a buffer which is too large')
  }
  const newBuf = Buffer.alloc(totalWidth)
  newBuf.fill(buf, 0, buf.length)
  return newBuf
}

/**
 * Converts the provided number to a Buffer and returns it.
 * @param num The number to convert.
 * @param numBytes The number of bytes in the number.
 * @param bufferBytes The number of bytes in the output Buffer.
 * @param bigEndian The endianness of the output buffer.
 * @returns The buffer.
 */
const numberToBuffer = (
  num: number,
  numBytes: number = 4,
  bufferBytes: number = 32,
  bigEndian: boolean = true
): Buffer => {
  const minBytes = Math.max(bufferBytes, numBytes)
  const buf: Buffer = Buffer.alloc(minBytes)
  if (bigEndian) {
    buf.writeIntBE(num, minBytes - numBytes, numBytes)
  } else {
    buf.writeIntLE(num, 0, numBytes)
  }
  return buf
}

/**
 * Converts a number to a packed BigEndian buffer.
 * @param num The number in question
 * @param minLength The minimum number of bytes to return
 * @returns The packed buffer
 */
const numberToBufferPacked = (num: number, minLength: number = 1): Buffer => {
  const buf: Buffer = Buffer.alloc(4)
  buf.writeInt32BE(num, 0)
  return removeEmptyBytes(buf, minLength)
}

/**
 * Removes the empty bytes at the beginning of a big-endian buffer.
 * @param buf The buffer in question.
 * @param minLength The minimum number of bytes to return
 * @returns The trimmed buffer with the non-empty bytes.
 */
const removeEmptyBytes = (buf: Buffer, minLength: number): Buffer => {
  let firstNonZeroIndex = 0
  while (firstNonZeroIndex < buf.length && buf[firstNonZeroIndex] === 0) {
    firstNonZeroIndex++
  }
  const startIndex = Math.min(firstNonZeroIndex, buf.length - 1)
  const index =
    buf.length - startIndex < minLength ? buf.length - minLength : startIndex
  return index < 0
    ? Buffer.concat([Buffer.from('00'.repeat(0 - index), 'hex'), buf])
    : buf.slice(index)
}

/**
 * Returns whether or not the numbers represented by the provided buffers are equal.
 * @param first
 * @param second
 */
const numbersEqual = (first: Buffer, second: Buffer): boolean => {
  const firstString = remove0x(bufToHexString(first))
  const secondString = remove0x(bufToHexString(second))

  const getIndexOfFirstNonZero = (str: string): number => {
    let i = 0
    for (; i < firstString.length; i++) {
      if (firstString.charAt(i) !== '0') {
        break
      }
    }
    return i
  }

  return (
    firstString.substr(getIndexOfFirstNonZero(firstString)) ===
    secondString.substr(getIndexOfFirstNonZero(secondString))
  )
}

const bufferToAddress = (addressAsBuffer: Buffer): string => {
  return bufToHexString(padLeft(addressAsBuffer, 20))
}

/**
 * A collection of useful utilities for comparing buffers.
 */
export const bufferUtils = {
  lt,
  lte,
  gt,
  gte,
  max,
  min,
  padLeft,
  padRight,
  numberToBuffer,
  numberToBufferPacked,
  numbersEqual,
  bufferToAddress,
}
