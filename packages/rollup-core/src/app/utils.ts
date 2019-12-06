/* External Imports */
import { add0x } from '@pigi/core-utils'

/* Internal Imports */
import { TokenType, RollupTransaction, Address } from '../types'

/* Constants */
export const NON_EXISTENT_SLOT_INDEX = add0x(
  Buffer.alloc(1)
    .fill('\x00')
    .toString('hex')
)

/* Utilities */
export const generateTransferTx = (
  sender: Address,
  recipient: Address,
  tokenType: TokenType,
  amount: number
): RollupTransaction => {
  return {
    sender,
    recipient,
    tokenType,
    amount,
  }
}
