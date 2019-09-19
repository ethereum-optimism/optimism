/* External Imports */
import { ethers } from 'ethers'
export const abi = new ethers.utils.AbiCoder()

export const transferAbiTypes = [
  'address', // sender address
  'address', // recipient address
  'bool', // token type
  'uint32', // amount
]
export const swapAbiTypes = [
  'address', // sender address
  'bool', // token type
  'uint32', // amount
  'uint32', // min output amount
  'uint', // timeout
]
export const signedTransactionAbiTypes = [
  'bytes', // signature
  'bytes', // transaction
]
export const swapTransitionAbiTypes = [
  'bytes32', // state root
  'uint32', // sender slot
  'uint32', // uniswap slot
  'bool', // token type
  'uint32', // input amount
  'uint32', // min output amount
  'uint', // timeout
  'bytes', // transaction signature
]
export const transferTransitionAbiTypes = [
  'bytes32', // state root
  'uint32', // sender slot
  'uint32', // recipient slot
  'bool', // token type
  'uint32', // amount
  'bytes', // transaction signature
]
export const createAndTransferTransitionAbiTypes = [
  'bytes32', // state root
  'uint32', // sender slot
  'uint32', // recipient slot
  'address', // created public key
  'bool', // token type
  'uint32', // amount
  'bytes', // transaction signature
]
export const stateAbiTypes = [
  'address', // owner address
  'uint32', // uni balance
  'uint32', // pigi balance
]
export const stateReceiptAbiTypes = [
  'bytes32', // state root
  'uint', // block number
  'uint', // transition index
  'uint32', // state slot
  'bytes32[]', // inclusion proof
  'bytes', // state
]
