/* External Imports */
import BigNum = require('bn.js')

/* Internal Imports */
import { abi } from '../utils'
import { StateObject } from './state-object'

const STATE_OBJECT_ABI_TYPES = [
  'uint256',
  'uint256',
  'uint256',
  'bytes',
  'bytes',
]

export interface StateUpdateArgs {
  start: number | BigNum
  end: number | BigNum
  block: number | BigNum
  plasmaContract: string
  newState: StateObject
}

/**
 * Represents a StateUpdate, which wraps each state
 * update but doesn't have a witness.
 */
export class StateUpdate {
  public start: BigNum
  public end: BigNum
  public block: BigNum
  public plasmaContract: string
  public newState: StateObject

  public implicit?: boolean
  public implicitStart?: BigNum
  public implicitEnd?: BigNum

  constructor(args: StateUpdateArgs) {
    this.start = new BigNum(args.start, 'hex')
    this.end = new BigNum(args.end, 'hex')
    this.block = new BigNum(args.block, 'hex')
    this.plasmaContract = args.plasmaContract
    this.newState = args.newState
  }

  /**
   * @returns the encoded state update.
   */
  get encoded(): string {
    return abi.encode(STATE_OBJECT_ABI_TYPES, [
      this.start,
      this.end,
      this.block,
      this.plasmaContract,
      this.newState.encoded,
    ])
  }

  /**
   * Determines if this object equals another.
   * @param other Object to compare to.
   * @returns `true` if the two are equal, `false` otherwise.
   */
  public equals(other: StateUpdate): boolean {
    return this.encoded === other.encoded
  }

  /**
   * Breaks a StateUpdate into the implicit and
   * explicit components that make it up.
   * @param stateUpdate Object to break down
   * @returns a list of StateUpdates.
   */
  public components(): StateUpdate[] {
    const components = []

    if (this.implicitStart === undefined || this.implicitEnd === undefined) {
      return [this]
    }

    // Left implicit component.
    if (!this.start.eq(this.implicitStart)) {
      components.push(
        new StateUpdate({
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
        new StateUpdate({
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
        new StateUpdate({
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
