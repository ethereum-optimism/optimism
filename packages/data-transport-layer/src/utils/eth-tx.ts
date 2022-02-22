/* Imports: External */
import { ethers } from 'ethers'
import { sequencerBatch } from '@eth-optimism/core-utils'

export const parseSignatureVParam = (
  v: number | ethers.BigNumber | string,
  chainId: number
): number => {
  v = ethers.BigNumber.from(v).toNumber()
  // Handle unprotected transactions
  if (v === 27 || v === 28) {
    return v
  }
  // Handle EIP155 transactions
  return v - 2 * chainId - 35
}

// The indices take into account the 4 byte selector
export const compressBatchWithZlib = (calldata: string | Buffer): string => {
  const batch = sequencerBatch.decode(calldata.toString('hex'))
  const encoded = sequencerBatch.encode(batch, { zlib: true })
  return encoded
}
