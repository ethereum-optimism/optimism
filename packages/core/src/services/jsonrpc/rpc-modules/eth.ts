/* External Imports */
import { Service } from '@nestd/core'
import BigNum from 'bn.js'

/* Services */
import { EthDataService } from '../../eth/eth-data.service'
import { ContractService } from '../../eth/contract.service'

/* Internal Imports */
import { BaseRpcModule } from './base-rpc-module'
import { EthereumTransactionReceipt } from '../../../models/eth'

/**
 * Subdispatcher that handles Ethereum-related requests.
 */
@Service()
export class EthRpcModule extends BaseRpcModule {
  public readonly prefix = 'pg_'

  constructor(
    private readonly eth: EthDataService,
    private readonly contract: ContractService
  ) {
    super()
  }

  /**
   * Submits a deposit for a given user.
   * User's account must be unlocked.
   * @param token Token to deposit.
   * @param amount Amount of the token to deposit.
   * @param owner Address to deposit from.
   * @returns the Ethereum transaction receipt of the deposit.
   */
  public async deposit(
    token: string,
    amount: string,
    owner: string
  ): Promise<EthereumTransactionReceipt> {
    return this.contract.deposit(
      new BigNum(token, 'hex'),
      new BigNum(amount, 'hex'),
      owner
    )
  }

  /**
   * @returns the current plasma block number.
   */
  public async getCurrentPlasmaBlock(): Promise<number> {
    return this.contract.getCurrentBlock()
  }

  /**
   * Queries the token ID for a given token contract address
   * @param tokenAddress Token contract address.
   * @returns the token ID for that address.
   */
  public async getTokenId(tokenAddress: string): Promise<string> {
    return this.contract.getTokenId(tokenAddress)
  }

  /**
   * Lists a token so that it can be deposited.
   * Tokens must be listed before they can be
   * deposited.
   * @param tokenAddress Address of the token contract to list.
   * @param [sender] Sender to use for the transaction. Defaults to first unlocked account.
   * @returns the Ethereum transaction receipt for the listing.
   */
  public async listToken(
    tokenAddress: string,
    sender?: string
  ): Promise<EthereumTransactionReceipt> {
    return this.contract.listToken(tokenAddress, sender)
  }

  /**
   * @returns the current *Ethereum* block number.
   */
  public async getCurrentEthBlock(): Promise<number> {
    return this.eth.getCurrentBlock()
  }

  /**
   * Queries the ETH balance for an account.
   * @param address Address to query.
   * @returns the ETH balance for the given account.
   */
  public async getEthBalance(address: string): Promise<BigNum> {
    return this.eth.getBalance(address)
  }
}
