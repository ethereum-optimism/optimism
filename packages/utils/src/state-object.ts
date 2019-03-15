import BigNum from 'bn.js'
import { abi } from './utils'

const STATE_OBJECT_ABI_TYPES = [
  'uint256',
  'uint256',
  'uint256',
  'address',
  'bytes',
]

export interface StateObjectData {
  start: BigNum
  end: BigNum
  block: BigNum
  predicate: string
  state: string

  implicit?: boolean
  implicitStart?: BigNum
  implicitEnd?: BigNum
}

/**
 * Class that represents a simple state object.
 * State objects are the fundamental building
 * blocks of our plasma chain design.
 */
export class StateObject {
  public start: BigNum
  public end: BigNum
  public block: BigNum
  public predicate: string
  public state: string

  public implicit?: boolean
  public implicitStart?: BigNum
  public implicitEnd?: BigNum

  constructor(args: StateObjectData) {
    this.start = args.start
    this.end = args.end
    this.block = args.block
    this.predicate = args.predicate
    this.state = args.state
  }

  /**
   * @returns the encoded state object.
   */
  get encoded(): string {
    return abi.encode(STATE_OBJECT_ABI_TYPES, [
      this.start,
      this.end,
      this.block,
      this.predicate,
      this.state,
    ])
  }

  /**
   * Determines if this object equals another.
   * @param other Object to compare to.
   * @returns `true` if the two are equal, `false` otherwise.
   */
  public equals(other: StateObject): boolean {
    return (
      this.start.eq(other.start) &&
      this.end.eq(other.end) &&
      this.block.eq(other.block) &&
      this.predicate === other.predicate &&
      this.state === other.state
    )
  }

  /**
   * Breaks a StateObject into the implicit and
   * explicit components that make it up.
   * @param stateObject Object to break down
   * @returns a list of StateObjects.
   */
  public components(): StateObject[] {
    const components = []

    if (this.implicitStart === undefined || this.implicitEnd === undefined) {
      return [this]
    }

    // Left implicit component.
    if (!this.start.eq(this.implicitStart)) {
      components.push(
        new StateObject({
          ...this,
          ...{
            end: this.start,
            start: this.implicitStart,

            implicit: true,
          },
        })
      )
    }

    // Right implicit component.
    if (!this.end.eq(this.implicitEnd)) {
      components.push(
        new StateObject({
          ...this,
          ...{
            end: this.implicitEnd,
            start: this.end,

            implicit: true,
          },
        })
      )
    }

    // Explicit component.
    if (this.start.lt(this.end)) {
      components.push(
        new StateObject({
          ...this,
          ...{
            end: this.end,
            start: this.start,
          },
        })
      )
    }

    return components
  }
}
