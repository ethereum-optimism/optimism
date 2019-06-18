/* Internal Imports */
import { abi } from '../eth'
import { StateObject, AbiEncodable } from '../../../interfaces'

/**
 * Creates a StateObject from an encoded StateObject.
 * @param encoded The encoded StateObject.
 * @returns the StateObject.
 */
const fromEncoded = (encoded: string): AbiStateObject => {
  const decoded = abi.decode(AbiStateObject.abiTypes, encoded)
  return new AbiStateObject(decoded[0], decoded[1])
}

/**
 * Represents a basic abi encodable StateObject
 */
export class AbiStateObject implements StateObject, AbiEncodable {
  public static abiTypes = ['address', 'bytes']

  constructor(readonly predicate: string, readonly parameters: string) {}

  /**
   * @returns the abi encoded StateObject.
   */
  get encoded(): string {
    return abi.encode(AbiStateObject.abiTypes, [
      this.predicate,
      this.parameters,
    ])
  }

  /**
   * Casts a value to a StateObject.
   * @param value Thing to cast to a StateObject.
   * @returns the StateObject.
   */
  public static from(value: string): AbiStateObject {
    if (typeof value === 'string') {
      return fromEncoded(value)
    }

    throw new Error('Got invalid argument type when casting to StateObject.')
  }
}
