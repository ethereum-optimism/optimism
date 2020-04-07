/* External Imports */
import {
  add0x,
  getDeployedContractAddress,
  getLogger,
  logError,
} from '@eth-optimism/core-utils'
import {
  L1ToL2TransactionPasserContractDefinition,
  L2ToL1MessageReceiverContractDefinition,
} from '@eth-optimism/ovm'
import { Address } from '@eth-optimism/rollup-core'

import { Contract, providers, Wallet } from 'ethers'
import { createMockProvider, deployContract } from 'ethereum-waffle'

/* Internal Imports */
import { DEFAULT_ETHNODE_GAS_LIMIT, Environment } from '../index'
import { L1NodeContext } from '../../types'
import { JsonRpcProvider, Provider } from 'ethers/providers'

const log = getLogger('local-l1-node')

/**
 * Initializes the L1 node based on configuration, returning the L1NodeContext.
 *
 * @returns The L1NodeContext object with all necessary L1 node info.
 */
export const initializeL1Node = async (): Promise<L1NodeContext> => {
  const wallet: Wallet = getSequencerWallet()
  const provider: Provider = getProvider(wallet)
  const sequencerWallet: Wallet = wallet.connect(provider)

  const l2ToL1MessageReceiver: Contract = await getL2ToL1MessageReceiverContract(
    provider,
    sequencerWallet,
    0
  )
  const l1ToL2TransactionPasser: Contract = await getL1ToL2TransactionPasserContract(
    provider,
    sequencerWallet,
    1
  )

  return {
    provider,
    sequencerWallet,
    l2ToL1MessageReceiver,
    l1ToL2TransactionPasser,
  }
}

/**
 * Gets the wallet for the sequencer based on configuration in environment variables.
 *
 * @returns The sequencer Wallet.
 */
const getSequencerWallet = (): Wallet => {
  let sequencerWallet: Wallet
  if (Environment.sequencerPrivateKey()) {
    sequencerWallet = new Wallet(add0x(Environment.sequencerPrivateKey()))
    log.info(
      `Initialized sequencer wallet from private key. Address: ${sequencerWallet.address}`
    )
  } else if (Environment.sequencerMnemonic()) {
    sequencerWallet = Wallet.fromMnemonic(Environment.sequencerMnemonic())
    log.info(
      `Initialized sequencer wallet from mnemonic. Address: ${sequencerWallet.address}`
    )
  } else {
    const msg: string = `No L1 Sequencer Mnemonic Provided! Set the L1_SEQUENCER_MNEMONIC or L1_SEQUENCER_PRIVATE_KEY env var!.`
    log.error(msg)
    throw Error(msg)
  }

  return sequencerWallet
}

/**
 * Gets the provider for the L1 node based on configuration. If no existing L1 node
 * URL is configured, this will deploy a local node.
 *
 * @param wallet The wallet to initialize with a sufficiently large balance if deploying a test node.
 * @returns The provider to use.
 */
const getProvider = (wallet: Wallet): Provider => {
  if (Environment.l1NodeWeb3Url()) {
    log.info(`Connecting to L1 web3 URL: ${Environment.l1NodeWeb3Url()}`)
    return new JsonRpcProvider(Environment.l1NodeWeb3Url())
  } else {
    log.info(`Deploying local L1 node on port ${Environment.localL1NodePort()}`)
    return startLocalL1Node(wallet, Environment.localL1NodePort())
  }
}

/**
 * Starts a local node on the provided port.
 *
 * @param sequencerWallet The Wallet to use for the Sequencer in contracts that need Sequencer ownership or reference.
 * @param port The port the node should be reachable at.
 * @returns The Provider for the local node.
 */
const startLocalL1Node = (sequencerWallet: Wallet, port: number): Provider => {
  const opts = {
    gasLimit: DEFAULT_ETHNODE_GAS_LIMIT,
    allowUnlimitedContractSize: true,
    locked: false,
    port,
    accounts: [
      {
        balance: '0xfffffffffffffffffffffffffff',
        secretKey: add0x(sequencerWallet.privateKey),
      },
    ],
  }
  if (!!Environment.localL1NodePersistentDbPath()) {
    log.info(
      `Found configuration for L1 Node DB Path: ${Environment.localL1NodePersistentDbPath()}`
    )
    opts['db_path'] = Environment.localL1NodePersistentDbPath()
  }

  const provider: providers.Web3Provider = createMockProvider(opts)
  log.info(`Local L1 node created with config: ${JSON.stringify(opts)}`)

  return provider
}

/**
 * Gets the L2ToL1MessageReceiver contract to use with the L1 node. This will automatically
 * deploy a new L2ToL1MessageReceiver contract if one does not exist for the specified provider.
 *
 * @param provider The provider to use to determine if the contract has already been deployed.
 * @param wallet The wallet to use for the contract.
 * @param nonceWhenDeployed If the contract has already been deployed, this is the nonce for the deploy transaction.
 * @returns The L2ToL1MessageReceiver contract.
 */
const getL2ToL1MessageReceiverContract = async (
  provider: Provider,
  wallet: Wallet,
  nonceWhenDeployed: number
): Promise<Contract> => {
  const l2ToL1MessageReceiverAddress: Address =
    Environment.l2ToL1MessageReceiverAddress() ||
    (await getDeployedContractAddress(
      nonceWhenDeployed,
      provider,
      wallet.address
    ))

  let l2ToL1MessageReceiver: Contract
  if (l2ToL1MessageReceiverAddress) {
    log.info(
      `Using existing L2ToL1MessageReceiver deployed at ${l2ToL1MessageReceiverAddress}`
    )
    l2ToL1MessageReceiver = new Contract(
      l2ToL1MessageReceiverAddress,
      L2ToL1MessageReceiverContractDefinition.abi,
      wallet
    )
  } else {
    log.info(`Deploying L2ToL1MessageReceiver!`)
    l2ToL1MessageReceiver = await deployL2ToL1MessageReceiver(wallet)
    log.info(
      `L2ToL1MessageReceiver deployed at address ${l2ToL1MessageReceiver.address}`
    )
  }

  return l2ToL1MessageReceiver
}

/**
 * Deploys the L2ToL1MessageReceiver contract using the provided Wallet.
 *
 * @param wallet The wallet to use for the deployment
 * @returns The resulting Contract.
 */
const deployL2ToL1MessageReceiver = async (
  wallet: Wallet
): Promise<Contract> => {
  log.info(`Deploying L2ToL1MessageReceiver to local L1 Node`)

  let contract: Contract
  try {
    contract = await deployContract(
      wallet,
      L2ToL1MessageReceiverContractDefinition,
      [wallet.address, Environment.l2ToL1MessageFinalityDelayInBlocks()]
    )
  } catch (e) {
    logError(log, 'Error Deploying L2ToL1MessageReceiver', e)
    throw e
  }

  log.info(
    `L2ToL1MessageReceiver deployed to local L1 Node at address ${contract.address}`
  )
  return contract
}

/**
 * Gets the L1ToL2transactionPasser contract to use with the L1 node. This will automatically
 * deploy a new L1ToL2transactionPasser contract if one does not exist for the specified provider.
 *
 * @param provider The provider to use to determine if the contract has already been deployed.
 * @param wallet The wallet to use for the contract.
 * @param nonceWhenDeployed If the contract has already been deployed, this is the nonce for the deploy transaction.
 * @returns The L1ToL2transactionPasser contract.
 */
const getL1ToL2TransactionPasserContract = async (
  provider: Provider,
  wallet: Wallet,
  nonceWhenDeployed: number
): Promise<Contract> => {
  const l1ToL2transactionPasserAddress: Address =
    Environment.l1ToL2TransactionPasserAddress() ||
    (await getDeployedContractAddress(
      nonceWhenDeployed,
      provider,
      wallet.address
    ))

  let l2ToL1transactionPasser: Contract
  if (l1ToL2transactionPasserAddress) {
    log.info(
      `Using existing L1ToL2transactionPasser deployed at ${l1ToL2transactionPasserAddress}`
    )
    l2ToL1transactionPasser = new Contract(
      l1ToL2transactionPasserAddress,
      L2ToL1MessageReceiverContractDefinition.abi,
      wallet
    )
  } else {
    log.info(`Deploying L1ToL2transactionPasser!`)
    l2ToL1transactionPasser = await deployL1ToL2transactionPasser(wallet)
    log.info(
      `L1ToL2transactionPasser deployed at address ${l2ToL1transactionPasser.address}`
    )
  }

  return l2ToL1transactionPasser
}

/**
 * Deploys the L1ToL2transactionPasser contract using the provided Wallet.
 *
 * @param wallet The wallet to use for the deployment
 * @returns The resulting Contract.
 */
const deployL1ToL2transactionPasser = async (
  wallet: Wallet
): Promise<Contract> => {
  log.info(`Deploying L2ToL1MessageReceiver to local L1 Node`)

  let contract: Contract
  try {
    contract = await deployContract(
      wallet,
      L1ToL2TransactionPasserContractDefinition,
      []
    )
  } catch (e) {
    logError(log, 'Error Deploying L1ToL2transactionPasser', e)
    throw e
  }

  log.info(
    `L1ToL2transactionPasser deployed to local L1 Node at address ${contract.address}`
  )
  return contract
}
