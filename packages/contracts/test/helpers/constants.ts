/* External Imports */
import { ethers } from 'ethers'
import { defaultAccounts } from 'ethereum-waffle'
import xor from 'buffer-xor'

/* Internal Imports */
import { makeHexString, makeAddress, fromHexString, toHexString } from './utils'

export const DEFAULT_ACCOUNTS = defaultAccounts
export const DEFAULT_ACCOUNTS_BUIDLER = defaultAccounts.map((account) => {
  return {
    balance: ethers.BigNumber.from(account.balance).toHexString(),
    privateKey: account.secretKey,
  }
})

export const OVM_TX_GAS_LIMIT = 10_000_000
export const RUN_OVM_TEST_GAS = 20_000_000
export const FORCE_INCLUSION_PERIOD_SECONDS = 600

export const NULL_BYTES32 = makeHexString('00', 32)
export const NON_NULL_BYTES32 = makeHexString('11', 32)
export const ZERO_ADDRESS = makeAddress('00')
export const NON_ZERO_ADDRESS = makeAddress('11')

export const VERIFIED_EMPTY_CONTRACT_HASH =
  '0x00004B1DC0DE000000004B1DC0DE000000004B1DC0DE000000004B1DC0DE0000'

export const STORAGE_XOR_VALUE =
  '0xFEEDFACECAFEBEEFFEEDFACECAFEBEEFFEEDFACECAFEBEEFFEEDFACECAFEBEEF'

export const NUISANCE_GAS_COSTS = {
  NUISANCE_GAS_SLOAD: 20000,
  NUISANCE_GAS_SSTORE: 20000,
  MIN_NUISANCE_GAS_PER_CONTRACT: 30000,
  NUISANCE_GAS_PER_CONTRACT_BYTE: 100,
  MIN_GAS_FOR_INVALID_STATE_ACCESS: 30000,
}

// TODO: get this exported/imported somehow in a way that we can do math on it.  unfortunately using require('.....artifacts/contract.json') is erroring...
export const Helper_TestRunner_BYTELEN = 3654

export const STORAGE_XOR =
  '0xfeedfacecafebeeffeedfacecafebeeffeedfacecafebeeffeedfacecafebeef'
export const getStorageXOR = (key: string): string => {
  return toHexString(xor(fromHexString(key), fromHexString(STORAGE_XOR)))
}

export const EMPTY_ACCOUNT_CODE_HASH =
  '0x00004B1DC0DE000000004B1DC0DE000000004B1DC0DE000000004B1DC0DE0000'
export const KECCAK_256_NULL =
  '0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470'
