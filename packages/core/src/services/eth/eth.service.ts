/* External Imports */
import { Service, OnStart } from '@nestd/core'
import BigNum from 'bn.js'
import { isString } from 'util'
import { FullEventFilter, EventLog } from 'watch-eth'
import Web3 from 'web3'

/* Internal Imports */
import { EthereumAccount, isAccount } from '../../models/eth'
import { ConfigService } from '../config.service'
import { CONFIG } from '../../constants'

@Service()
export class EthService implements OnStart {
  private web3: Web3

  constructor(private readonly config: ConfigService) {}

  public async onStart(): Promise<void> {
    this.web3 = new Web3(
      new Web3.providers.HttpProvider(this.ethereumEndpoint())
    )
  }

  /**
   * @returns `true` if the node is connected to Ethereum, `false` otherwise.
   */
  public async connected(): Promise<boolean> {
    if (!this.web3) {
      return false
    }

    try {
      await this.web3.eth.net.isListening()
      return true
    } catch (e) {
      return false
    }
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
   * Returns the addresses of all exposed web3 accounts.
   * @returns the list of addresses.
   */
  public async getAccounts(): Promise<string[]> {
    return this.web3.eth.getAccounts()
  }

  /**
   * Signs some data with the given address.
   * @param address Address to sign with.
   * @param data Data to sign.
   * @returns the signed address.
   */
  public async sign(address: string, data: string): Promise<string> {
    return this.web3.eth.sign(data, address)
  }

  /**
   * @returns the list of address in the user's wallet.
   */
  public async getWalletAccounts(): Promise<string[]> {
    const wallet = this.web3.eth.accounts.wallet
    const keys = Object.keys(wallet)
    return keys.filter((key) => {
      return this.web3.utils.isAddress(key)
    })
  }

  /**
   * Returns the account object for a given account.
   * @param address Address of the account.
   * @returns the account object.
   */
  public async getWalletAccount(address: string): Promise<EthereumAccount> {
    const wallet: { [key: string]: string | {} } = this.web3.eth.accounts.wallet
    for (const key of Object.keys(wallet)) {
      const value = wallet[key]
      if (key === address && !isString(value) && isAccount(value)) {
        return value as EthereumAccount
      }
    }

    throw new Error('Account not found.')
  }

  /**
   * Checks if the wallet has the given account.
   * @param address Address to check.
   * @returns `true` if the wallet has account, `false` otherwise.
   */
  public async hasWalletAccount(address: string): Promise<boolean> {
    const accounts = await this.getWalletAccounts()
    return accounts.includes(address)
  }

  /**
   * Adds an account to the user's wallet.
   * @param privateKey the account's private key.
   */
  public async addWalletAccount(privateKey: string): Promise<void> {
    await this.web3.eth.accounts.wallet.add(privateKey)
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
    const events = await contract.getPastEvents(filter.event, {
      ...(filter.indexed || {}),
      fromBlock: filter.fromBlock,
      toBlock: filter.toBlock,
    })
    return events.map((event) => {
      return new EventLog(event)
    })
  }

  /**
   * @returns the current Ethereum endpoint.
   */
  private ethereumEndpoint(): string {
    return this.config.get(CONFIG.ETHEREUM_ENDPOINT)
  }
}
