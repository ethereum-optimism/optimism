/* External Imports */
import { ethers } from 'ethers'
export const abi = new ethers.utils.AbiCoder()

export const transferAbiTypes = [
  'address', // sender slot index address
  'address', // receiver slot index address
  'uint32', // token type
  'uint32', // amount
]
export const signedTransactionAbiTypes = [
  'bytes', // signature
  'bytes', // transaction
]
