/* External Imports */
import { ethers } from 'ethers'
import { defaultAccounts } from 'ethereum-waffle'

export const FORCE_INCLUSION_PERIOD_SECONDS = 60_000
export const MIN_ROLLUP_TX_GAS = 100_000
export const DEFAULT_ACCOUNTS = defaultAccounts
export const DEFAULT_ACCOUNTS_HARDHAT = defaultAccounts.map((account) => {
  return {
    balance: ethers.BigNumber.from(account.balance).toHexString(),
    privateKey: account.secretKey,
  }
})

export const OVM_TX_GAS_LIMIT = 10_000_000
export const RUN_OVM_TEST_GAS = 20_000_000
