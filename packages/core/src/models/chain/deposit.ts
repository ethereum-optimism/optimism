/* External Imports */
import BigNum = require('bn.js')
import { StateObject, StateObjectData } from '@pigi/utils'

/* Internal Imports */
import { EthereumEvent } from '../eth'

export interface DepositArgs extends StateObjectData {
  owner: string
}

/**
 * Represents a plasma chain deposit.
 */
export class Deposit extends StateObject {
  public readonly owner: string

  constructor(args: DepositArgs) {
    super(args)

    this.owner = args.owner
  }

  /**
   * Checks if this deposit equals some other deposit.
   * @param other Other deposit to check against.
   * @returns `true` if this deposit equals the other, `false` otherwise.
   */
  public equals(other: Deposit): boolean {
    return (
      this.owner === other.owner &&
      this.state === other.state &&
      this.predicate === other.predicate &&
      this.start.eq(other.start) &&
      this.end.eq(other.end) &&
      this.block.eq(other.block)
    )
  }

  /**
   * Creates a DepositEvent from an EthereumEvent.
   * @param event The EthereumEvent to cast.
   * @returns the DepositEvent object.
   */
  public static fromEthereumEvent(event: EthereumEvent): Deposit {
    return new Deposit({
      owner: event.data.depositer as string,
      start: event.data.untypedStart as BigNum,
      end: event.data.untypedEnd as BigNum,
      block: event.data.plasmaBlockNumber as BigNum,
      predicate: null,
      state: null,
    })
  }

  /**
   * Creates a DepositEvent from some arguments.
   * @param args The arguments to cast.
   * @returns the DepositEvent object.
   */
  public static from(args: EthereumEvent): Deposit {
    if (args instanceof EthereumEvent) {
      return Deposit.fromEthereumEvent(args)
    }

    throw new Error('Cannot cast to Deposit.')
  }
}
