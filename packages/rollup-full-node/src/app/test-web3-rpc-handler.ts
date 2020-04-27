/* External Imports */
import {
  getLogger,
  numberToHexString,
  castToNumber,
  rlpEncodeTransactionWithRandomSig,
} from '@eth-optimism/core-utils'
import { GAS_LIMIT } from '@eth-optimism/ovm'
import { JsonRpcProvider } from 'ethers/providers'

/* Internal Imports */
import { DefaultWeb3Handler, latestBlock } from './web3-rpc-handler'
import {
  L2NodeContext,
  L2ToL1MessageSubmitter,
  UnsupportedMethodError,
  Web3RpcMethods,
} from '../types'
import { getCurrentTime, initializeL2Node } from './util'
import { NoOpL2ToL1MessageSubmitter } from './message-submitter'

const log = getLogger('test-web3-handler')

/**
 * Test Handler that provides extra functionality for testing.
 */
export class TestWeb3Handler extends DefaultWeb3Handler {
  public static readonly successString = 'success'

  private timestampIncreaseSeconds: number = 0
  private timestampIncreaseSnapshots: Object = {}

  /**
   * Creates a local node, deploys the L2ExecutionManager to it, and returns a
   * TestHandler that handles Web3 requests to it.
   *
   * @param messageSubmitter (optional) The L2MessageSubmitter to use.
   * @param provider (optional) The web3 provider to use.
   * @param l2NodeContext (optional) The L2NodeContext to use.
   * @returns The constructed Web3 handler.
   */
  public static async create(
    messageSubmitter: L2ToL1MessageSubmitter = new NoOpL2ToL1MessageSubmitter(),
    provider?: JsonRpcProvider,
    l2NodeContext?: L2NodeContext
  ): Promise<TestWeb3Handler> {
    const timestamp = getCurrentTime()
    const context: L2NodeContext =
      l2NodeContext || (await initializeL2Node(provider))
    const blockNumber = await context.provider.getBlockNumber()
    const handler = new TestWeb3Handler(messageSubmitter, context)
    handler.blockTimestamps[numberToHexString(blockNumber)] = timestamp
    return handler
  }

  protected constructor(
    messageSubmitter: L2ToL1MessageSubmitter = new NoOpL2ToL1MessageSubmitter(),
    context: L2NodeContext
  ) {
    super(messageSubmitter, context)
  }

  /**
   * Override to add some test RPC methods.
   */
  public async handleRequest(method: string, params: any[]): Promise<string> {
    switch (method) {
      case Web3RpcMethods.increaseTimestamp:
        this.assertParameters(params, 1)
        this.increaseTimestamp(params[0])
        log.debug(`Set increased timestamp by ${params[0]} seconds.`)
        return TestWeb3Handler.successString
      case Web3RpcMethods.mine:
        return this.context.provider.send(Web3RpcMethods.mine, params)
      case Web3RpcMethods.sendTransaction:
        this.assertParameters(params, 1)
        return this.sendTransaction(params[0])
        break
      case Web3RpcMethods.snapshot:
        this.assertParameters(params, 0)
        return this.snapshot()
      case Web3RpcMethods.revert:
        this.assertParameters(params, 1)
        return this.revert(params[0])
      case Web3RpcMethods.accounts:
        this.assertParameters(params, 0)
        return this.accounts()
      default:
        return super.handleRequest(method, params)
    }
  }

  /**
   * Returns the configured timestamp if there is one, else standard timestamp calculation.
   * @returns The timestamp.
   */
  protected getTimestamp(): number {
    return super.getTimestamp() + this.timestampIncreaseSeconds
  }

  /**
   * Sets timestamp to use for future transactions.
   * @param increaseSeconds The increase in seconds as a hex string
   */
  private increaseTimestamp(increaseSeconds: any): void {
    try {
      const increaseNumber = castToNumber(increaseSeconds)
      if (increaseNumber < 0) {
        throw Error('invalid param')
      }
      this.timestampIncreaseSeconds += increaseNumber
    } catch (e) {
      const msg: string = `Expected parameter for ${Web3RpcMethods.increaseTimestamp} to be a positive number or string of a positive, base-10 number. Received: ${increaseSeconds}`
      log.error(msg)
      throw new UnsupportedMethodError(msg)
    }
  }

  /**
   * Takes a snapshot of the current node state.
   * @returns The snapshot id that can be used as an parameter of the revert endpoint
   */
  private async snapshot(): Promise<string> {
    const snapShotId = await this.context.provider.send(
      Web3RpcMethods.snapshot,
      []
    )
    this.timestampIncreaseSnapshots[snapShotId] = this.timestampIncreaseSeconds
    return snapShotId
  }

  /**
   * Sends a transactions to the backend node to be run.
   * Note: This is only exposed in testing so all accounts
   * are authorized to send transactions
   *
   * @param The transaction to send
   */
  public async sendTransaction(ovmTx: any): Promise<string> {
    if (!ovmTx.nonce) {
      ovmTx.nonce = await this.getTransactionCount(ovmTx.from, latestBlock)
    }
    if (!ovmTx.to) {
      ovmTx.to = '0x'
    }
    if (!ovmTx.gasPrice) {
      ovmTx.gasPrice = 0
    }
    if (!ovmTx.gasLimit) {
      ovmTx.gasLimit = GAS_LIMIT
    }
    ovmTx.value = 0
    return this.sendRawTransaction(
      rlpEncodeTransactionWithRandomSig(ovmTx),
      ovmTx.from
    )
  }

  /**
   * Reverts state to the specified snapshot
   * @param The snapshot id of the snapshot to restore
   */
  private async revert(snapShotId: string): Promise<string> {
    const response = await this.context.provider.send(Web3RpcMethods.revert, [
      snapShotId,
    ])
    this.timestampIncreaseSeconds = this.timestampIncreaseSnapshots[snapShotId]
    return response
  }

  public async accounts(): Promise<string> {
    log.debug('Getting accounts')
    const response = await this.context.provider.send(
      Web3RpcMethods.accounts,
      []
    )
    log.debug(`Received accounts [${response}].`)
    return response
  }
}
