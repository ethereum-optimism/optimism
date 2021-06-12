/* Imports: External */
import { ethers } from 'ethers'

export const parseSignatureVParam = (
  v: number | ethers.BigNumber,
  chainId: number
): number => {
  return ethers.BigNumber.from(v).toNumber() - 2 * chainId - 35
}
