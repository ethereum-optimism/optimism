import {
  Provider,
  TransactionReceipt,
  TransactionResponse,
} from '@ethersproject/abstract-provider'
import { ethers } from 'ethers'
import { ProviderLike, TransactionLike, CrossChainMessage } from '../base'

/**
 * Returns the canonical encoding of a cross chain message. This encoding is used in various
 * locations within the Optimistic Ethereum smart contracts.
 *
 * @param message Cross chain message to encode.
 * @returns Canonical encoding of the message.
 */
export const encodeCrossChainMessage = (message: CrossChainMessage): string => {
  throw new Error('Not implemented')
}

/**
 * Returns the canonical hash of a cross chain message. This hash is used in various locations
 * within the Optimistic Ethereum smart contracts and is the keccak256 hash of the result of
 * encodeCrossChainMessage.
 *
 * @param message Cross chain message to hash.
 * @returns Canonical hash of the message.
 */
export const hashCrossChainMessage = (message: CrossChainMessage): string => {
  throw new Error('Not implemented')
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
  } else {
    return provider
  }
}

/**
 * Pulls a transaction hash out of a TransactionLike object.
 *
 * @param transaction TransactionLike to convert into a transaction hash.
 * @returns Transaction hash corresponding to the TransactionLike input.
 */
export const getTransactionHash = (transaction: TransactionLike): string => {
  if (typeof transaction === 'string') {
    return transaction
  } else if ((transaction as TransactionReceipt).transactionHash) {
    return (transaction as TransactionReceipt).transactionHash
  } else {
    return (transaction as TransactionResponse).hash
  }
}

// Number of blocks before the first user transaction block is created.
// Should always be 1 for now.
export const NUM_L2_GENESIS_BLOCKS = 1

/**
 * Number of confirmations required for an L1 to L2 transaction to be considered by the Sequencer.
 */
export const L1_TO_L2_TX_CONFIRMATIONS = {
  mainnet: 50,
  kovan: 50,
  local: 50,
  unknown: 50,
}

/**
 * Number of blocks that a transaction needs to wait before it completes its challenge period.
 */
export const CHALLENGE_PERIOD_BLOCKS = {
  mainnet: 40320,
  kovan: 4,
  local: 0,
  unknown: 0,
}

/**
 * Average number of seconds between L1 blocks.
 */
export const L1_BLOCK_INTERVAL_SECONDS = {
  mainnet: 15,
  kovan: 15,
  local: 15,
  unknown: 15,
}
