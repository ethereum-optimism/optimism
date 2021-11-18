/* Imports: External */
import { ethers } from 'ethers'

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
