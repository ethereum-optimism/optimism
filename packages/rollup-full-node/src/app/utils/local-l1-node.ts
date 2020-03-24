/* External Imports */
import { L2ToL1MessageReceiverContractDefinition } from '@eth-optimism/ovm'

import { Contract, ethers, providers, Wallet } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Internal Imports */
import { DEFAULT_ETHNODE_GAS_LIMIT } from '../index'

const mnemonic: string =
  process.env.LOCAL_L1_MNEMONIC || ethers.Wallet.createRandom().mnemonic
const finalityDelayInBlocks: string =
  process.env.FINALITY_DELAY_IN_BLOCKS || '0'

export const startLocalL1Node = async (
  port: number
): Promise<providers.Web3Provider> => {
  const provider: providers.Web3Provider = createMockProvider({
    gasLimit: DEFAULT_ETHNODE_GAS_LIMIT,
    allowUnlimitedContractSize: true,
    locked: false,
    port,
    mnemonic,
  })

  const wallet = getWallets(provider)[0]
  await deployL2ToL1MessageReceiver(wallet)

  return provider
}

export const deployL2ToL1MessageReceiver = async (
  wallet: Wallet
): Promise<Contract> => {
  const contract = await deployContract(
    wallet,
    L2ToL1MessageReceiverContractDefinition,
    [wallet.address, parseInt(finalityDelayInBlocks, 10)]
  )

  process.env.L2_TO_L1_MESSAGE_RECEIVER_ADDRESS = contract.address
  return contract
}
