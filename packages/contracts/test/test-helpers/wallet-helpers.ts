/* External Imports */
import { ethers, Wallet } from 'ethers'

/* Internal Imports */
import { DEFAULT_ACCOUNTS } from './constants'

export const getWallets = (provider?: any): Wallet[] => {
  return DEFAULT_ACCOUNTS.map((account) => {
    return new ethers.Wallet(account.secretKey, provider)
  })
}

export const signTransaction = async (
  wallet: Wallet,
  transaction: any
): Promise<string> => {
  return wallet.signTransaction(transaction)
}

export const getRawSignedComponents = (signed: string): any[] => {
  return [signed.slice(130, 132), signed.slice(2, 66), signed.slice(66, 130)]
}

export const getSignedComponents = (signed: string): any[] => {
  return ethers.utils.RLP.decode(signed).slice(-3)
}
