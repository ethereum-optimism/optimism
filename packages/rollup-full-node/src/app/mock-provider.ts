/* Internal Imports */
import { EthnodeProxy } from '.'

/* External Imports */
import { utils, providers, ContractFactory, Wallet, Contract } from 'ethers'
import { getLogger } from '@pigi/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

const log = getLogger('rollup-mock-provider')

export const createMockOvmProvider = async () => {
  // First, we create a mock provider which is identical to a normal ethers "mock provider"
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  // Next initialize a mock fullnode for us to interact with
  const fullnode = createMockEthnodeProxy()
  const executionManagerAddress = await fullnode.deployExecutionManager()

  // Then we replace `send()` with our modified send that uses the execution manager as a proxy
  const origSend = provider.send
  provider.send = async (method: string, params: any) => {
    log.info('Sending -- Method:', method, 'Params:', params)

    // Convert the message or response if we need to
    const response = await fullnode.handleRequest(method, params)

    log.info('Received Response --', response)
    return response
  }

  // The return our slightly modified provider & the execution manager address
  return [provider, executionManagerAddress]
}

const createMockEthnodeProxy = (): EthnodeProxy => {
  // Note this is a mock provider intended for internal use
  const ethnodeProvider = createMockProvider()
  const [proxyWallet] = getWallets(ethnodeProvider)
  return new EthnodeProxy(ethnodeProvider, proxyWallet)
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
