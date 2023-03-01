import { ethers } from 'ethers'

export const encodeRawKey = (rawKey: string) => {
  if (rawKey.length < 32) {
    return ethers.utils.formatBytes32String(rawKey)
  }
  const hash = ethers.utils.keccak256(ethers.utils.toUtf8Bytes(rawKey))
  return hash.slice(0, 64) + 'ff'
}
