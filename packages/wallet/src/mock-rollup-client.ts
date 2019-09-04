/* External Imports */
import { KeyValueStore, RpcClient, serializeObject } from '@pigi/core'

/* Internal Imports */
import {
  Address,
  Balances,
  State,
  Transaction,
  MockedSignature,
  TransactionReceipt,
  UNISWAP_ADDRESS,
  AGGREGATOR_API,
} from '.'

/**
 * Simple Rollup Client enabling getting balances & sending transactions.
 */
export class MockRollupClient {
  public rpcClient: RpcClient
  public uniswapAddress: Address

  /**
   * Initializes the MockRollupClient
   * @param db the KeyValueStore used by the Rollup Client. Note this is mocked
   *           and so we don't currently use the DB.
   * @param sign a function used for signing messages.
   *             TODO: replace sign(...) with a reference to the keystore.
   */
  constructor(
    readonly db: KeyValueStore,
    readonly sign: (address: string, message: string) => Promise<string>
  ) {}

  /**
   * Connects to an aggregator using a provided rpcClient
   * @param rpcClient the rpcClient to use -- normally it's a SimpleClient
   */
  public connect(rpcClient: RpcClient) {
    // Create a new simple JSON rpc server for the rollup client
    this.rpcClient = rpcClient
    // TODO: Persist the aggregator url
  }

  /**
   * Connects to an aggregator using a provided rpcClient
   * @param rpcClient the rpcClient to use -- normally it's a SimpleClient
   */
  public async getBalances(account: Address): Promise<Balances> {
    const balances = await this.rpcClient.handle<Balances>(
      AGGREGATOR_API.getBalances,
      account
    )
    return balances
  }

  public async getUniswapBalances(): Promise<Balances> {
    return this.getBalances(UNISWAP_ADDRESS)
  }

  public async sendTransaction(
    transaction: Transaction,
    account: Address
  ): Promise<State> {
    const signature = await this.sign(account, serializeObject(transaction))
    const result = await this.rpcClient.handle<TransactionReceipt>(
      AGGREGATOR_API.applyTransaction,
      {
        signature,
        transaction,
      }
    )
    return result.stateUpdate
  }

  public async requestFaucetFunds(
    account: Address,
    amount: number
  ): Promise<Balances> {
    const result = await this.rpcClient.handle<Balances>(
      AGGREGATOR_API.requestFaucetFunds,
      [account, amount]
    )
    return result
  }
}
