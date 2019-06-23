/* Internal Imports */
import { abi } from 'src/app'
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

  constructor(readonly predicateAddress: string, readonly data: string) {}

  /**
   * @returns the abi encoded StateObject.
   */
  get encoded(): string {
    return abi.encode(AbiStateObject.abiTypes, [
      this.predicateAddress,
      this.data,
    ])
  }

  /**
   * @returns the jsonified AbiStateObject.
   */
  get jsonified(): any {
    return {
      predicateAddress: this.predicateAddress,
      data: this.data,
    }
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
