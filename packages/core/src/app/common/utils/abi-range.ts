/* External Imports */
import BigNum = require('bn.js')
import debug from 'debug'
const log = debug('info:state-update')

/* Internal Imports */
import { abi } from '../eth'
import { AbiEncodable, Range } from '../../../interfaces/common/utils'
import { hexStringify } from '../utils'

/**
 * Creates a AbiStateUpdate from an encoded AbiStateUpdate.
 * @param encoded The encoded AbiStateUpdate.
 * @returns the AbiStateUpdate.
 */
const fromEncoded = (encoded: string): AbiRange => {
  const decoded = abi.decode(AbiRange.abiTypes, encoded)
  return new AbiRange(
    new BigNum(decoded[0].toString()),
    new BigNum(decoded[1].toString())
  )
}

/**
 * Represents a basic abi encodable AbiRange
 */
export class AbiRange implements Range, AbiEncodable {
  public static abiTypes = ['uint128', 'uint128']

  constructor(
    readonly start: BigNum,
    readonly end: BigNum
  ) {}

  /**
   * @returns the abi encoded AbiRange.
   */
  get encoded(): string {
    return abi.encode(AbiRange.abiTypes, [
      hexStringify(this.start),
      hexStringify(this.end)
    ])
  }

  /**
   * Casts a value to a AbiRange.
   * @param value Thing to cast to a AbiRange.
   * @returns the AbiRange.
   */
  public static from(value: string): AbiRange {
    if (typeof value === 'string') {
      return fromEncoded(value)
    }

    throw new Error('Got invalid argument type when casting to AbiStateUpdate.')
  }

  /**
   * Determines if this object equals another.
   * @param other Object to compare to.
   * @returns `true` if the two are equal, `false` otherwise.
   */
  public equals(other: AbiRange): boolean {
    return this.encoded === other.encoded
  }
}
