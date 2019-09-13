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
  TransactionReceipt,
  UNISWAP_ADDRESS,
  AGGREGATOR_API,
  SignedTransactionReceipt,
} from '../index'

/**
 * Simple Rollup Client enabling getting balances & sending transactions.
 */
export class MockRollupClient {
  public rpcClient: RpcClient

  /**
   * Initializes the MockRollupClient
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
    const signature = await this.signatureProvider.sign(
      account,
      serializeObject(transaction)
    )
    const result: SignedTransactionReceipt = await this.rpcClient.handle<
      SignedTransactionReceipt
    >(AGGREGATOR_API.applyTransaction, {
      signature,
      transaction,
    })
    // TODO: Probably want too check aggregator sig and store some stuff
    return result.transactionReceipt.updatedState
  }

  public async requestFaucetFunds(
    transaction: Transaction,
    account: Address
  ): Promise<Balances> {
    const signature = await this.signatureProvider.sign(
      account,
      serializeObject(transaction)
    )
    const result = await this.rpcClient.handle<SignedTransactionReceipt>(
      AGGREGATOR_API.requestFaucetFunds,
      {
        signature,
        transaction,
      }
    )
    // TODO: Probably want too check aggregator sig and store some stuff
    return result.transactionReceipt.updatedState[account].balances
  }
}
