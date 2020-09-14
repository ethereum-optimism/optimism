/* External Imports */
import { ethers } from 'ethers'
import { defaultAccounts } from 'ethereum-waffle'

/* Internal Imports */
import { makeHexString, makeAddress } from './byte-utils'

export const DEFAULT_ACCOUNTS = defaultAccounts
export const DEFAULT_ACCOUNTS_BUIDLER = defaultAccounts.map((account) => {
  return {
    balance: ethers.BigNumber.from(account.balance).toHexString(),
    privateKey: account.secretKey,
  }
})

export const GAS_LIMIT = 1_000_000_000

export const NULL_BYTES32 = makeHexString('00', 32)
export const NON_NULL_BYTES32 = makeHexString('11', 32)
export const ZERO_ADDRESS = makeAddress('00')
export const NON_ZERO_ADDRESS = makeAddress('11')
