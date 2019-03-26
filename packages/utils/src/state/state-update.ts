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

interface StateUpdateArgs {
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
}
