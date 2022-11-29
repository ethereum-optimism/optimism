import { BigNumberish, BigNumber } from '@ethersproject/bignumber'
import { keccak256 } from '@ethersproject/keccak256'
import { defaultAbiCoder } from '@ethersproject/abi'

import {
  decodeVersionedNonce,
  encodeCrossDomainMessageV0,
  encodeCrossDomainMessageV1,
  big0,
  big1,
} from './encoding'

/**
 * Bedrock output oracle data.
 */
export interface BedrockOutputData {
  outputRoot: string
  l1Timestamp: number
  l2BlockNumber: number
  l2OutputIndex: number
}

/**
 * Bedrock state commitment
 */
export interface OutputRootProof {
  version: string
  stateRoot: string
  messagePasserStorageRoot: string
  latestBlockhash: string
}

/**
 * Bedrock proof data required to finalize an L2 to L1 message.
 */
export interface BedrockCrossChainMessageProof {
  outputRootProof: OutputRootProof
  withdrawalProof: string[]
}

/**
 * Parameters that govern the L2OutputOracle.
 */
export type L2OutputOracleParameters = {
  submissionInterval: number
  startingBlockNumber: number
  l2BlockTime: number
}

/**
 * Hahses a cross domain message.
 *
 * @param nonce     The cross domain message nonce
 * @param sender    The sender of the cross domain message
 * @param target    The target of the cross domain message
 * @param value     The value being sent with the cross domain message
 * @param gasLimit  The gas limit of the cross domain execution
 * @param data      The data passed along with the cross domain message
 */
export const hashCrossDomainMessage = (
  nonce: BigNumber,
  sender: string,
  target: string,
  value: BigNumber,
  gasLimit: BigNumber,
  data: string
) => {
  const [, version] = decodeVersionedNonce(nonce)
  if (version.eq(big0)) {
    return hashCrossDomainMessagev0(target, sender, data, nonce)
  } else if (version.eq(big1)) {
    return hashCrossDomainMessagev1(
      nonce,
      sender,
      target,
      value,
      gasLimit,
      data
    )
  }
  throw new Error(`unknown version ${version.toString()}`)
}

/**
 * Hahses a V0 cross domain message
 *
 * @param target    The target of the cross domain message
 * @param sender    The sender of the cross domain message
 * @param data      The data passed along with the cross domain message
 * @param nonce     The cross domain message nonce
 */
export const hashCrossDomainMessagev0 = (
  target: string,
  sender: string,
  data: string,
  nonce: BigNumber
) => {
  return keccak256(encodeCrossDomainMessageV0(target, sender, data, nonce))
}

/**
 * Hahses a V1 cross domain message
 *
 * @param nonce     The cross domain message nonce
 * @param sender    The sender of the cross domain message
 * @param target    The target of the cross domain message
 * @param value     The value being sent with the cross domain message
 * @param gasLimit  The gas limit of the cross domain execution
 * @param data      The data passed along with the cross domain message
 */
export const hashCrossDomainMessagev1 = (
  nonce: BigNumber,
  sender: string,
  target: string,
  value: BigNumberish,
  gasLimit: BigNumberish,
  data: string
) => {
  return keccak256(
    encodeCrossDomainMessageV1(nonce, sender, target, value, gasLimit, data)
  )
}

/**
 * Hashes a withdrawal
 *
 * @param nonce     The cross domain message nonce
 * @param sender    The sender of the cross domain message
 * @param target    The target of the cross domain message
 * @param value     The value being sent with the cross domain message
 * @param gasLimit  The gas limit of the cross domain execution
 * @param data      The data passed along with the cross domain message
 */
export const hashWithdrawal = (
  nonce: BigNumber,
  sender: string,
  target: string,
  value: BigNumber,
  gasLimit: BigNumber,
  data: string
): string => {
  const types = ['uint256', 'address', 'address', 'uint256', 'uint256', 'bytes']
  const encoded = defaultAbiCoder.encode(types, [
    nonce,
    sender,
    target,
    value,
    gasLimit,
    data,
  ])
  return keccak256(encoded)
}

/**
 * Hahses an output root proof
 *
 * @param proof OutputRootProof
 */
export const hashOutputRootProof = (proof: OutputRootProof): string => {
  return keccak256(
    defaultAbiCoder.encode(
      ['bytes32', 'bytes32', 'bytes32', 'bytes32'],
      [
        proof.version,
        proof.stateRoot,
        proof.messagePasserStorageRoot,
        proof.latestBlockhash,
      ]
    )
  )
}
