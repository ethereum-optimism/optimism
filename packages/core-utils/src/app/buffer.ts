import { Address } from 'cluster'
import { BigNumber } from 'ethers/utils'
import { AddressZero } from 'ethers/constants'
import { bufToHexString } from '../app/misc'

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
  const buf: Buffer = Buffer.alloc(bufferBytes)
  if (bigEndian) {
    buf.writeIntBE(num, bufferBytes - numBytes, numBytes)
  } else {
    buf.writeIntLE(num, 0, numBytes)
  }
  return buf
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
  bufferToAddress,
}
