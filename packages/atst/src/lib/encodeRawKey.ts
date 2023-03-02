import { ethers } from 'ethers'

import { WagmiBytes } from '../types/WagmiBytes'

export const encodeRawKey = (rawKey: string): WagmiBytes => {
  if (rawKey.length < 32) {
    return ethers.utils.formatBytes32String(rawKey) as WagmiBytes
  }
  const hash = ethers.utils.keccak256(ethers.utils.toUtf8Bytes(rawKey))
  return (hash.slice(0, 64) + 'ff') as WagmiBytes
}
