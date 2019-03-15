import BigNum from 'bn.js'

import { Deposit } from '../chain'
import { EthereumEvent } from '../eth'

export class DepositEvent extends Deposit {
  /**
   * @returns the total amount deposited.
   */
  get amount(): BigNum {
    return this.end.sub(this.start)
  }

  /**
   * Creates a DepositEvent from an EthereumEvent.
   * @param event The EthereumEvent to cast.
   * @returns the DepositEvent object.
   */
  public static fromEthereumEvent(event: EthereumEvent): DepositEvent {
    return new DepositEvent({
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
  public static from(args: EthereumEvent): DepositEvent {
    if (args instanceof EthereumEvent) {
      return DepositEvent.fromEthereumEvent(args)
    }

    throw new Error('Cannot cast to DepositEvent.')
  }
}
