/* tslint:disable:no-bitwise */

// TODO: Add bcoin acknowledgements for this code!

/**
 * Checks a length assertion and throws an error
 * if the assertion isn't `true`.
 * @param check Length assertion to check.
 */
export const assertLen = (check: boolean): void => {
  assert(check, new RangeError('Invalid length for database key.'))
}

/**
 * Checks a type assertion and throws an error
 * if the assertion isn't `true`.
 * @param check Type assertion to check.
 */
export const assertType = (check: boolean): void => {
  assert(check, new TypeError('Invalid type for database key.'))
}

/**
 * Checks an assertion and throws an error
 * if the assertion isn't `true`.
 * @param check Length assertion to check.
 * @param err Error to throw if the assertion fails.
 */
export const assert = (check: boolean, err?: Error): void => {
  err = err || new Error('Assertion error.')
  if (!check) {
    if (Error.captureStackTrace) {
      Error.captureStackTrace(err, assert)
    }
    throw err
  }
}

/**
 * Parses a value into a key ID.
 * @param id Value to parse.
 * @returns the key ID.
 */
export const makeID = (id: string | number): number => {
  let parsed: number
  if (typeof id === 'string') {
    assert(id.length === 1)
    parsed = id.charCodeAt(0)
  } else {
    parsed = id
  }

  assert((parsed & 0xff) === parsed)
  assert(parsed !== 0xff)

  return parsed
}

/**
 * Computes the size of a string.
 * @param value String to check.
 * @param encoding Encoding of the string.
 * @returns the size of the string.
 */
export const sizeString = (value: string, encoding: string): number => {
  return 1 + Buffer.byteLength(value, encoding)
}

/**
 * Reads a string from a buffer.
 * @param buf Buffer to read string from.
 * @param offset Byte offset to start reading.
 * @param encoding Encoding of the string.
 * @returns the read string.
 */
export const readString = (
  buf: Buffer,
  offset: number,
  encoding: string
): string => {
  assertLen(offset + 1 <= buf.length)
  assertLen(offset + 1 + buf[offset] <= buf.length)
  return buf.toString(encoding, offset + 1, offset + 1 + buf[offset])
}

/**
 * Writes a string to a buffer.
 * @param buf Buffer to write to.
 * @param value Value to write.
 * @param offset Offset to start writing at.
 * @param encoding String encoding to use.
 * @returns the size of the buffer.
 */
export const writeString = (
  buf: Buffer,
  value: string,
  offset: number,
  encoding: string
): number => {
  const size = Buffer.byteLength(value, encoding)

  assertType(size <= 255)
  assertLen(offset + 1 <= buf.length)

  buf[offset] = size

  if (size > 0) {
    assertType(buf.write(value, offset + 1, encoding) === size)
  }

  return 1 + size
}

/**
 * Gets the size of a buffer.
 * @param buf Buffer to check.
 * @returns the size of the buffer.
 */
export const sizeBuffer = (buf: Buffer): number => {
  return 1 + buf.length
}

/**
 * Reads a buffer from another buffer.
 * @param buf Buffer to read.
 * @param offset Offset to start reading from.
 * @returns the read buffer.
 */
export const readBuffer = (buf: Buffer, offset: number): Buffer => {
  assertLen(offset + 1 <= buf.length)
  assertLen(offset + 1 + buf[offset] <= buf.length)
  return buf.slice(offset + 1, offset + 1 + buf[offset])
}

/**
 * Writes a buffer to another buffer.
 * @param buf Buffer to write to.
 * @param value Buffer to write.
 * @param offset Offset to start writing from.
 * @returns the length of the new buffer.
 */
export const writeBuffer = (
  buf: Buffer,
  value: Buffer,
  offset: number
): number => {
  assertLen(value.length <= 255)
  assertLen(offset + 1 <= buf.length)
  buf[offset] = value.length
  assertLen(value.copy(buf, offset + 1) === value.length)
  return 1 + value.length
}

/**
 * Checks the hex size of an item.
 * @param value Item to check.
 * @returns the size of the item.
 */
export const sizeHex = (value: Buffer | string): number => {
  if (Buffer.isBuffer(value)) {
    return value.length
  }
  return value.length >>> 1
}

/**
 * Writes a hex value to a buffer.
 * @param buf Buffer to write to.
 * @param value Hex value to write.
 * @param offset Offset to start writing from.
 * @returns the size of the new buffer.
 */
export const writeHex = (
  buf: Buffer,
  value: Buffer | string,
  offset: number
): number => {
  if (Buffer.isBuffer(value)) {
    return value.copy(buf, offset)
  }
  return buf.write(value, offset, 'hex')
}

export const BUFFER_MIN = Buffer.alloc(0)
export const BUFFER_MAX = Buffer.alloc(255, 0xff)
