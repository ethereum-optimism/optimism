/* External Imports */
import {
  KeyValueStore,
  RpcClient,
  serializeObject,
  SignatureProvider,
} from '@pigi/core'

/* Internal Imports */
import {
  Address,
  Balances,
  State,
  Transaction,
  UNISWAP_ADDRESS,
  AGGREGATOR_API,
  SignedTransactionReceipt,
  SignedStateReceipt,
} from './index'

/**
 * Simple Rollup Client enabling getting balances & sending transactions.
 */
export class RollupClient {
  public rpcClient: RpcClient

  /**
   * Initializes the RollupClient
   * @param db the KeyValueStore used by the Rollup Client. Note this is mocked
   *           and so we don't currently use the DB.
   * @param signatureProvider
   */
  constructor(
    private readonly db: KeyValueStore,
    private readonly signatureProvider: SignatureProvider
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
  public async getState(account: Address): Promise<SignedStateReceipt> {
    return this.rpcClient.handle<SignedStateReceipt>(
      AGGREGATOR_API.getState,
      account
    )
  }

  public async getUniswapBalances(): Promise<SignedStateReceipt> {
    return this.getState(UNISWAP_ADDRESS)
  }

  public async sendTransaction(
    transaction: Transaction,
    account: Address
  ): Promise<SignedTransactionReceipt> {
    const signature = await this.signatureProvider.sign(
      account,
      serializeObject(transaction)
    )
    return this.rpcClient.handle<SignedTransactionReceipt>(
      AGGREGATOR_API.applyTransaction,
      {
        signature,
        transaction,
      }
    )
  }

  public async requestFaucetFunds(
    transaction: Transaction,
    account: Address
  ): Promise<SignedTransactionReceipt> {
    const signature = await this.signatureProvider.sign(
      account,
      serializeObject(transaction)
    )
    return this.rpcClient.handle<SignedTransactionReceipt>(
      AGGREGATOR_API.requestFaucetFunds,
      {
        signature,
        transaction,
      }
    )
  }
}
