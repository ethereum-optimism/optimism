/* Internal Imports */
import { abi, keccak256 } from '../utils'
import { StateUpdate } from './state-update'

const TRANSACTION_ABI_TYPES = ['bytes', 'bytes']

interface TransactionArgs {
  stateUpdate: StateUpdate
  witness: string
}

/**
 * Creates a Transaction from an encoded Transaction.
 * @param encoded The encoded Transaction.
 * @returns the Transaction.
 */
const fromEncoded = (encoded: string): Transaction => {
  const decoded = abi.decode(TRANSACTION_ABI_TYPES, encoded)
  return new Transaction({
    stateUpdate: decoded[0],
    witness: decoded[1],
  })
}

/**
 * Represents a basic plasma chain transaction.
 */
export class Transaction {
  public stateUpdate: StateUpdate
  public witness: string

  constructor(args: TransactionArgs) {
    this.stateUpdate = args.stateUpdate
    this.witness = args.witness
  }

  /**
   * @returns the hash of the transaction.
   */
  get hash(): string {
    return keccak256(this.encoded)
  }

  /**
   * @returns the encoded transaction.
   */
  get encoded(): string {
    return abi.encode(TRANSACTION_ABI_TYPES, [this.stateUpdate, this.witness])
  }

  /**
   * Casts a value to a Transaction.
   * @param value Thing to cast to a Transaction.
   * @returns the Transaction.
   */
  public static from(value: string): Transaction {
    if (typeof value === 'string') {
      return fromEncoded(value)
    }

    throw new Error('Got invalid argument type when casting to Transaction.')
  }
}
