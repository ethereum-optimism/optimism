import BigNum = require('bn.js')

import { TransactionReceipt, EventFilter, EventLog } from '../../common'

/**
 * EthClient exposes an API for interacting with Ethereum.
 */
export interface EthClient {
  /**
   * @returns `true` if connected to the node, `false` otherwise.
   */
  connected(): Promise<boolean>

  /**
   * @returns the current Ethereum block number.
   */
  getBlockNumber(): Promise<number>

  /**
   * Queries the balance of an address.
   * @param address Address to query.
   * @returns the balance of that address.
   */
  getBalance(address: string): Promise<BigNum>

  /**
   * Pulls the code at a given address.
   * @param address Address to query.
   * @returns the code at that address.
   */
  getCode(address: string): Promise<string>

  /**
   * Queries events that match a filter.
   * @param filter Filter to match.
   * @returns a list of events that match the filter.
   */
  getEvents(filter: EventFilter): Promise<EventLog[]>

  /**
   * Sends a signed transaction to the network.
   * @param transaction Signed transaction to send.
   * @returns a transaction receipt.
   */
  sendSignedTransaction(
    transaction: string | Buffer
  ): Promise<TransactionReceipt>
}
