/* External Imports */
import { hexToAscii } from 'web3-utils'

/* Internal Imports */
import { EthereumEvent } from '../eth'

interface PlasmaChainArgs {
  plasmaChainAddress: string
  plasmaChainName: string
  operatorEndpoint: string
  operatorAddress: string
}

/**
 * Represents plasma chain data.
 */
export class PlasmaChain {
  public readonly plasmaChainAddress: string
  public readonly plasmaChainName: string
  public readonly operatorEndpoint: string
  public readonly operatorAddress: string

  constructor(args: PlasmaChainArgs) {
    this.plasmaChainAddress = args.plasmaChainAddress
    this.plasmaChainName = args.plasmaChainName
    this.operatorEndpoint = args.operatorEndpoint
    this.operatorAddress = args.operatorAddress
  }

  /**
   * Creates a PlasmaChain from an EthereumEvent.
   * @param event The EthereumEvent to cast.
   * @returns the PlasmaChain object.
   */
  public static fromEthereumEvent(event: EthereumEvent): PlasmaChain {
    return new PlasmaChain({
      operatorAddress: event.raw.OperatorAddress as string,
      operatorEndpoint: encodeURI(
        hexToAscii(event.raw.OperatorEndpoint as string)
      ).replace(/%00/gi, ''),
      plasmaChainAddress: event.raw.PlasmaChainAddress as string,
      plasmaChainName: hexToAscii(event.raw.PlasmaChainName as string),
    })
  }

  /**
   * Creates a PlasmaChain from some arguments.
   * @param args The arguments to cast.
   * @returns the PlasmaChain object.
   */
  public static from(args: EthereumEvent): PlasmaChain {
    if (args instanceof EthereumEvent) {
      return PlasmaChain.fromEthereumEvent(args)
    }

    throw new Error('Cannot cast to PlasmaChain.')
  }
}
