/* External Imports */
import { getLogger } from '@eth-optimism/core-utils/build/src'

import { Contract, ContractFactory, providers, Wallet } from 'ethers'
import { Web3Provider } from 'ethers/providers'
import * as waffle from 'ethereum-waffle'

/* Internal Imports */
import { FullnodeHandler, L2ToL1MessageSubmitter } from '../types'
import { NoOpL2ToL1MessageSubmitter } from './message-submitter'
import { DefaultWeb3Handler } from './web3-rpc-handler'
import { FullnodeRpcServer } from './fullnode-rpc-server'

const log = getLogger('utils')

const balance = '10000000000000000000000000000000000';

const privateKeys = [
  '0x29f3edee0ad3abf8e2699402e0e28cd6492c9be7eaab00d732a791c33552f797',
  '0x5c8b9227cd5065c7e3f6b73826b8b42e198c4497f6688e3085d5ab3a6d520e74',
  '0x50c8b3fc81e908501c8cd0a60911633acaca1a567d1be8e769c5ae7007b34b23',
  '0x706618637b8ca922f6290ce1ecd4c31247e9ab75cf0530a0ac95c0332173d7c5',
  '0xe217d63f0be63e8d127815c7f26531e649204ab9486b134ec1a0ae9b0fee6bcf',
  '0x8101cca52cd2a6d8def002ffa2c606f05e109716522ca2440b2cc84e4d49700b',
  '0x837fd366bc7402b65311de9940de0d6c0ba3125629b8509aebbfb057ebeaaa25',
  '0xba35c32f7cbda6a6cedeea5f73ff928d1e41557eddfd457123f6426a43adb1e4',
  '0x71f7818582e55456cb575eea3d0ce408dcf4cbbc3d845e86a7936d2f48f74035',
  '0x03c909455dcef4e1e981a21ffb14c1c51214906ce19e8e7541921b758221b5ae'
];

const defaultAccounts = privateKeys
  .map(secretKey => ({balance, secretKey}));

/**
 * Creates a Provider that uses the provided handler to handle `send`s.
 *
 * @param fullnodeHandler The handler to use for the provider's send function.
 * @return The provider.
 */
export const createProviderForHandler = (
  fullnodeHandler: FullnodeHandler
): Web3Provider => {
  // First, we create a mock provider which is identical to a normal ethers "mock provider"
  const provider = waffle.createMockProvider()

  // Then we replace `send()` with our modified send that uses the execution manager as a proxy
  provider.send = async (method: string, params: any) => {
    log.debug('Sending -- Method:', method, 'Params:', params)

    // Convert the message or response if we need to
    const response = await fullnodeHandler.handleRequest(method, params)

    log.debug('Received Response --', response)
    return response
  }

  // The return our slightly modified provider & the execution manager address
  return provider
}

class MockProvider extends providers.Web3Provider {
  private fullnodeRpcServer
  constructor(httpProvider, _fullnodeRpcServer) {
    super(httpProvider);
    this.fullnodeRpcServer = _fullnodeRpcServer
  }

  getWallets() {
    const items = defaultAccounts;
    return items.map((x: any) => new Wallet(x.secretKey, this));
  }
  closeOVM() {
    if (!!this.fullnodeRpcServer) this.fullnodeRpcServer.close()
  }
}

export async function createMockProvider(
  port: number = 9999,
  messageSubmitter: L2ToL1MessageSubmitter = new NoOpL2ToL1MessageSubmitter()
) {
  const host = '0.0.0.0'
  const fullnodeHandler = await DefaultWeb3Handler.create(messageSubmitter)
  const fullnodeRpcServer = new FullnodeRpcServer(fullnodeHandler, host, port)
  fullnodeRpcServer.listen()
  const baseUrl = `http://${host}:${port}`
  const httpProvider = new providers.JsonRpcProvider(baseUrl)
  const web3Provider = new MockProvider(httpProvider, fullnodeRpcServer)

  return web3Provider
}

const defaultDeployOptions = {
  gasLimit: 4000000,
  gasPrice: 9000000000,
}

/**
 * Helper function for generating initcode based on a contract definition & constructor arguments
 */
export async function deployOvmContract(
  wallet: Wallet,
  contractJSON: any,
  args: any[] = [],
  overrideOptions: providers.TransactionRequest = {}
) {
  // Get the factory and deploy the contract
  const factory = new ContractFactory(
    contractJSON.abi,
    contractJSON.bytecode,
    wallet
  )
  const contract = await factory.deploy(...args, {
    ...defaultDeployOptions,
    ...overrideOptions,
  })

  // Now get the deployment tx reciept so we can find the contract address
  // NOTE: We need to get the address manually because we do not have EOAs
  const deploymentTxReceipt = await wallet.provider.getTransactionReceipt(
    contract.deployTransaction.hash
  )
  // Create a new contract object with this wallet & the **real** address
  return new Contract(
    deploymentTxReceipt.contractAddress,
    contractJSON.abi,
    wallet
  )
}
