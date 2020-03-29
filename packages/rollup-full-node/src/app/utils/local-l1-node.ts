/* External Imports */
import {
  getDeployedContractAddress,
  getLogger,
  logError,
} from '@eth-optimism/core-utils'
import { L2ToL1MessageReceiverContractDefinition } from '@eth-optimism/ovm'

import { Contract, ethers, providers, Wallet } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Internal Imports */
import { DEFAULT_ETHNODE_GAS_LIMIT, Environment } from '../index'
import { L1NodeContext } from '../../types'

const log = getLogger('local-l1-node')

/**
 * Starts a local node on the provided port, using the provided mnemonic to
 * deploy the necessary contracts for bootstrapping.
 *
 * @param sequencerMnemonic The mnemonic to use for the Sequencer in contracts that need Sequencer ownership or reference.
 * @param port The port the node should be reachable at.
 * @returns The L1 node context with all info necessary to use the L1 node.
 */
export const startLocalL1Node = async (
  sequencerMnemonic: string,
  port: number
): Promise<L1NodeContext> => {
  const opts = {
    gasLimit: DEFAULT_ETHNODE_GAS_LIMIT,
    allowUnlimitedContractSize: true,
    locked: false,
    port,
    mnemonic: sequencerMnemonic,
  }
  if (!!Environment.localL1NodePersistentDbPath()) {
    log.info(
      `Found configuration for L1 Node DB Path: ${Environment.localL1NodePersistentDbPath()}`
    )
    opts['db_path'] = Environment.localL1NodePersistentDbPath()
  }

  const provider: providers.Web3Provider = createMockProvider(opts)
  log.info(`Local L1 node created with config: ${JSON.stringify(opts)}`)

  const sequencerWallet = getWallets(provider)[0]

  const receiverAddress: string = await getDeployedContractAddress(
    0,
    provider,
    sequencerWallet.address
  )

  let l2ToL1MessageReceiver: Contract
  if (!receiverAddress) {
    l2ToL1MessageReceiver = await deployL2ToL1MessageReceiver(sequencerWallet)
  } else {
    log.info(
      `Found deployed L2ToL1MessageReceiver contract at address: ${receiverAddress}`
    )
    l2ToL1MessageReceiver = new Contract(
      receiverAddress,
      L2ToL1MessageReceiverContractDefinition.interface,
      provider
    )
  }

  return {
    provider,
    sequencerWallet,
    l2ToL1MessageReceiver,
  }
}

export const deployL2ToL1MessageReceiver = async (
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
