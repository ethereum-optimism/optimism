/* External Imports */
import {
  abi,
  AbiRange,
  AbiEncodable,
  BigNumber,
  hexStringify,
} from '@pigi/core-utils'
import { AbiStateObject } from './state-object'
import { OwnershipBody, Transaction } from '../../types'

/* Internal Imports */

/**
 * Creates a AbiOwnershipBody from an encoded AbiOwnershipBody.
 * @param encoded The encoded AbiOwnershipBody.
 * @returns the AbiOwnershipBody.
 */
const fromEncodedOwnershipBody = (encoded: string): AbiOwnershipBody => {
  const decoded = abi.decode(AbiOwnershipBody.abiTypes, encoded)
  const newState = AbiStateObject.from(decoded[0])
  const originBlock = new BigNumber(decoded[1].toString())
  const maxBlock = new BigNumber(decoded[2].toString())
  return new AbiOwnershipBody(newState, originBlock, maxBlock)
}

/**
 * Represents a basic abi encodable AbiOwnershipBody
 */
export class AbiOwnershipBody implements OwnershipBody, AbiEncodable {
  public static abiTypes = ['bytes', 'uint128', 'uint128']

  constructor(
    readonly newState: AbiStateObject,
    readonly originBlock: BigNumber,
    readonly maxBlock: BigNumber
  ) {}

  /**
   * @returns the abi encoded AbiOwnershipBody.
   */
  get encoded(): string {
    return abi.encode(AbiOwnershipBody.abiTypes, [
      this.newState.encoded,
      hexStringify(this.originBlock),
      hexStringify(this.maxBlock),
    ])
  }

  /**
   * @returns the jsonified AbiOwnershipBody.
   */
  get jsonified(): any {
    return {
      newState: this.newState.jsonified,
      originBlock: hexStringify(this.originBlock),
      maxBlock: hexStringify(this.maxBlock),
    }
  }
  /**
   * Casts a value to a AbiOwnershipBody.
   * @param encoded Thing to cast to a AbiOwnershipBody.
   * @returns the AbiOwnershipBody.
   */
  public static from(encoded: string): AbiOwnershipBody {
    if (typeof encoded === 'string') {
      return fromEncodedOwnershipBody(encoded)
    }

    throw new Error('Got invalid argument type when casting to AbiStateUpdate.')
  }
}

/**
 * Creates a AbiOwnershipTransaction from an encoded AbiOwnershipTransaction.
 * @param encoded The encoded AbiOwnershipTransaction.
 * @returns the AbiOwnershipTransaction.
 */
const fromEncodedOwnershipTransaction = (
  encoded: string
): AbiOwnershipTransaction => {
  const decoded = abi.decode(AbiOwnershipTransaction.abiTypes, encoded)
  const depositAddress = decoded[0]
  const range = AbiRange.from(decoded[1])
  const body = AbiOwnershipBody.from(decoded[2])
  return new AbiOwnershipTransaction(depositAddress, range, body)
}

/**
 * Represents a basic abi encodable AbiOwnershipTransaction
 */
export class AbiOwnershipTransaction implements Transaction, AbiEncodable {
  // [depositAddress, range obj, body obj]
  public static abiTypes = ['address', 'bytes', 'bytes']

  constructor(
    readonly depositAddress: string,
    readonly range: AbiRange,
    readonly body: AbiEncodable
  ) {}

  /**
   * @returns the abi encoded AbiOwnershipTransaction.
   */
  get encoded(): string {
    return abi.encode(AbiOwnershipTransaction.abiTypes, [
      this.depositAddress,
      this.range.encoded,
      this.body.encoded,
    ])
  }

  /**
   * @returns the jsonified AbiOwnershipTransaction.
   */
  get jsonified(): any {
    return {
      depositAddress: this.depositAddress,
      range: this.range.jsonified,
      body: this.body.encoded,
    }
  }

  /**
   * Casts a value to a AbiOwnershipTransaction.
   * @param value Thing to cast to a AbiOwnershipTransaction.
   * @returns the AbiOwnershipTransaction.
   */
  public static from(encoded: string): AbiOwnershipTransaction {
    if (typeof encoded === 'string') {
      return fromEncodedOwnershipTransaction(encoded)
    }

    throw new Error('Got invalid argument type when casting to AbiStateUpdate.')
  }

  /**
   * Determines if this object equals another.
   * @param other Object to compare to.
   * @returns `true` if the two are equal, `false` otherwise.
   */
  public equals(other: AbiOwnershipTransaction): boolean {
    return this.encoded === other.encoded
  }
}
