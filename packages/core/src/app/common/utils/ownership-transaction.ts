/* External Imports */
import BigNum = require('bn.js')
import debug from 'debug'
const log = debug('info:state-update')

/* Internal Imports */
import { abi } from '../eth'
import {
  AbiEncodable,
  Transaction,
  OwnershipParameters,
} from '../../../interfaces'
import { AbiStateObject } from './state-object'
import { AbiRange } from './abi-range'
import { hexStringify } from '../utils'

/**
 * Creates a AbiOwnershipParameters from an encoded AbiOwnershipParameters.
 * @param encoded The encoded AbiOwnershipParameters.
 * @returns the AbiOwnershipParameters.
 */
const fromEncodedOwnershipParams = (
  encoded: string
): AbiOwnershipParameters => {
  const decoded = abi.decode(AbiOwnershipParameters.abiTypes, encoded)
  const newState = AbiStateObject.from(decoded[0])
  const originBlock = new BigNum(decoded[1].toString())
  const maxBlock = new BigNum(decoded[2].toString())
  return new AbiOwnershipParameters(newState, originBlock, maxBlock)
}

/**
 * Represents a basic abi encodable AbiOwnershipParameters
 */
export class AbiOwnershipParameters implements OwnershipParameters, AbiEncodable {
  public static abiTypes = ['bytes', 'uint128', 'uint128']

  constructor(
    readonly newState: AbiStateObject,
    readonly originBlock: BigNum,
    readonly maxBlock: BigNum
  ) {}

  /**
   * @returns the abi encoded AbiOwnershipParameters.
   */
  get encoded(): string {
    return abi.encode(AbiOwnershipParameters.abiTypes, [
      this.newState.encoded,
      hexStringify(this.originBlock),
      hexStringify(this.maxBlock),
    ])
  }

  /**
   * @returns the jsonified AbiOwnershipParameters.
   */
  get jsonified(): any {
    return {
      newState: this.newState.jsonified,
      originBlock: hexStringify(this.originBlock),
      maxBlock: hexStringify(this.maxBlock)
    }
  }
  /**
   * Casts a value to a AbiOwnershipParameters.
   * @param encoded Thing to cast to a AbiOwnershipParameters.
   * @returns the AbiOwnershipParameters.
   */
  public static from(encoded: string): AbiOwnershipParameters {
    if (typeof encoded === 'string') {
      return fromEncodedOwnershipParams(encoded)
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
  const methodId = decoded[1]
  const parameters = AbiOwnershipParameters.from(decoded[2])
  const range = AbiRange.from(decoded[3])
  return new AbiOwnershipTransaction(
    depositAddress,
    methodId,
    parameters,
    range
  )
}

/**
 * Represents a basic abi encodable AbiOwnershipTransaction
 */
export class AbiOwnershipTransaction implements Transaction, AbiEncodable {
  public static abiTypes = ['address', 'bytes32', 'bytes', 'bytes']

  constructor(
    readonly depositAddress: string,
    readonly methodId: string,
    readonly parameters: AbiEncodable,
    readonly range: AbiRange
  ) {}

  /**
   * @returns the abi encoded AbiOwnershipTransaction.
   */
  get encoded(): string {
    return abi.encode(AbiOwnershipTransaction.abiTypes, [
      this.depositAddress,
      this.methodId,
      this.parameters.encoded,
      this.range.encoded,
    ])
  }

  /**
   * @returns the jsonified AbiOwnershipTransaction.
   */
  get jsonified(): any {
    return {
      depositAddress: this.depositAddress,
      methodId: this.methodId,
      parameters: this.parameters.encoded,
      range: this.range.jsonified,
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
