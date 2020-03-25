/* Externals Import */
import { getDeployedContractAddress, getLogger } from '@eth-optimism/core-utils'
import {
  GAS_LIMIT,
  L2ExecutionManagerContractDefinition,
  L2ToL1MessagePasserContractDefinition,
  CHAIN_ID,
  L2_TO_L1_MESSAGE_PASSER_OVM_ADDRESS,
} from '@eth-optimism/ovm'
import { Address } from '@eth-optimism/rollup-core'

import { Contract, Wallet } from 'ethers'
import { createMockProvider, getWallets } from 'ethereum-waffle'

/* Internal Imports */
import {
  DEFAULT_ETHNODE_GAS_LIMIT,
  deployContract,
  Environment,
} from '../index'
import { JsonRpcProvider } from 'ethers/providers'
import { L2NodeContext } from '../../types'

const log = getLogger('l2-node')

/* Configuration */

/**
 * Initializes the L2 Node, which entails:
 * - Creating a local instance if the given provider is undefined.
 * - Creating a wallet to use to interact with it.
 * - Deploying the ExecutionManager if it does not already exist.
 *
 * @param web3Provider (optional) The JsonRpcProvider to use to connect to a remote node.
 * @returns The L2NodeContext necessary to interact with the L2 Node.
 */
export async function initializeL2Node(
  web3Provider?: JsonRpcProvider
): Promise<L2NodeContext> {
  let provider: JsonRpcProvider = web3Provider
  if (!web3Provider) {
    const opts = {
      gasLimit: DEFAULT_ETHNODE_GAS_LIMIT,
      allowUnlimitedContractSize: true,
    }
    const persistedGanacheDbPath = Environment.localL2NodePersistentDbPath()
    if (!!persistedGanacheDbPath) {
      opts['db_path'] = persistedGanacheDbPath
      opts['network_id'] = CHAIN_ID
    }
    provider = createMockProvider(opts)
  }

  // Initialize a fullnode for us to interact with
  let wallet: Wallet

  // If we're given a provider, our wallet must be configured from a private key file
  if (web3Provider) {
    wallet = !!Environment.l2WalletMnemonic()
      ? Wallet.fromMnemonic(Environment.l2WalletMnemonic())
      : Wallet.createRandom()

    wallet.connect(provider)
  } else {
    ;[wallet] = getWallets(provider)
  }

  let nonce: number = 0
  const executionManagerAddress: Address = await getDeployedContractAddress(
    nonce++,
    provider,
    wallet.address
  )

  let executionManager: Contract
  let l2ToL1MessagePasser: Contract
  if (executionManagerAddress) {
    log.info(
      `Using existing ExecutionManager deployed at ${executionManagerAddress}`
    )
    executionManager = new Contract(
      executionManagerAddress,
      L2ExecutionManagerContractDefinition.abi,
      wallet
    )
  } else {
    executionManager = await deployExecutionManager(wallet)
  }

  log.info(
    `Using existing L2ToL1MessagePasser deployed at ${L2_TO_L1_MESSAGE_PASSER_OVM_ADDRESS}`
  )
  l2ToL1MessagePasser = new Contract(
    L2_TO_L1_MESSAGE_PASSER_OVM_ADDRESS,
    L2ToL1MessagePasserContractDefinition.abi,
    wallet
  )

  return {
    wallet,
    provider,
    executionManager,
    l2ToL1MessagePasser,
  }
}

/**
 * Deploys the ExecutionManager contract with the provided wallet and whitelist,
 * returning the resulting Contract.
 *
 * @param wallet The wallet to be used, containing all connection info.
 * @returns The deployed Contract.
 */
export async function deployExecutionManager(
  wallet: Wallet
): Promise<Contract> {
  log.debug('Deploying execution manager...')

  const executionManager: Contract = await deployContract(
    wallet,
    L2ExecutionManagerContractDefinition,
    [Environment.opcodeWhitelistMask(), wallet.address, GAS_LIMIT, true],
    { gasLimit: DEFAULT_ETHNODE_GAS_LIMIT }
  )

  log.info('Deployed execution manager to address:', executionManager.address)

  return executionManager
}
