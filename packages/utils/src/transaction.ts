import BigNum = require('bn.js')
import { abi, keccak256 } from './utils'
import { StateObject } from './state-object'
import { MerkleTreeNode } from './sum-tree'

const TRANSACTION_ABI_TYPES = ['uint256', 'bytes[]', 'bytes', 'bytes']

interface TransactionArgs {
  block: number | BigNum
  inclusionProof: MerkleTreeNode[]
  witness: string
  newState: StateObject
}

/**
 * Creates a Transaction from an encoded Transaction.
 * @param encoded The encoded Transaction.
 * @returns the Transaction.
 */
const fromEncoded = (encoded: string): Transaction => {
  const decoded = abi.decode(TRANSACTION_ABI_TYPES, encoded)
  return new Transaction({
    block: decoded[0],
    inclusionProof: decoded[1],
    witness: decoded[2],
    newState: decoded[3],
  })
}

/**
 * Represents a basic plasma chain transaction.
 */
export class Transaction {
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

  public block: BigNum
  public inclusionProof: MerkleTreeNode[]
  public witness: string
  public newState: StateObject

  constructor(args: TransactionArgs) {
    this.block = new BigNum(args.block, 'hex')
    this.inclusionProof = args.inclusionProof
    this.witness = args.witness
    this.newState = args.newState
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
      this.block,
      this.inclusionProof,
      this.witness,
      this.newState.encoded,
    ])
  }
}
