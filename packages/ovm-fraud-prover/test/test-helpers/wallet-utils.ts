/* External Imports */
import { ethers, Wallet } from 'ethers'

/* Internal Imports */
import { DEFAULT_ACCOUNTS } from './constants'

/**
 * Retrieves the default wallets as Ethers wallets.
 * @param provider Ethers provider to attach the wallets to.
 * @returns Array of Ethers wallets.
 */
export const getWallets = (provider: any): Wallet[] => {
  return DEFAULT_ACCOUNTS.map((account) => {
    return new ethers.Wallet(account.secretKey, provider)
  })
}