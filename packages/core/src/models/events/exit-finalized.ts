import BigNum from 'bn.js'
import { EthereumEvent } from '../eth'

interface ExitFinalizedEventArgs {
  token: BigNum
  start: BigNum
  end: BigNum
  id: BigNum
}

export class ExitFinalizedEvent {
  /**
   * Creates a ExitFinalizedEvent from an EthereumEvent.
   * @param event The EthereumEvent to cast.
   * @returns the ExitFinalizedEvent object.
   */
  public static fromEthereumEvent(event: EthereumEvent): ExitFinalizedEvent {
    return new ExitFinalizedEvent({
      end: event.data.untypedEnd as BigNum,
      id: event.data.exitID as BigNum,
      start: event.data.untypedStart as BigNum,
      token: event.data.tokenType as BigNum,
    })
  }

  /**
   * Creates a ExitFinalizedEvent from some arguments.
   * @param args The arguments to cast.
   * @returns the ExitFinalizedEvent object.
   */
  public static from(args: EthereumEvent): ExitFinalizedEvent {
    if (args instanceof EthereumEvent) {
      return ExitFinalizedEvent.fromEthereumEvent(args)
    }

    throw new Error('Cannot cast to ExitFinalizedEvent.')
  }

  public token: BigNum
  public start: BigNum
  public end: BigNum
  public id: BigNum

  constructor(event: ExitFinalizedEventArgs) {
    this.token = event.token
    this.start = event.start
    this.end = event.end
    this.id = event.id
  }
}
