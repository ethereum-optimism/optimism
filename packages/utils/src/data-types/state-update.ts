/* External Imports */
import BigNum = require('bn.js')
import debug from 'debug'
const log = debug('info:state-update')

/* Internal Imports */
import { abi } from '../eth'
import { StateUpdate, AbiEncodable, Range } from '../interfaces/data-types'
import { hexStringify } from '../misc'
import { AbiStateObject } from './state-object'

/**
 * Creates a AbiStateUpdate from an encoded AbiStateUpdate.
 * @param encoded The encoded AbiStateUpdate.
 * @returns the AbiStateUpdate.
 */
const fromEncoded = (encoded: string): AbiStateUpdate => {
  const decoded = abi.decode(AbiStateUpdate.abiTypes, encoded)
  const stateObject = AbiStateObject.from(decoded[0])
  return new AbiStateUpdate(
    stateObject,
    { start: new BigNum(decoded[1].toString()), end: new BigNum(decoded[2].toString()) },
    decoded[3],
    decoded[4]
  )
}

/**
 * Represents a basic abi encodable AbiStateUpdate
 */
export class AbiStateUpdate implements StateUpdate, AbiEncodable {
  public static abiTypes = ['bytes', 'uint128', 'uint128', 'uint32', 'address']

  constructor(
    readonly stateObject: AbiStateObject,
    readonly range: Range,
    readonly blockNumber: number,
    readonly plasmaContract: string
  ) {}

  /**
   * @returns the abi encoded AbiStateUpdate.
   */
  get encoded(): string {
    log('this is the state object:')
    log(this.stateObject.encoded)
    return abi.encode(AbiStateUpdate.abiTypes, [
      this.stateObject.encoded,
      hexStringify(this.range.start),
      hexStringify(this.range.end),
      this.blockNumber,
      this.plasmaContract,
    ])
  }

  /**
   * Casts a value to a AbiStateUpdate.
   * @param value Thing to cast to a AbiStateUpdate.
   * @returns the AbiStateUpdate.
   */
  public static from(value: string): AbiStateUpdate {
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
  public equals(other: AbiStateUpdate): boolean {
    return this.encoded === other.encoded
  }
}
