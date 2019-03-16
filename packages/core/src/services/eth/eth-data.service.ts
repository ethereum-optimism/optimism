/* External Imports */
import { Service } from '@nestd/core'
import BigNum = require('bn.js')
import { FullEventFilter, EventLog } from 'watch-eth'
import Web3 from 'web3'

/* Services */
import { Web3Service } from './web3.service'

/**
 * Service used for interacting with Ethereum.
 * Does *not* handle plasma chain contract
 * related calls.
 */
@Service()
export class EthDataService {
  constructor(private readonly web3Service: Web3Service) {}

  /**
   * @returns the current Web3 instance.
   */
  get web3(): Web3 {
    return this.web3Service.web3
  }

  /**
   * @returns `true` if connected to Ethereum, `false` otherwise.
   */
  public async connected(): Promise<boolean> {
    return this.web3Service.connected()
  }

  /**
   * Returns the current ETH balance of an address.
   * Queries the main chain, *not* the plasma chain.
   * @param address Address to query.
   * @returns The account's ETH balance.
   */
  public async getBalance(address: string): Promise<BigNum> {
    const balance = await this.web3.eth.getBalance(address)
    return new BigNum(balance, 10)
  }

  /**
   * @returns The current ETH block.
   */
  public async getCurrentBlock(): Promise<number> {
    return this.web3.eth.getBlockNumber()
  }

  /**
   * Returns the bytecode for the contract at the given address
   * @param address Contract address.
   * @returns the contract's bytecode.
   */
  public async getContractBytecode(address: string): Promise<string> {
    return this.web3.eth.getCode(address)
  }

  /**
   * Queries events with a given filter.
   * @param filter an event filter.
   * @returns all events that match the filter.
   */
  public async getEvents(filter: FullEventFilter): Promise<EventLog[]> {
    const contract = new this.web3.eth.Contract(filter.abi, filter.address)
    const events = await contract.getPastEvents(
      filter.event,
      {
        filter: filter.indexed || {},
        fromBlock: filter.fromBlock,
        toBlock: filter.toBlock,
      },
      null
    )
    return events.map((event) => {
      return new EventLog(event)
    })
  }
}
