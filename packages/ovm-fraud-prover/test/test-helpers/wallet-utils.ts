import { ethers, Wallet } from 'ethers'
import { DEFAULT_ACCOUNTS } from './constants'

export const getWallets = (provider: any): Wallet[] => {
  return DEFAULT_ACCOUNTS.map((account) => {
    return new ethers.Wallet(account.secretKey, provider)
  })
}