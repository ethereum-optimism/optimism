/* External Imports */
import BigNum = require('bn.js')
import debug from 'debug'
const log = debug('info:abiRange')

/* Internal Imports */
import { abi } from '../eth'
import { AbiEncodable, Range } from '../interfaces/data-types'
import { hexStringify } from '../utils'

/**
 * Creates a Range from an encoded Range.
 * @param encoded The encoded Range.
 * @returns the Range.
 */
const fromEncoded = (encoded: string): AbiRange => {
  const decoded = abi.decode(AbiRange.abiTypes, encoded)
  return new AbiRange(
    new BigNum(decoded[0].toString()),
    new BigNum(decoded[1].toString())
  )
}

/**
 * Represents a basic abi encodable Range
 */
export class AbiRange implements Range, AbiEncodable {
  public static abiTypes = ['uint128', 'uint128']

  constructor(
    readonly start: BigNum,
    readonly end: BigNum,
  ) {}

  /**
   * @returns the abi encoded Range.
   */
  get encoded(): string {
    return abi.encode(AbiRange.abiTypes, [
      hexStringify(this.start),
      hexStringify(this.end)
    ])
  }

  /**
   * Casts a value to a Range.
   * @param value Thing to cast to a Range.
   * @returns the Range.
   */
  public static from(value: string): AbiRange {
    if (typeof value === 'string') {
      return fromEncoded(value)
    }

    throw new Error('Got invalid argument type when casting to AbiRange.')
  }
}
