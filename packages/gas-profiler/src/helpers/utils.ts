import { ethers, Wallet, Contract } from 'ethers'
import { Ganache } from './ganache'
import { Provider } from 'ethers/providers'
import { Interface } from 'ethers/utils'

export interface Toolbox {
  provider: Provider
  wallet: Wallet
  ganache?: Ganache
}

/**
 * Utility; automatically spins up tools necessary for executing contracts.
 * @returns a "toolbox" of useful objects.
 */
export const getToolbox = async (): Promise<Toolbox> => {
  // Just an arbitrary secret key for testing.
  const sk =
    '0x0123456789012345678901234567890123456789012345678901234567890123'

  // Launch ganache with a reasonably high gas limit, seed our account with some ETH.
  const ganache = new Ganache({
    accounts: [
      {
        secretKey: sk,
        balance: ethers.utils.parseEther('100'),
      },
    ],
    gasLimit: 0x989680,
  })
  await ganache.start()

  // Connect the `ethers` provider and create a wallet.
  const provider = new ethers.providers.JsonRpcProvider(
    `http://localhost:${ganache.port}`
  )
  const wallet = new ethers.Wallet(sk, provider)

  return {
    provider,
    wallet,
    ganache,
  }
}

/**
 * Utility; converts an `ethers` contract object into a corresponding interface.
 * @param contract `ethers` contract object to convert.
 * @returns an interface object for the contract.
 */
export const getInterface = (contract: Contract): Interface => {
  return new ethers.utils.Interface(contract.interface.abi)
}
