/* External Imports */
import BigNum = require('bn.js')
import { StateUpdate, StateObject } from '@pigi/utils'

/* Internal Imports */
import { EthereumEvent } from '../eth'

/**
 * Represents a plasma chain deposit.
 */
export class Deposit extends StateUpdate {
  /**
   * Creates a DepositEvent from an EthereumEvent.
   * @param event The EthereumEvent to cast.
   * @returns the DepositEvent object.
   */
  public static fromEthereumEvent(event: EthereumEvent): Deposit {
    return new Deposit({
      start: event.data.untypedStart as BigNum,
      end: event.data.untypedEnd as BigNum,
      block: event.data.plasmaBlockNumber as BigNum,
      plasmaContract: event.data.plasmaContract as string,
      newState: StateObject.from(event.data.newState as string),
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
