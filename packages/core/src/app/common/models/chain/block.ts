/* Internal Imports */
import { EthereumEvent } from '../eth'

export interface BlockArgs {
  number: number
  hash: string
}

/**
 * Represents a plasma chain block.
 */
export class PlasmaBlock {
  public readonly number: number
  public readonly hash: string

  constructor(args: BlockArgs) {
    this.number = args.number
    this.hash = args.hash
  }

  /**
   * Creates a PlasmaBlock from an EthereumEvent.
   * @param event The EthereumEvent to cast.
   * @returns the PlasmaBlock object.
   */
  public static fromEthereumEvent(event: EthereumEvent): PlasmaBlock {
    return new PlasmaBlock({
      hash: event.raw.submittedHash as string,
      number: event.block.toNumber(),
    })
  }

  /**
   * Creates a PlasmaBlock from some arguments.
   * @param args The arguments to cast.
   * @returns the PlasmaBlock object.
   */
  public static from(args: EthereumEvent): PlasmaBlock {
    if (args instanceof EthereumEvent) {
      return PlasmaBlock.fromEthereumEvent(args)
    }

    throw new Error('Cannot cast to Block.')
  }
}
