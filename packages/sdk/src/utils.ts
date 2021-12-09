import assert from 'assert'
import {
  Provider,
  TransactionReceipt,
  TransactionResponse,
} from '@ethersproject/abstract-provider'
import { getContractInterface } from '@eth-optimism/contracts'
import { ethers } from 'ethers'
import {
  ProviderLike,
  TransactionLike,
  DirectionlessCrossChainMessage,
} from './interfaces'

/**
 * Returns the canonical encoding of a cross chain message. This encoding is used in various
 * locations within the Optimistic Ethereum smart contracts.
 *
 * @param message Cross chain message to encode.
 * @returns Canonical encoding of the message.
 */
export const encodeCrossChainMessage = (
  message: DirectionlessCrossChainMessage
): string => {
  return getContractInterface('L2CrossDomainMessenger').encodeFunctionData(
    'relayMessage',
    [message.target, message.sender, message.message, message.messageNonce]
  )
}

/**
 * Returns the canonical hash of a cross chain message. This hash is used in various locations
 * within the Optimistic Ethereum smart contracts and is the keccak256 hash of the result of
 * encodeCrossChainMessage.
 *
 * @param message Cross chain message to hash.
 * @returns Canonical hash of the message.
 */
export const hashCrossChainMessage = (
  message: DirectionlessCrossChainMessage
): string => {
  return ethers.utils.solidityKeccak256(
    ['bytes'],
    [encodeCrossChainMessage(message)]
  )
}

/**
 * Converts a ProviderLike into a provider. Assumes that if the ProviderLike is a string then
 * it is a JSON-RPC url.
 *
 * @param provider ProviderLike to turn into a provider.
 * @returns ProviderLike as a provider.
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
