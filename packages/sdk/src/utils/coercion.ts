import { ethers, Provider, Signer, TransactionReceipt, TransactionResponse } from 'ethers'
import { assert } from './assert'
import {
  SignerOrProviderLike,
  ProviderLike,
  TransactionLike,
  NumberLike,
  AddressLike,
} from '../interfaces'

/**
 * Converts a SignerOrProviderLike into a Signer or a Provider. Assumes that if the input is a
 * string then it is a JSON-RPC url.
 *
 * @param signerOrProvider SignerOrProviderLike to turn into a Signer or Provider.
 * @returns Input as a Signer or Provider.
 */
export const toSignerOrProvider = (
  signerOrProvider: SignerOrProviderLike
): Signer | Provider => {
  if (typeof signerOrProvider === 'string') {
    return new ethers.JsonRpcProvider(signerOrProvider)
  }
  return signerOrProvider
}

/**
 * Converts a ProviderLike into a Provider. Assumes that if the input is a string then it is a
 * JSON-RPC url.
 *
 * @param provider ProviderLike to turn into a Provider.
 * @returns Input as a Provider.
 */
export const toProvider = (provider: ProviderLike): Provider => {
  if (typeof provider === 'string') {
    return new ethers.JsonRpcProvider(provider)
  }
  return provider
}

/**
 * Pulls a transaction hash out of a TransactionLike object.
 *
 * @param transaction TransactionLike to convert into a transaction hash.
 * @returns Transaction hash corresponding to the TransactionLike input.
 */
export const toTransactionHash = (transaction: TransactionLike): string => {
  if (typeof transaction === 'string') {
    assert(
      ethers.isHexString(transaction, 32),
      'Invalid transaction hash'
    )
    return transaction
  } else if ((transaction as TransactionReceipt).hash) {
    return (transaction as TransactionReceipt).hash
  } else if ((transaction as TransactionResponse).hash) {
    return (transaction as TransactionResponse).hash
  } else {
    throw new Error('Invalid transaction')
  }
}

/**
 * Converts a number-like into an ethers BigNumber.
 *
 * @param num Number-like to convert into a BigNumber.
 * @returns Number-like as a BigNumber.
 */
export const toBigNumber = (num: NumberLike): BigInt => {
  return BigInt(Number(num))
}

/**
 * Converts a number-like into a number.
 *
 * @param num Number-like to convert into a number.
 * @returns Number-like as a number.
 */
export const toNumber = (num: NumberLike): number => {
  return Number(toBigNumber(num))
}

/**
 * Converts an address-like into a 0x-prefixed address string.
 *
 * @param addr Address-like to convert into an address.
 * @returns Address-like as an address.
 */
export const toAddress = async (addr: AddressLike): Promise<string> => {
  if (typeof addr === 'string') {
    assert(ethers.isAddress(addr), 'Invalid address')
    return ethers.getAddress(addr)
  } else {
    assert(ethers.isAddress(addr.address), 'Invalid address')
    return ethers.getAddress(await addr.getAddress())
  }
}
