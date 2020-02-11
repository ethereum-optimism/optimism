/* External Imports */
import { add0x, getLogger } from '@eth-optimism/core-utils'
import { Transaction } from '@eth-optimism/rollup-core'

/* Internal Imports */

import { abi, transactionAbiTypes } from './common'

const log = getLogger('abiEncoders')

/**
 * ABI-encodes the provided Transaction.
 *
 * @param tx The transaction to ABI-encode.
 * @returns The ABI-encoded Transaction as a string.
 */
export const abiEncodeTransaction = (tx: Transaction): string => {
  return abi.encode(transactionAbiTypes, [
    add0x(tx.ovmEntrypoint),
    add0x(tx.ovmCalldata),
  ])
}
