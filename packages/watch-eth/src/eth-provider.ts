/* External Imports */
import Web3 from 'web3'

/* Internal Imports */
import { EthProvider, FullEventFilter, EventLog } from './interfaces'
import { DefaultEventLog } from './models'

/**
 * Basic EthProvider implementation.
 */
export class DefaultEthProvider implements EthProvider {
  constructor(private web3: Web3) {}

  /**
   * Checks whether the web3 node is connected.
   * @returns `true` if the node is connected, `false` otherwise.
   */
  public async connected(): Promise<boolean> {
    try {
      await this.web3.eth.net.isListening()
      return true
    } catch {
      return false
    }
  }

  /**
   * @returns the current Ethereum block number.
   */
  public async getCurrentBlock(): Promise<number> {
    return this.web3.eth.getBlockNumber()
  }

  /**
   * Returns a list of events that match a given filter.
   * @param filter Filter to match.
   * @returns all events that match the filter.
   */
  public async getEvents(filter: FullEventFilter): Promise<EventLog[]> {
    const contract = new this.web3.eth.Contract(filter.abi, filter.address)
    const events = await contract.getPastEvents(filter.event, {
      ...(filter.indexed || {}),
      fromBlock: filter.fromBlock,
      toBlock: filter.toBlock,
    } as any)
    return events.map((event) => {
      return new DefaultEventLog(event)
    })
  }
}
