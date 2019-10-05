/* External Imports */
import { add0x, BigNumber, getLogger, serializeObject } from '@pigi/core'

/* Internal Imports */
import {
  CreateAndTransferTransition,
  isSwapTransaction,
  isTransferTransaction,
  SignedTransaction,
  State,
  StateReceipt,
  Swap,
  SwapTransition,
  RollupTransaction,
  Transfer,
  TransferTransition,
  RollupTransition,
  isSwapTransition,
  isTransferTransition,
  isCreateAndTransferTransition,
} from '../types'
import {
  abi,
  createAndTransferTransitionAbiTypes,
  signedTransactionAbiTypes,
  stateAbiTypes,
  stateReceiptAbiTypes,
  swapAbiTypes,
  swapTransitionAbiTypes,
  transferAbiTypes,
  transferTransitionAbiTypes,
} from './common'
import { PIGI_TOKEN_TYPE, UNI_TOKEN_TYPE } from '../index'
import { ethers } from 'ethers'

const log = getLogger('abiEncoders')

/**
 * ABI-encodes the provided SignedTransaction.
 * @param signedTransaction The SignedTransaction to AbI-encode.
 * @returns The ABI-encoded SignedTransaction as a string.
 */
export const abiEncodeSignedTransaction = (
  signedTransaction: SignedTransaction
): string => {
  log.debug(
    `ABI encoding SignedTransaction: ${serializeObject(signedTransaction)}`
  )
  return abi.encode(signedTransactionAbiTypes, [
    signedTransaction.signature,
    abiEncodeTransaction(signedTransaction.transaction),
  ])
}

/**
 * ABI-encodes the provided RollupTransaction.
 * @param transaction The transaction to AbI-encode.
 * @returns The ABI-encoded RollupTransaction as a string.
 */
export const abiEncodeTransaction = (
  transaction: RollupTransaction
): string => {
  if (isSwapTransaction(transaction)) {
    return abiEncodeSwap(transaction)
  } else if (isTransferTransaction(transaction)) {
    return abiEncodeTransfer(transaction)
  }
  const message: string = `Unknown transaction type: ${JSON.stringify(
    transaction
  )}`
  log.error(message)
  throw Error(message)
}

/**
 * ABI-encodes the provided RollupTransition.
 * @param transition The transition to AbI-encode.
 * @returns The ABI-encoded RollupTransition as a string.
 */
export const abiEncodeTransition = (transition: RollupTransition): string => {
  if (isSwapTransition(transition)) {
    return abiEncodeSwapTransition(transition)
  } else if (isTransferTransition(transition)) {
    return abiEncodeTransferTransition(transition)
  } else if (isCreateAndTransferTransition(transition)) {
    return abiEncodeCreateAndTransferTransition(transition)
  }
  const message: string = `Unknown transition type: ${JSON.stringify(
    transition
  )}`
  log.error(message)
  throw Error(message)
}

/**
 * ABI-encodes the provided State
 * @param state The state to ABI-encode
 * @returns the ABI-encoded string.
 */
export const abiEncodeState = (state: State): string => {
  log.debug(`ABI encoding State: ${serializeObject(state)}`)
  return abi.encode(stateAbiTypes, [
    state.pubkey,
    state.balances[UNI_TOKEN_TYPE],
    state.balances[PIGI_TOKEN_TYPE],
  ])
}

/**
 * ABI-encodes the provided StateReceipt
 * @param stateReceipt The StateReceipt to ABI-encode
 * @returns the ABI-encoded string.
 */
export const abiEncodeStateReceipt = (stateReceipt: StateReceipt): string => {
  log.debug(`ABI encoding StateReceipt: ${serializeObject(stateReceipt)}`)
  return abi.encode(stateReceiptAbiTypes, [
    add0x(stateReceipt.stateRoot),
    stateReceipt.blockNumber,
    stateReceipt.transitionIndex,
    stateReceipt.slotIndex,
    stateReceipt.inclusionProof.map((hex) => add0x(hex)),
    abiEncodeState(stateReceipt.state),
  ])
}

/*********************
 * PRIVATE FUNCTIONS *
 *********************/

/**
 * ABI-encodes the provided Transfer.
 * @param transfer The Transfer to AbI-encode.
 * @returns The ABI-encoded Transfer as a string.
 */
const abiEncodeTransfer = (transfer: Transfer): string => {
  log.debug(`ABI encoding Transfer: ${serializeObject(transfer)}`)
  return abi.encode(transferAbiTypes, [
    transfer.sender,
    transfer.recipient,
    transfer.tokenType,
    transfer.amount,
  ])
}

/**
 * ABI-encodes the provided Swap.
 * @param swap The Swap to AbI-encode.
 * @returns The ABI-encoded Swap as a string.
 */
const abiEncodeSwap = (swap: Swap): string => {
  log.debug(`ABI encoding Swap: ${serializeObject(swap)}`)
  return abi.encode(swapAbiTypes, [
    swap.sender,
    swap.tokenType,
    swap.inputAmount,
    swap.minOutputAmount,
    swap.timeout,
  ])
}

/**
 * ABI-encodes the provided SwapTransition.
 * @param trans The transition to AbI-encode.
 * @returns The ABI-encoded SwapTransition as a string.
 */
const abiEncodeSwapTransition = (trans: SwapTransition): string => {
  log.debug(`ABI encoding SwapTransition: ${serializeObject(trans)}`)
  return abi.encode(swapTransitionAbiTypes, [
    add0x(trans.stateRoot),
    trans.senderSlotIndex,
    trans.uniswapSlotIndex,
    trans.tokenType,
    trans.inputAmount,
    trans.minOutputAmount,
    trans.timeout,
    add0x(trans.signature),
  ])
}

/**
 * ABI-encodes the provided TransferTransition.
 * @param trans The transition to AbI-encode.
 * @returns The ABI-encoded TransferTransition as a string.
 */
const abiEncodeTransferTransition = (trans: TransferTransition): string => {
  log.debug(`ABI encoding TransferTransition: ${serializeObject(trans)}`)
  return abi.encode(transferTransitionAbiTypes, [
    add0x(trans.stateRoot),
    trans.senderSlotIndex,
    trans.recipientSlotIndex,
    trans.tokenType,
    trans.amount,
    add0x(trans.signature),
  ])
}

/**
 * ABI-encodes the provided CreateAndTransferTransition.
 * @param trans The transition to AbI-encode.
 * @returns The ABI-encoded CreateAndTransferTransition as a string.
 */
const abiEncodeCreateAndTransferTransition = (
  trans: CreateAndTransferTransition
): string => {
  log.debug(
    `ABI encoding CreateAndTransferTransition: ${serializeObject(trans)}`
  )
  return abi.encode(createAndTransferTransitionAbiTypes, [
    add0x(trans.stateRoot),
    trans.senderSlotIndex,
    trans.recipientSlotIndex,
    trans.createdAccountPubkey,
    trans.tokenType,
    trans.amount,
    add0x(trans.signature),
  ])
}
