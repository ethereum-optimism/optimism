import { ethers } from 'ethers'

import { WagmiBytes } from '../types/WagmiBytes'

/**
 * Creates an attesation key from a raw string
 * Converts to bytes32 if key is less than 32 bytes
 * Hashes key if key is greater than 32 bytes
 */
export const createKey = (rawKey: string): WagmiBytes => {
  if (rawKey.length < 32) {
    return ethers.utils.formatBytes32String(rawKey) as WagmiBytes
  }
  const hash = ethers.utils.keccak256(ethers.utils.toUtf8Bytes(rawKey))
  return (hash.slice(0, 64) + 'ff') as WagmiBytes
}

/**
 * @deprecated use createKey instead
 * Will be removed in v1.0.0
 */
export const encodeRawKey: typeof createKey = (rawKey) => {
  console.warn('encodeRawKey is deprecated, use createKey instead')
  return createKey(rawKey)
}
