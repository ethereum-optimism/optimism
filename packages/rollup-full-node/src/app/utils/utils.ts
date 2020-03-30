/* External Imports */
import { getLogger } from '@eth-optimism/core-utils/build/index'

import { providers, ContractFactory, Wallet, Contract } from 'ethers'
import * as waffle from 'ethereum-waffle'
import { FullnodeHandler, L2ToL1MessageSubmitter } from '../../types'
import { Web3Provider } from 'ethers/providers'

/* Internal Imports */
import { DefaultWeb3Handler } from '../web3-rpc-handler'
import { FullnodeRpcServer } from '../fullnode-rpc-server'
import { NoOpL2ToL1MessageSubmitter } from '../message-submitter'

const log = getLogger('utils')
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
  httpProvider['closeOVM'] = () => {
    if (!!fullnodeRpcServer) {
      fullnodeRpcServer.close()
    }
  }
  return httpProvider
}

export function getWallets(httpProvider) {
  const walletsToReturn = []
  for (let i = 0; i < 9; i++) {
    const privateKey = '0x' + ('5' + i).repeat(32)
    const nextWallet = new Wallet(privateKey, httpProvider)
    walletsToReturn[i] = nextWallet
  }
  return walletsToReturn
}

export async function deployContract(
  wallet,
  contractJSON,
  args,
  overrideOptions
) {
  const factory = new ContractFactory(
    contractJSON.abi,
    contractJSON.bytecode || contractJSON.evm.bytecode,
    wallet
  )

  const contract = await factory.deploy(...args)
  await contract.deployed()
  return contract
}

/**
 * Gets the current number of seconds since the epoch.
 *
 * @returns The seconds since epoch.
 */
export function getCurrentTime(): number {
  return Math.round(new Date().getTime() / 1000)
}
