/* External Imports */
import { defaultAccounts } from 'ethereum-waffle'

export const DEFAULT_ACCOUNTS_HARDHAT = defaultAccounts.map((account) => {
  return {
    balance: account.balance,
    privateKey: account.secretKey,
  }
})

export const RUN_OVM_TEST_GAS = 20_000_000
export const L2_GAS_DISCOUNT_DIVISOR = 32
export const ENQUEUE_GAS_COST = 60_000

export const NON_NULL_BYTES32 =
  '0x1111111111111111111111111111111111111111111111111111111111111111'
export const NON_ZERO_ADDRESS = '0x1111111111111111111111111111111111111111'
