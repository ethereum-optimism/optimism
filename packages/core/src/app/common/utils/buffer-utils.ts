// Promisify the it.next(cb) function
export const itNext = (it): Promise<{ key: Buffer; value: Buffer }> => {
  return new Promise((resolve, reject) => {
    it.next((err, key, value) => {
      if (err) {
        reject(err)
      }
      resolve({ key, value })
    })
  })
}

// Promisify the it.end(cb) function
export const itEnd = (it) => {
  return new Promise((resolve, reject) => {
    it.end((err) => {
      if (err) {
        reject(err)
      }
      resolve()
    })
  })
}

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
 * A collection of useful utilities for comparing buffers.
 */
export const bufferUtils = {
  lt,
  lte,
  gt,
  gte,
  max,
  min,
}
