import { Block } from '../chain'
import { EthereumEvent } from '../eth'

interface BlockSubmittedEventArgs {
  number: number
  hash: string
}

export class BlockSubmittedEvent {
  /**
   * Creates a BlockSubmittedEvent from an EthereumEvent.
   * @param event The EthereumEvent to cast.
   * @returns the BlockSubmittedEvent object.
   */
  public static fromEthereumEvent(event: EthereumEvent): BlockSubmittedEvent {
    return new BlockSubmittedEvent({
      hash: event.raw.submittedHash as string,
      number: event.block.toNumber(),
    })
  }

  /**
   * Creates a BlockSubmittedEvent from some arguments.
   * @param args The arguments to cast.
   * @returns the BlockSubmittedEvent object.
   */
  public static from(args: EthereumEvent): BlockSubmittedEvent {
    if (args instanceof EthereumEvent) {
      return BlockSubmittedEvent.fromEthereumEvent(args)
    }

    throw new Error('Cannot cast to BlockSubmittedEvent.')
  }

  public number: number
  public hash: string

  constructor(event: BlockSubmittedEventArgs) {
    this.number = event.number
    this.hash = event.hash
  }

  public toBlock(): Block {
    return { number: this.number, hash: this.hash }
  }
}
