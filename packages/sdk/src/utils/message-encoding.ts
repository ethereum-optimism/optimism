import { getContractInterface } from '@eth-optimism/contracts'
import { ethers } from 'ethers'
import { CoreCrossChainMessage } from '../interfaces'

/**
 * Returns the canonical encoding of a cross chain message. This encoding is used in various
 * locations within the Optimistic Ethereum smart contracts.
 *
 * @param message Cross chain message to encode.
 * @returns Canonical encoding of the message.
 */
export const encodeCrossChainMessage = (
  message: CoreCrossChainMessage
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
  message: CoreCrossChainMessage
): string => {
  return ethers.utils.solidityKeccak256(
    ['bytes'],
    [encodeCrossChainMessage(message)]
  )
}
