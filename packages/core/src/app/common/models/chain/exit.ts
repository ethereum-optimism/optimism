/* External Imports */
import BigNum = require('bn.js')
import { StateUpdate, StateObject } from '@pigi/utils'

/* Internal Imports */
import { EthereumEvent } from '../eth'

/**
 * Represents a plasma chain exit.
 */
export class Exit extends StateUpdate {
  /**
   * Creates an Exit from an EthereumEvent.
   * @param event The EthereumEvent to cast.
   * @returns the Exit object.
   */
  public static fromEthereumEvent(event: EthereumEvent): Exit {
    return new Exit({
      start: event.data.untypedStart as BigNum,
      end: event.data.untypedEnd as BigNum,
      block: event.data.eventBlockNumber as BigNum,
      plasmaContract: event.data.plasmaContract as string,
      newState: new StateObject({
        predicate: null,
        parameters: null,
      }),
    })
  }

  /**
   * Creates an Exit from some arguments.
   * @param args The arguments to cast.
   * @returns the Exit object.
   */
  public static from(args: EthereumEvent): Exit {
    if (args instanceof EthereumEvent) {
      return Exit.fromEthereumEvent(args)
    }

    throw new Error('Cannot cast to Exit.')
  }
}
