/* Internal Imports */
import { abi } from '../utils'

const STATE_OBJECT_ABI_TYPES = ['address', 'bytes']

export interface StateObjectData {
  predicate: string
  parameters: string
}

/**
 * Class that represents a simple state object.
 * State objects are the fundamental building
 * blocks of our plasma chain design.
 */
export class StateObject {
  public predicate: string
  public parameters: string

  constructor(args: StateObjectData) {
    this.predicate = args.predicate
    this.parameters = args.parameters
  }

  /**
   * @returns the encoded state object.
   */
  get encoded(): string {
    return abi.encode(STATE_OBJECT_ABI_TYPES, [this.predicate, this.parameters])
  }

  /**
   * Creates a StateObject from its encoded form.
   * @param encoded The encoded StateObject.
   * @returns the StateObject.
   */
  public static fromEncoded(encoded: string): StateObject {
    const decoded = abi.decode(STATE_OBJECT_ABI_TYPES, encoded)
    return new StateObject({
      predicate: decoded[0],
      parameters: decoded[1],
    })
  }

  /**
   * Creates a StateObject from some arguments.
   * @param args Arguments to cast.
   * @returns the StateObject.
   */
  public static from(args: string): StateObject {
    if (typeof args === 'string') {
      return StateObject.fromEncoded(args)
    }

    throw new Error('Cannot cast to StateObject.')
  }
}
