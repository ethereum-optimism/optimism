/* Imports: External */
import { ethers } from 'ethers'

export const parseSignatureVParam = (
  v: number | ethers.BigNumber | string,
  chainId: number
): number => {
  v = ethers.BigNumber.from(v).toNumber()
  // Handle normalized v
  if (v === 0 || v === 1) {
    return v
  }
  // Handle unprotected transactions
  if (v === 27 || v === 28) {
    return v - 27
  }
  // Handle EIP155 transactions
  return v - 2 * chainId - 35
}
