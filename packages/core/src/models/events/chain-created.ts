import { hexToAscii } from 'web3-utils'
import { EventLog } from 'web3-core/types'

import { EthereumEvent, isEventLog } from '../eth'

interface ChainCreatedEventArgs {
  plasmaChainAddress: string
  plasmaChainName: string
  operatorEndpoint: string
  operatorAddress: string
}

export class ChainCreatedEvent {
  /**
   * Creates a ChainCreatedEvent from an EthereumEvent.
   * @param event The EthereumEvent to cast.
   * @returns the ChainCreatedEvent object.
   */
  public static fromEthereumEvent(event: EthereumEvent): ChainCreatedEvent {
    return new ChainCreatedEvent({
      operatorAddress: event.raw.OperatorAddress as string,
      operatorEndpoint: encodeURI(
        hexToAscii(event.raw.OperatorEndpoint as string)
      ).replace(/%00/gi, ''),
      plasmaChainAddress: event.raw.PlasmaChainAddress as string,
      plasmaChainName: hexToAscii(event.raw.PlasmaChainName as string),
    })
  }

  /**
   * Creates a ChainCreatedEvent from an EventLog.
   * @param event The EventLog to cast.
   * @returns the ChainCreatedEvent object.
   */
  public static fromEventLog(event: EventLog): ChainCreatedEvent {
    const ethereumEvent = EthereumEvent.from(event)
    return ChainCreatedEvent.from(ethereumEvent)
  }

  /**
   * Creates a ChainCreatedEvent from some arguments.
   * @param args The arguments to cast.
   * @returns the ChainCreatedEvent object.
   */
  public static from(args: EthereumEvent | EventLog): ChainCreatedEvent {
    if (args instanceof EthereumEvent) {
      return ChainCreatedEvent.fromEthereumEvent(args)
    } else if (isEventLog(args)) {
      return ChainCreatedEvent.fromEventLog(args)
    }

    throw new Error('Cannot cast to ChainCreatedEvent.')
  }

  public plasmaChainAddress: string
  public plasmaChainName: string
  public operatorEndpoint: string
  public operatorAddress: string

  constructor(event: ChainCreatedEventArgs) {
    this.plasmaChainAddress = event.plasmaChainAddress
    this.plasmaChainName = event.plasmaChainName
    this.operatorEndpoint = event.operatorEndpoint
    this.operatorAddress = event.operatorAddress
  }
}
