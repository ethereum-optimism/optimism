/* External Imports */
import BigNum = require('bn.js')
import {
  Transaction,
  Range,
  AbiStateObject,
  EcdsaSignature,
  AbiEncodable,
  abi,
  keccak256,
  hexStringify,
  StateUpdate,
} from '@pigi/core'

/**
 * Creates a Transaction from an encoded Transaction.
 * @param encoded The encoded Transaction.
 * @returns the Transaction.
 */
const fromEncoded = (encoded: string): OwnershipTransaction => {
  const decoded = abi.decode(OwnershipTransaction.abiTypes, encoded)
  return new OwnershipTransaction(
    decoded[0],
    decoded[1],
    parseInt(decoded[2].toString(), 10),
    {
      start: new BigNum(decoded[3].toString()),
      end: new BigNum(decoded[4].toString()),
    },
    decoded[5],
    { newStateObject: AbiStateObject.from(decoded[6]) },
    { v: decoded[7], r: decoded[8], s: decoded[9] }
  )
}

/**
 * Represents an Ownership transaction
 */
export class OwnershipTransaction implements Transaction, AbiEncodable {
  public static abiTypes = [
    'address',
    'address',
    'uint64',
    'uint128',
    'uint128',
    'bytes1',
    'bytes',
    'bytes32',
    'bytes32',
    'bytes1',
  ]

  constructor(
    readonly plasmaContract: string,
    readonly depositContract: string,
    readonly block: number,
    readonly range: Range,
    readonly methodId: string,
    readonly parameters: { newStateObject: AbiStateObject },
    readonly witness: EcdsaSignature
  ) {}

  /**
   * @returns the encoded transaction.
   */
  get encoded(): string {
    return abi.encode(OwnershipTransaction.abiTypes, [
      this.plasmaContract,
      this.depositContract,
      this.block,
      hexStringify(this.range.start),
      hexStringify(this.range.end),
      this.methodId,
      this.parameters.newStateObject.encoded,
      this.witness.v,
      this.witness.r,
      this.witness.s,
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
