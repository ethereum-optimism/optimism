/* External Imports */
import { add0x, getLogger, remove0x } from '@eth-optimism/core-utils'
import { OPCODE_WHITELIST_MASK } from '@eth-optimism/ovm'

import { createMockProvider, getWallets } from 'ethereum-waffle'
import { Contract, Wallet } from 'ethers'
import { Web3Provider } from 'ethers/providers'

/* Internal Imports */
import { DEFAULT_ETHNODE_GAS_LIMIT } from './index'
import { DefaultWeb3Handler } from './web3-rpc-handler'
import { UnsupportedMethodError, Web3RpcMethods } from '../types'

const log = getLogger('test-web3-handler')

/**
 * Test Handler that provides extra functionality for testing.
 */
export class TestWeb3Handler extends DefaultWeb3Handler {
  public static readonly successString = 'success'

  private timestampIncreaseSeconds: number = 0

  /**
   * Creates a local node, deploys the L2ExecutionManager to it, and returns a
   * TestHandler that handles Web3 requests to it.
   *
   * @param provider (optional) The web3 provider to use.
   * @returns The constructed Web3 handler.
   */
  public static async create(
    provider: Web3Provider = createMockProvider({
      gasLimit: DEFAULT_ETHNODE_GAS_LIMIT,
    })
  ): Promise<TestWeb3Handler> {
    // Initialize a fullnode for us to interact with
    const [wallet] = getWallets(provider)
    const executionManager: Contract = await DefaultWeb3Handler.deployExecutionManager(
      wallet,
      OPCODE_WHITELIST_MASK
    )

    return new TestWeb3Handler(provider, wallet, executionManager)
  }

  protected constructor(
    provider: Web3Provider,
    wallet: Wallet,
    executionManager: Contract
  ) {
    super(provider, wallet, executionManager)
  }

  /**
   * Override to add some test RPC methods.
   */
  public async handleRequest(method: string, params: any[]): Promise<string> {
    if (method === Web3RpcMethods.increaseTimestamp) {
      this.assertParameters(params, 1)
      this.increaseTimestamp(params[0])
      log.debug(`Set increased timestamp by ${params[0]} seconds.`)
      return TestWeb3Handler.successString
    }
    if (method === Web3RpcMethods.getTimestamp) {
      this.assertParameters(params, 0)
      return add0x(this.getTimestamp().toString(16))
    }

    return super.handleRequest(method, params)
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
      const increaseNumber = parseInt(remove0x(increaseSeconds), 16)
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
}
