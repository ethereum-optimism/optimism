/* External Imports */
import BigNum = require('bn.js')
import debug from 'debug'
const log = debug('info:state-update')

/* Internal Imports */
import { abi } from '../eth'
import { AbiEncodable, Transaction, OwnershipParameters } from '../../../interfaces'
import { AbiStateObject } from './state-object';
import { AbiRange } from './abi-range'
import { hexStringify } from '../utils'

/**
 * Creates a AbiOwnershipParameters from an encoded AbiOwnershipParameters.
 * @param encoded The encoded AbiOwnershipParameters.
 * @returns the AbiOwnershipParameters.
 */
const fromEncodedOwnershipParams = (encoded: string): AbiOwnershipParameters => {
  const decoded = abi.decode(AbiOwnershipParameters.abiTypes, encoded)
  const newState = AbiStateObject.from(decoded[0])
  const originBlock = new BigNum(decoded[1].toString())
  const maxBlock = new BigNum(decoded[2].toString())
  return new AbiOwnershipParameters(
    newState,
    originBlock,
    maxBlock
  )
}

/**
 * Represents a basic abi encodable AbiOwnershipParameters
 */
export class AbiOwnershipParameters 
implements OwnershipParameters, AbiEncodable {
  public static abiTypes = ['bytes', 'uint128', 'uint128']

  constructor(
    readonly newState: AbiStateObject,
    readonly originBlock: BigNum,
    readonly maxBlock: BigNum
  ) {}

  /**
   * @returns the abi encoded AbiStateUpdate.
   */
  get encoded(): string {
    return abi.encode(AbiOwnershipParameters.abiTypes, [
      this.newState.encoded,
      hexStringify(this.originBlock),
      hexStringify(this.maxBlock)
    ])
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
 * Creates a AbiOwnershipParameters from an encoded AbiOwnershipParameters.
 * @param encoded The encoded AbiOwnershipParameters.
 * @returns the AbiOwnershipParameters.
 */
const fromEncodedOwnershipTransaction = (encoded: string): AbiOwnershipTransaction => {
  const decoded = abi.decode(AbiOwnershipTransaction.abiTypes, encoded)
  const depositContract = decoded[0]
  const methodId = decoded[1]
  const parameters = AbiOwnershipParameters.from(decoded[2])
  const range = AbiRange.from(decoded[3])
  return new AbiOwnershipTransaction(
    depositContract,
    methodId,
    parameters,
    range
  )
}


/**
 * Represents a basic abi encodable AbiTransaction
 */
export class AbiOwnershipTransaction 
implements Transaction, AbiEncodable {
  public static abiTypes = ['address', 'bytes32', 'bytes', 'bytes']

  constructor(
    readonly depositContract: string,
    readonly methodId: string,
    readonly parameters: AbiEncodable,
    readonly range: AbiRange
  ) {}

  /**
   * @returns the abi encoded AbiStateUpdate.
   */
  get encoded(): string {
    return abi.encode(AbiOwnershipTransaction.abiTypes, [
      this.depositContract,
      this.methodId,
      this.parameters.encoded,
      this.range.encoded,
    ])
  }

  /**
   * Casts a value to a AbiStateUpdate.
   * @param value Thing to cast to a AbiStateUpdate.
   * @returns the AbiStateUpdate.
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