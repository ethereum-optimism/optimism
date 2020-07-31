/* External Imports */
import { ethers } from 'ethers'
import { defaultAccounts } from 'ethereum-waffle'
import {
  Opcode,
  DEFAULT_UNSAFE_OPCODES as UNSAFE_OPCODES,
} from '@eth-optimism/rollup-core'

/* Internal Imports */
export { ZERO_ADDRESS } from '@eth-optimism/core-utils'

export const DEFAULT_ACCOUNTS = defaultAccounts
export const DEFAULT_ACCOUNTS_BUIDLER = defaultAccounts.map((account) => {
  return {
    balance: ethers.BigNumber.from(account.balance).toHexString(),
    privateKey: account.secretKey,
  }
})

export const GAS_LIMIT = 1_000_000_000
export const DEFAULT_OPCODE_WHITELIST_MASK =
  '0x600a0000000000000000001fffffffffffffffff0fcf000063f000013fff0fff'

export const L2_TO_L1_MESSAGE_PASSER_OVM_ADDRESS =
  '0x4200000000000000000000000000000000000000'

export const CHAIN_ID = 108
export const ZERO_UINT = '00'.repeat(32)
export const DEFAULT_FORCE_INCLUSION_PERIOD = 600

export const DEFAULT_UNSAFE_OPCODES = UNSAFE_OPCODES.concat([Opcode.CHAINID])

export const HALTING_OPCODES = Opcode.HALTING_OP_CODES
export const HALTING_OPCODES_NO_JUMP = HALTING_OPCODES.filter(
  (x) => x.name !== 'JUMP'
)
export const JUMP_OPCODES = [Opcode.JUMP, Opcode.JUMPI]
export const WHITELISTED_NOT_HALTING_OR_CALL = Opcode.ALL_OP_CODES.filter(
  (x) =>
    DEFAULT_UNSAFE_OPCODES.indexOf(x) < 0 &&
    HALTING_OPCODES.indexOf(x) < 0 &&
    x.name !== 'CALL'
)

export const fillHexBytes = (byte: string): string => {
  return '0x' + byte.repeat(32)
}
