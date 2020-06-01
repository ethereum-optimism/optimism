/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'

import { ContractFactory, Wallet } from 'ethers'

const log = getLogger('utils')

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
