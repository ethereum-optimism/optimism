/* External Imports */
import { add0x } from '@pigi/core-utils'

/* Internal Imports */
import { TokenType, RollupTransaction, Address, EVMBytecode } from '../types'

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

/**
 * Takes EVMBytecode and serializes it into a single Buffer.
 *
 * @param bytecode The bytecode to serialize into a single Buffer.
 * @returns The resulting Buffer.
 */
export const bytecodeToBuffer = (bytecode: EVMBytecode): Buffer => {
  return Buffer.concat(
    bytecode.map((b) => {
      return !!b.consumedBytes
        ? Buffer.concat([b.opcode.code, b.consumedBytes])
        : b.opcode.code
    })
  )
}
