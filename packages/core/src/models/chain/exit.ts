/* External Imports */
import BigNum from 'bn.js'
import { StateObject, StateObjectData } from '@pigi/utils'

/* Internal Imports */
import { EthereumEvent } from '../eth'

export interface ExitArgs extends StateObjectData {
  owner: string
  id: string
}

/**
 * Represents a plasma chain exit.
 */
export class Exit extends StateObject {
  public readonly owner: string
  public readonly id: string
  public completed?: boolean
  public finalized?: boolean

  constructor(args: ExitArgs) {
    super({
      ...args,
      predicate: null,
      state: null,
    })

    this.owner = args.owner
    this.id = args.id
  }

  /**
   * Creates an Exit from an EthereumEvent.
   * @param event The EthereumEvent to cast.
   * @returns the Exit object.
   */
  public static fromEthereumEvent(event: EthereumEvent): Exit {
    return new Exit({
      owner: event.data.exiter as string,
      start: event.data.untypedStart as BigNum,
      end: event.data.untypedEnd as BigNum,
      block: event.data.eventBlockNumber as BigNum,
      id: event.data.exitID as string,
      predicate: null,
      state: null,
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
