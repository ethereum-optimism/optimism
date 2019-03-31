import BigNum = require('bn.js')
import Web3 from 'web3'

import {
  EthClient,
  TransactionReceipt,
  EventFilter,
  EventLog,
} from '../../../interfaces'

/**
 * Simple EthClient implementation that uses Web3 over HTTP under the hood.
 */
export class DefaultEthClient implements EthClient {
  private web3: Web3

  constructor(endpoint = 'http://127.0.0.1:8545') {
    this.web3 = new Web3(endpoint)
  }

  /**
   * @returns `true` if connected via web3, `false` otherwise.
   */
  public async connected(): Promise<boolean> {
    try {
      await this.web3.eth.net.isListening()
      return true
    } catch (e) {
      return false
    }
  }

  /**
   * @returns the current Ethereum block number.
   */
  public async getBlockNumber(): Promise<number> {
    return this.web3.eth.getBlockNumber()
  }

  /**
   * Queries the balance of an address.
   * @param address Address to query.
   * @returns the balance of the address.
   */
  public async getBalance(address: string): Promise<BigNum> {
    const balance = await this.web3.eth.getBalance(address)
    return new BigNum(balance, 10)
  }

  /**
   * Queries the code at an address.
   * @param address Address to query.
   * @returns the code at that address.
   */
  public async getCode(address: string): Promise<string> {
    return this.web3.eth.getCode(address)
  }

  /**
   * Queries events that match some filter.
   * @param filter Filter to match.
   * @returns a list of events that match the filter.
   */
  public async getEvents(filter: EventFilter): Promise<EventLog[]> {
    const contract = new this.web3.eth.Contract(filter.abi, filter.address)
    return contract.getPastEvents(
      filter.event,
      {
        filter: filter.indexed || {},
        fromBlock: filter.fromBlock,
        toBlock: filter.toBlock,
      },
      null
    )
  }

  /**
   * Sends a signed transaction to the network.
   * @param transaction Signed transaciton to send.
   * @returns the transaction receipt.
   */
  public async sendSignedTransaction(
    transaction: string | Buffer
  ): Promise<TransactionReceipt> {
    transaction = Buffer.isBuffer(transaction)
      ? transaction.toString('hex')
      : transaction
    return this.web3.eth.sendSignedTransaction(transaction)
  }
}
