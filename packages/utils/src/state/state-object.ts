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
}
