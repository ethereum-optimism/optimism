import { getContractInterface } from '@eth-optimism/contracts'
import { BigNumber, ethers } from 'ethers'

import { CoreCrossChainMessage } from '../interfaces'
import { UniversalMessengerIface } from './contracts'

/**
 * Returns the v0 message encoding.
 *
 * @param message Message to encode.
 * @returns v0 message encoding.
 */
export const encodeV0 = (message: CoreCrossChainMessage): string => {
  return getContractInterface('L2CrossDomainMessenger').encodeFunctionData(
    'relayMessage',
    [message.target, message.sender, message.message, message.messageNonce]
  )
}

/**
 * Returns the v1 message encoding.
 *
 * @param message Message to encode.
 * @returns v1 message encoding.
 */
export const encodeV1 = (message: CoreCrossChainMessage): string => {
  return UniversalMessengerIface.encodeFunctionData('relayMessage', [
    message.messageNonce,
    message.sender,
    message.target,
    message.value,
    message.minGasLimit,
    message.message,
  ])
}

/**
 * Pulls version byte from nonce.
 *
 * @param nonce Nonce to pull version byte from.
 * @returns Version byte.
 */
export const getVersionFromNonce = (nonce: BigNumber): number => {
  return nonce.shr(240).toNumber()
}

/**
 * Returns the canonical encoding of a cross chain message. This encoding is used in various
 * locations within the Optimism smart contracts.
 *
 * @param message Cross chain message to encode.
 * @returns Canonical encoding of the message.
 */
export const encodeCrossChainMessage = (
  message: CoreCrossChainMessage
): string => {
  const version = getVersionFromNonce(message.messageNonce)
  switch (version) {
    case 0:
      return encodeV0(message)
    case 1:
      return encodeV1(message)
    default:
      throw new Error(`unsupported message version: ${version}`)
  }
}

/**
 * Returns the canonical hash of a cross chain message. This hash is used in various locations
 * within the Optimism smart contracts and is the keccak256 hash of the result of
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

/**
 * Computes the withdrawal hash for a given message.
 *
 * @param message Message to compute the withdrawal hash for.
 * @returns Computed withdrawal hash.
 */
export const hashWithdrawal = (message: CoreCrossChainMessage): string => {
  return ethers.utils.solidityKeccak256(
    ['bytes'],
    [
      ethers.utils.defaultAbiCoder.encode(
        ['uint256', 'address', 'address', 'uint256', 'uint256', 'bytes'],
        [
          message.messageNonce,
          message.sender,
          message.target,
          message.value,
          message.minGasLimit,
          message.message,
        ]
      ),
    ]
  )
}
