import BigNum from 'bn.js'

import { Deposit } from '../chain'
import { EthereumEvent } from '../eth'

interface DepositEventArgs {
  owner: string
  start: BigNum
  end: BigNum
  token: BigNum
  block: BigNum
}

export class DepositEvent {
  /**
   * Creates a DepositEvent from an EthereumEvent.
   * @param event The EthereumEvent to cast.
   * @returns the DepositEvent object.
   */
  public static fromEthereumEvent(event: EthereumEvent): DepositEvent {
    return new DepositEvent({
      block: event.data.plasmaBlockNumber as BigNum,
      end: event.data.untypedEnd as BigNum,
      owner: event.data.depositer as string,
      start: event.data.untypedStart as BigNum,
      token: event.data.tokenType as BigNum,
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

  public owner: string
  public start: BigNum
  public end: BigNum
  public token: BigNum
  public block: BigNum

  constructor(event: DepositEventArgs) {
    this.owner = event.owner
    this.start = event.start
    this.end = event.end
    this.token = event.token
    this.block = event.block
  }

  /**
   * @returns the total amount deposited.
   */
  get amount(): BigNum {
    return this.end.sub(this.start)
  }

  /**
   * Converts the deposit event to a deposit object.
   * @returns the deposit object.
   */
  public toDeposit(): Deposit {
    return new Deposit({
      block: this.block,
      end: this.end,
      owner: this.owner,
      start: this.start,
      token: this.token,
    })
  }
}
