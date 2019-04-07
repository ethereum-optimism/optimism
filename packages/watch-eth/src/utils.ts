import crypto = require('crypto')

/**
 * Creates a simple (not cryptographically secure) hash.
 * @param message Message to be hashed.
 * @returns the hashed value.
 */
export const hash = (message: string): string => {
  return crypto
    .createHash('md5')
    .update(message)
    .digest('hex')
}

/**
 * Creates a promise that resolves after a certain period of time.
 * @param ms Number of milliseconds to sleep.
 * @returns a promise that resolves later.
 */
export const sleep = (ms: number): Promise<void> => {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}
