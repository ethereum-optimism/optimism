/* External Imports */
import { defaultAccounts } from 'ethereum-waffle'

export const FORCE_INCLUSION_PERIOD_SECONDS = 600
export const DEFAULT_ACCOUNTS = defaultAccounts
export const DEFAULT_ACCOUNTS_HARDHAT = defaultAccounts.map((account) => {
  return {
    balance: account.balance,
    privateKey: account.secretKey,
  }
})

export const OVM_TX_GAS_LIMIT = 10_000_000
export const RUN_OVM_TEST_GAS = 20_000_000
