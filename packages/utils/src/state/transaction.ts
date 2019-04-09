/* Internal Imports */
import { abi, keccak256 } from '../eth'
import { StateUpdate } from './state-update'

const TRANSACTION_ABI_TYPES = ['bytes', 'bytes']

interface TransactionArgs {
  stateUpdate: StateUpdate
  transactionWitness: string
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
    transactionWitness: decoded[1],
  })
}

/**
 * Represents a basic plasma chain transaction.
 */
export class Transaction {
  public stateUpdate: StateUpdate
  public transactionWitness: string

  constructor(args: TransactionArgs) {
    this.stateUpdate = args.stateUpdate
    this.transactionWitness = args.transactionWitness
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
    return abi.encode(TRANSACTION_ABI_TYPES, [
      this.stateUpdate,
      this.transactionWitness,
    ])
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
