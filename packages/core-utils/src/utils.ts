/* External Imports */
import { BigNumber, utils } from 'ethers'
import { AbiCoder } from '@ethersproject/abi'
import { keccak256 } from '@ethersproject/keccak256'

export const getLen = (pos: { start; end }) => (pos.end - pos.start) * 2

export const encodeHex = (val: any, len: number) =>
  remove0x(BigNumber.from(val).toHexString()).padStart(len, '0')

export const toVerifiedBytes = (val: string, len: number) => {
  val = remove0x(val)
  if (val.length !== len) {
    throw new Error('Invalid length!')
  }
  return val
}

export const remove0x = (str: string): string => {
  if (str.startsWith('0x')) {
    return str.slice(2)
  }
  return str
}

export const add0x = (str: string): string => {
  if (!str.startsWith('0x')) {
    return `0x${str}`
  }
  return str
}

export const serializeEthSignTransaction = (transaction): string => {
  const abi = new AbiCoder()
  return abi.encode(
    ['uint256', 'uint256', 'uint256', 'uint256', 'address', 'bytes'],
    [
      transaction.nonce,
      transaction.gasLimit,
      transaction.gasPrice,
      transaction.chainId,
      transaction.to,
      transaction.data,
    ]
  )
}

// Use this function to compute the ETH_SIGN signature hash. The prefix
// boolean set to true prepend `\x19Ethereum Signed Message:` before hashing.
// Set to false if using metamask
export const sighashEthSign = (transaction, prefix?: boolean): string => {
  const serialized = serializeEthSignTransaction(transaction)
  const hash = keccak256(utils.arrayify(serialized))
  if (prefix) {
    return utils.hashMessage(utils.arrayify(hash))
  }
  return hash
}
