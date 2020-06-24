/* External Imports */
import { ethers } from 'ethers'
export const abi = new ethers.utils.AbiCoder()

export const transactionAbiTypes = [
  'bytes', // ovmEntrypoint Address
  'bytes', // ovmCalldata string
]

export const logAbiTypes = [
  'bytes32', // string data
  'bytes32[]', // string[] topics
  'uint', // logIndex
  'uint', // transactionIndex
  'bytes32', // transactionHash
  'bytes32', // blockHash
  'uint', // blockNumber
  'address', // address
]

export const transactionReceiptAbiTypes = [
  'bool', // status
  'bytes32', // transactionHash
  'uint', // transactionIndex
  'bytes32', // blockHash
  'uint', // blockNumber
  'address', // contractAddress
  'uint', // cumulativeGasUsed
  'uint', // gasUsed
  'bytes[]', // TransactionLog[]
]
