/* External Imports */
import { add0x, getLogger, serializeObject } from '@pigi/core-utils'

/* Internal Imports */
import {
  isTransferTransaction,
  SignedTransaction,
  RollupTransaction,
  Transfer,
  RollupBlock,
} from '../../types'
import { abi, signedTransactionAbiTypes, transferAbiTypes } from './common'

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
  if (isTransferTransaction(transaction)) {
    return abiEncodeTransfer(transaction)
  }
  const message: string = `Unknown transaction type: ${JSON.stringify(
    transaction
  )}`
  log.error(message)
  throw Error(message)
}

export const abiEncodeRollupBlock = (rollupBlock: RollupBlock): string => {
  // TODO: actually ABI encode blocks when they are solidified.
  return ''
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
    add0x(transfer.sender),
    add0x(transfer.recipient),
    transfer.tokenType,
    transfer.amount,
  ])
}
