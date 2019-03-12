import BigNum from 'bn.js'

import { Exit } from '../chain'
import { EthereumEvent } from '../eth'

interface ExitStartedEventArgs {
  token: BigNum
  start: BigNum
  end: BigNum
  id: BigNum
  block: BigNum
  owner: string
}

export class ExitStartedEvent {
  /**
   * Creates an ExitStartedEvent from an EthereumEvent.
   * @param event The EthereumEvent to cast.
   * @returns the ExitStartedEvent object.
   */
  public static fromEthereumEvent(event: EthereumEvent): ExitStartedEvent {
    return new ExitStartedEvent({
      block: event.data.eventBlockNumber as BigNum,
      end: event.data.untypedEnd as BigNum,
      id: event.data.exitID as BigNum,
      owner: event.data.exiter as string,
      start: event.data.untypedStart as BigNum,
      token: event.data.tokenType as BigNum,
    })
  }

  /**
   * Creates an ExitStartedEvent from some arguments.
   * @param args The arguments to cast.
   * @returns the ExitStartedEvent object.
   */
  public static from(args: EthereumEvent): ExitStartedEvent {
    if (args instanceof EthereumEvent) {
      return ExitStartedEvent.fromEthereumEvent(args)
    }

    throw new Error('Cannot cast to ExitStartedEvent.')
  }

  public token: BigNum
  public start: BigNum
  public end: BigNum
  public id: BigNum
  public block: BigNum
  public owner: string

  constructor(event: ExitStartedEventArgs) {
    this.token = event.token
    this.start = event.start
    this.end = event.end
    this.id = event.id
    this.block = event.block
    this.owner = event.owner
  }

  /**
   * Converts the event to an exit object.
   * @returns the exit object.
   */
  public toExit(): Exit {
    return new Exit({
      block: this.block,
      end: this.end,
      id: this.id,
      owner: this.owner,
      start: this.start,
      token: this.token,
    })
  }
}
