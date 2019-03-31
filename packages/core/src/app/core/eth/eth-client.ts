import BigNum = require('bn.js')
import Web3 from 'web3'

import { EthClient, TransactionReceipt, EventFilter, EventLog } from '../../../interfaces'

export class DefaultEthClient implements EthClient {
  private web3: Web3

  public async connected(): Promise<boolean> {
    try {
      await this.web3.eth.net.isListening()
      return true
    } catch (e) {
      return false
    }
  }

  public async getBlockNumber(): Promise<number> {
    return this.web3.eth.getBlockNumber()
  }

  public async getBalance(address: string): Promise<BigNum> {
    const balance = await this.web3.eth.getBalance(address)
    return new BigNum(balance, 10)
  }

  public async getCode(address: string): Promise<string> {
    return this.web3.eth.getCode(address)
  }

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

  public async sendSignedTransaction(
    transaction: string | Buffer
  ): Promise<TransactionReceipt> {
    transaction = Buffer.isBuffer(transaction)
      ? transaction.toString('hex')
      : transaction
    return this.web3.eth.sendSignedTransaction(transaction)
  }
}
