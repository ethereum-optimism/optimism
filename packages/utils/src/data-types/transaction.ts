/* Internal Imports */
import { abi, keccak256 } from '../eth'
import { AbiStateUpdate } from './state-update'

const TRANSACTION_ABI_TYPES = ['bytes', 'bytes']

interface TransactionArgs {
  stateUpdate: AbiStateUpdate
  transactionWitness: string
}

/**
 * Creates a AbiTransaction from an encoded AbiTransaction.
 * @param encoded The encoded AbiTransaction.
 * @returns the AbiTransaction.
 */
const fromEncoded = (encoded: string): AbiTransaction => {
  const decoded = abi.decode(TRANSACTION_ABI_TYPES, encoded)
  return new AbiTransaction({
    stateUpdate: decoded[0],
    transactionWitness: decoded[1],
  })
}

/**
 * Represents a basic plasma chain AbiTransaction.
 */
export class AbiTransaction {
  public stateUpdate: AbiStateUpdate
  public transactionWitness: string

  constructor(args: TransactionArgs) {
    this.stateUpdate = args.stateUpdate
    this.transactionWitness = args.transactionWitness
  }

  /**
   * @returns the hash of the AbiTransaction.
   */
  get hash(): string {
    return keccak256(this.encoded)
  }

  /**
   * @returns the encoded AbiTransaction.
   */
  get encoded(): string {
    return abi.encode(TRANSACTION_ABI_TYPES, [
      this.stateUpdate,
      this.transactionWitness,
    ])
  }

  /**
   * Casts a value to a AbiTransaction.
   * @param value Thing to cast to a AbiTransaction.
   * @returns the AbiTransaction.
   */
  public static from(value: string): AbiTransaction {
    if (typeof value === 'string') {
      return fromEncoded(value)
    }

    throw new Error('Got invalid argument type when casting to AbiTransaction.')
  }
}
