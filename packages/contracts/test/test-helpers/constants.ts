/* External Imports */
import { ethers } from 'ethers'
import { defaultAccounts } from 'ethereum-waffle'

/* Internal Imports */
import { EVMOpcode, Opcode } from './types'

export { ZERO_ADDRESS } from '@eth-optimism/core-utils'

export const DEFAULT_ACCOUNTS = defaultAccounts
export const DEFAULT_ACCOUNTS_BUIDLER = defaultAccounts.map((account) => {
  return {
    balance: ethers.BigNumber.from(account.balance).toHexString(),
    privateKey: account.secretKey,
  }
})

export const DEFAULT_UNSAFE_OPCODES: EVMOpcode[] = [
  Opcode.ADDRESS,
  Opcode.BALANCE,
  Opcode.BLOCKHASH,
  Opcode.CALLCODE,
  Opcode.CALLER,
  Opcode.CHAINID,
  Opcode.COINBASE,
  Opcode.CREATE,
  Opcode.CREATE2,
  Opcode.DELEGATECALL,
  Opcode.DIFFICULTY,
  Opcode.EXTCODESIZE,
  Opcode.EXTCODECOPY,
  Opcode.EXTCODEHASH,
  Opcode.GASLIMIT,
  Opcode.GASPRICE,
  Opcode.NUMBER,
  Opcode.ORIGIN,
  Opcode.SELFBALANCE,
  Opcode.SELFDESTRUCT,
  Opcode.SLOAD,
  Opcode.SSTORE,
  Opcode.STATICCALL,
  Opcode.TIMESTAMP,
]

export const GAS_LIMIT = 1_000_000_000
export const DEFAULT_OPCODE_WHITELIST_MASK =
  '0x600a0000000000000000001fffffffffffffffff0fcf000063f000013fff0fff'

export const L2_TO_L1_MESSAGE_PASSER_OVM_ADDRESS =
  '0x4200000000000000000000000000000000000000'

export const CHAIN_ID = 108
export const ZERO_UINT = '00'.repeat(32)
export const DEFAULT_FORCE_INCLUSION_PERIOD_SECONDS = 600

const TX_FLAT_GAS_FEE = 30_000
const MAX_SEQUENCED_GAS_PER_EPOCH = 2_000_000_000
const MAX_QUEUED_GAS_PER_EPOCH = 2_000_000_000
const GAS_RATE_LIMIT_EPOCH_LENGTH = 0
export const DEFAULT_GAS_METER_PARAMS = [
  TX_FLAT_GAS_FEE,
  GAS_LIMIT,
  GAS_RATE_LIMIT_EPOCH_LENGTH,
  MAX_SEQUENCED_GAS_PER_EPOCH,
  MAX_QUEUED_GAS_PER_EPOCH,
]