/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { zeroPad } from '@ethersproject/bytes'
import { Wallet } from 'ethers'
import {
  remove0x,
  numberToHexString,
  hexStrToBuf,
  makeAddressManager,
} from '../'
import { ZERO_ADDRESS } from '../constants'

export interface EIP155Transaction {
  nonce: number
  gasLimit: number
  gasPrice: number
  to: string
  data: string
  chainId: number
}

export interface SignatureParameters {
  messageHash: string
  v: string
  r: string
  s: string
}

export const DEFAULT_EIP155_TX: EIP155Transaction = {
  to: `0x${'12'.repeat(20)}`,
  nonce: 100,
  gasLimit: 1000000,
  gasPrice: 100000000,
  data: `0x${'99'.repeat(10)}`,
  chainId: 420,
}

export const getRawSignedComponents = (signed: string): any[] => {
  return [signed.slice(130, 132), signed.slice(2, 66), signed.slice(66, 130)]
}

export const getSignedComponents = (signed: string): any[] => {
  return ethers.utils.RLP.decode(signed).slice(-3)
}

export const encodeCompactTransaction = (transaction: any): string => {
  const nonce = zeroPad(transaction.nonce, 3)
  const gasLimit = zeroPad(transaction.gasLimit, 3)
  if (transaction.gasPrice % 1000000 !== 0)
    throw Error('gas price must be a multiple of 1000000')
  const compressedGasPrice: any = transaction.gasPrice / 1000000
  const gasPrice = zeroPad(compressedGasPrice, 3)
  const to = !transaction.to.length
    ? hexStrToBuf(ZERO_ADDRESS)
    : hexStrToBuf(transaction.to)
  const data = hexStrToBuf(transaction.data)

  return Buffer.concat([
    Buffer.from(gasLimit),
    Buffer.from(gasPrice),
    Buffer.from(nonce),
    Buffer.from(to),
    data,
  ]).toString('hex')
}

export const serializeEthSignTransaction = (
  transaction: EIP155Transaction
): string => {
  return ethers.utils.defaultAbiCoder.encode(
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

export const serializeNativeTransaction = (
  transaction: EIP155Transaction
): string => {
  return ethers.utils.serializeTransaction(transaction)
}

export const signEthSignMessage = async (
  wallet: Wallet,
  transaction: EIP155Transaction
): Promise<SignatureParameters> => {
  const serializedTransaction = serializeEthSignTransaction(transaction)
  const transactionHash = ethers.utils.keccak256(serializedTransaction)
  const transactionHashBytes = ethers.utils.arrayify(transactionHash)
  const transactionSignature = await wallet.signMessage(transactionHashBytes)

  const messageHash = ethers.utils.hashMessage(transactionHashBytes)
  let [v, r, s] = getRawSignedComponents(transactionSignature).map(
    (component) => {
      return remove0x(component)
    }
  )
  v = '0' + (parseInt(v, 16) - 27)
  return {
    messageHash,
    v,
    r,
    s,
  }
}

export const signNativeTransaction = async (
  wallet: Wallet,
  transaction: EIP155Transaction
): Promise<SignatureParameters> => {
  const serializedTransaction = serializeNativeTransaction(transaction)
  const transactionSignature = await wallet.signTransaction(transaction)

  const messageHash = ethers.utils.keccak256(serializedTransaction)
  let [v, r, s] = getSignedComponents(transactionSignature).map((component) => {
    return remove0x(component)
  })
  v = '0' + (parseInt(v, 16) - 420 * 2 - 8 - 27)
  return {
    messageHash,
    v,
    r,
    s,
  }
}

export const signTransaction = async (
  wallet: Wallet,
  transaction: EIP155Transaction,
  transactionType: number
): Promise<SignatureParameters> => {
  return transactionType === 2
    ? signEthSignMessage(wallet, transaction) //ETH Signed tx
    : signNativeTransaction(wallet, transaction) //Create EOA tx or EIP155 tx
}

export const encodeSequencerCalldata = async (
  wallet: Wallet,
  transaction: EIP155Transaction,
  transactionType: number
) => {
  const sig = await signTransaction(wallet, transaction, transactionType)
  const encodedTransaction = encodeCompactTransaction(transaction)
  const dataPrefix = `0x0${transactionType}${sig.r}${sig.s}${sig.v}`
  const calldata =
    transactionType === 1
      ? `${dataPrefix}${remove0x(sig.messageHash)}` // Create EOA tx
      : `${dataPrefix}${encodedTransaction}` // EIP155 tx or ETH Signed Tx
  return calldata
}
