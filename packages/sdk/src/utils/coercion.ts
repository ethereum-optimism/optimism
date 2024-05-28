import {
  Provider,
  TransactionReceipt,
  TransactionResponse,
} from '@ethersproject/abstract-provider'
import { Signer } from '@ethersproject/abstract-signer'
import { ethers, BigNumber } from 'ethers'

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
    return new ethers.providers.JsonRpcProvider(signerOrProvider)
  } else if (Provider.isProvider(signerOrProvider)) {
    return signerOrProvider
  } else if (Signer.isSigner(signerOrProvider)) {
    return signerOrProvider
  } else {
    throw new Error('Invalid provider')
  }
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
    return new ethers.providers.JsonRpcProvider(provider)
  } else if (Provider.isProvider(provider)) {
    return provider
  } else {
    throw new Error('Invalid provider')
  }
}

/**
 * Converts a ProviderLike into a JsonRpcProvider.
 *
 * @param provider ProviderLike to turn into a JsonRpcProvider.
 * @returns Input as a JsonRpcProvider.
 */
export const toJsonRpcProvider = (
  provider: ProviderLike
): ethers.providers.JsonRpcProvider => {
  const coerced = toProvider(provider)
  if ('send' in coerced) {
    // Existence of "send" is basically the only function that matters for determining if we can
    // use this provider as a JsonRpcProvider, because "send" is the function that we usually want
    // access to when we specifically care about having a JsonRpcProvider.
    return coerced as ethers.providers.JsonRpcProvider
  } else {
    throw new Error('Invalid JsonRpcProvider, does not have "send" function')
  }
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
      ethers.utils.isHexString(transaction, 32),
      'Invalid transaction hash'
    )
    return transaction
  } else if ((transaction as TransactionReceipt).transactionHash) {
    return (transaction as TransactionReceipt).transactionHash
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
export const toBigNumber = (num: NumberLike): BigNumber => {
  return ethers.BigNumber.from(num)
}

/**
 * Converts a number-like into a number.
 *
 * @param num Number-like to convert into a number.
 * @returns Number-like as a number.
 */
export const toNumber = (num: NumberLike): number => {
  return toBigNumber(num).toNumber()
}

/**
 * Converts an address-like into a 0x-prefixed address string.
 *
 * @param addr Address-like to convert into an address.
 * @returns Address-like as an address.
 */
export const toAddress = (addr: AddressLike): string => {
  if (typeof addr === 'string') {
    assert(ethers.utils.isAddress(addr), 'Invalid address')
    return ethers.utils.getAddress(addr)
  } else {
    assert(ethers.utils.isAddress(addr.address), 'Invalid address')
    return ethers.utils.getAddress(addr.address)
  }
}
