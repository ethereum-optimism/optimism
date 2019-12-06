/* External Imports */
import { getLogger, remove0x, serializeObject } from '@pigi/core-utils'

/* Internal imports */
import {
  SignedTransaction,
  RollupTransaction,
  Transfer,
  RollupBlock,
} from '../../types'
import { abi, signedTransactionAbiTypes, transferAbiTypes } from './common'

const log = getLogger('abiEncoders')

/**
 * Creates a SignedTransaction from an ABI-encoded SignedTransaction.
 * @param abiEncoded The ABI-encoded SignedTransaction.
 * @returns the SignedTransaction.
 */
export const parseSignedTransactionFromABI = (
  abiEncoded: string
): SignedTransaction => {
  log.debug(`ABI decoding SignedTransaction: ${serializeObject(abiEncoded)}`)
  const [signature, tx] = abi.decode(signedTransactionAbiTypes, abiEncoded)

  return {
    signature,
    transaction: parseTransactionFromABI(tx),
  }
}

/**
 * Parses the provided ABI-encoded transaction into a RollupTransaction
 * @param abiEncoded The ABI-encoded string.
 * @returns the parsed RollupTransaction.
 */
export const parseTransactionFromABI = (
  abiEncoded: string
): RollupTransaction => {
  // If it's not a swap, it must be a transfer
  return parseTransferFromABI(abiEncoded)
}

export const abiDecodeRollupBlock = (abiEncoded: string): RollupBlock => {
  // TODO: actually fill this out
  return {
    blockNumber: 1,
    stateRoot: '',
    signedTransactions: [],
  }
}

/*********************
 * PRIVATE FUNCTIONS *
 *********************/

/**
 * Creates a Transfer from an ABI-encoded Transfer.
 * @param abiEncoded The ABI-encoded Transfer.
 * @returns the Transfer.
 */
const parseTransferFromABI = (abiEncoded: string): Transfer => {
  log.debug(`ABI decoding Transfer: ${serializeObject(abiEncoded)}`)
  const [sender, recipient, tokenType, amount] = abi.decode(
    transferAbiTypes,
    abiEncoded
  )
  return {
    sender: remove0x(sender),
    recipient: remove0x(recipient),
    tokenType,
    amount,
  }
}

const getSlotIndex = (slotIndexString: string): string => {
  return remove0x(slotIndexString)
}
