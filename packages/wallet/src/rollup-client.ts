/* External Imports */
import {
  DefaultSignatureVerifier,
  getLogger,
  KeyValueStore,
  RpcClient,
  serializeObject,
  SignatureProvider,
  SignatureVerifier,
} from '@pigi/core'

/* Internal Imports */
import {
  Address,
  RollupTransaction,
  UNISWAP_ADDRESS,
  AGGREGATOR_API,
  SignedStateReceipt,
  abiEncodeTransaction,
  AGGREGATOR_ADDRESS,
  SignatureError,
  abiEncodeStateReceipt,
  EMPTY_AGGREGATOR_SIGNATURE,
  NON_EXISTENT_LEAF_ID,
} from './index'

const log = getLogger('rollup-client')

/**
 * Simple Rollup Client enabling getting balances & sending transactions.
 */
export class RollupClient {
  public rpcClient: RpcClient

  /**
   * Initializes the RollupClient
   * @param db the KeyValueStore used by the Rollup Client. Note this is mocked
   *           and so we don't currently use the DB.
   * @param signatureProvider The signer for this client
   * @param signatureVerifier The signature verifier for this client, able to verify
   * response signatures
   */
  constructor(
    private readonly db: KeyValueStore,
    private readonly signatureProvider: SignatureProvider,
    private readonly signatureVerifier: SignatureVerifier = DefaultSignatureVerifier.instance()
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
   * Gets the state for the provided account.
   * @param account the account whose state will be retrieved.
   * @returns the SignedStateReceipt for the account
   */
  public async getState(account: Address): Promise<SignedStateReceipt> {
    const receipt: SignedStateReceipt = await this.rpcClient.handle<
      SignedStateReceipt
    >(AGGREGATOR_API.getState, account)
    this.verifyTransactionReceipt(receipt)
    return receipt
  }

  public async getUniswapBalances(): Promise<SignedStateReceipt> {
    return this.getState(UNISWAP_ADDRESS)
  }

  public async sendTransaction(
    transaction: RollupTransaction,
    account: Address
  ): Promise<SignedStateReceipt[]> {
    const signature = await this.signatureProvider.sign(
      account,
      abiEncodeTransaction(transaction)
    )
    const receipts: SignedStateReceipt[] = await this.rpcClient.handle<
      SignedStateReceipt[]
    >(AGGREGATOR_API.applyTransaction, {
      signature,
      transaction,
    })

    return receipts
  }

  public async requestFaucetFunds(
    transaction: RollupTransaction,
    account: Address
  ): Promise<SignedStateReceipt> {
    const signature = await this.signatureProvider.sign(
      account,
      serializeObject(transaction)
    )
    const receipt: SignedStateReceipt = await this.rpcClient.handle<
      SignedStateReceipt
    >(AGGREGATOR_API.requestFaucetFunds, {
      signature,
      transaction,
    })
    this.verifyTransactionReceipt(receipt)
    return receipt
  }

  private verifyTransactionReceipt(receipt: SignedStateReceipt): void {
    if (
      receipt.signature === EMPTY_AGGREGATOR_SIGNATURE &&
      receipt.stateReceipt.slotIndex === NON_EXISTENT_LEAF_ID
    ) {
      return
    }

    const signer = this.signatureVerifier.verifyMessage(
      abiEncodeStateReceipt(receipt.stateReceipt),
      receipt.signature
    )
    if (signer !== AGGREGATOR_ADDRESS) {
      log.error(
        `Received invalid SignedStateReceipt from the Aggregator: ${serializeObject(
          receipt
        )}`
      )
      throw new SignatureError()
    }
  }
}
