/* External Imports */
import { ethers } from 'ethers'
import { defaultAccounts } from 'ethereum-waffle'

/* Internal Imports */
import { EVMOpcode, Opcode } from './types'
import { GasMeterOptions } from '../../src'
import { ZERO, BigNumber } from '@eth-optimism/core-utils'

export { ZERO_ADDRESS } from '@eth-optimism/core-utils'

export const DEFAULT_ACCOUNTS = defaultAccounts
export const DEFAULT_ACCOUNTS_BUIDLER = defaultAccounts.map((account) => {
  return {
    balance: new BigNumber(account.balance).toString('hex'),
    privateKey: account.secretKey,
  }
})

export const DEFAULT_UNSAFE_OPCODES: EVMOpcode[] = [
  Opcode.ADDRESS,
  Opcode.BALANCE,
  Opcode.BLOCKHASH,
  Opcode.CALL,
  Opcode.CALLCODE,
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

export const DEFAULT_SAFE_OPCODES: EVMOpcode[] = Opcode.ALL_OP_CODES.filter(
  (x) => DEFAULT_UNSAFE_OPCODES.indexOf(x) < 0
)

const calculateMask = (opcodes) => {
  // console.log(
  //   `Generating mask for opcodes: ${opcodes.map((x) => x.name).join(',')}`
  // )
  let maskHex: string = opcodes
    .map((x) => new BigNumber(2).pow(new BigNumber(x.code)))
    .reduce((prev: BigNumber, cur: BigNumber) => prev.add(cur), ZERO)
    .toString('hex')
  if (maskHex.length !== 64) {
    maskHex = '0'.repeat(64 - maskHex.length) + maskHex
  }
  // console.log(`mask: 0x${maskHex}`)
  return '0x' + maskHex
}

// const GATED_OPCODES = Opcode.HALTING_OP_CODES.push(Opcode.CALLER)
// calculateMask(GATED_OPCODES) //Calculate gated opcode mask

export const GAS_LIMIT = 1_000_000_000
export const DEFAULT_OPCODE_WHITELIST_MASK = calculateMask(DEFAULT_SAFE_OPCODES)

export const L2_TO_L1_MESSAGE_PASSER_OVM_ADDRESS =
  '0x4200000000000000000000000000000000000000'

export const CHAIN_ID = 108
export const ZERO_UINT = '00'.repeat(32)
export const DEFAULT_FORCE_INCLUSION_PERIOD_SECONDS = 600

const TX_FLAT_GAS_FEE = 30_000
const MAX_SEQUENCED_GAS_PER_EPOCH = 2_000_000_000
const MAX_QUEUED_GAS_PER_EPOCH = 2_000_000_000
const GAS_RATE_LIMIT_EPOCH_IN_SECONDS = 0

export const getDefaultGasMeterParams = (): number[] => {
  return [
    TX_FLAT_GAS_FEE,
    GAS_LIMIT,
    GAS_RATE_LIMIT_EPOCH_IN_SECONDS,
    MAX_SEQUENCED_GAS_PER_EPOCH,
    MAX_QUEUED_GAS_PER_EPOCH,
  ]
}

export const getDefaultGasMeterOptions = (): GasMeterOptions => {
  return {
    ovmTxFlatGasFee: TX_FLAT_GAS_FEE,
    ovmTxMaxGas: GAS_LIMIT,
    maxQueuedGasPerEpoch: MAX_QUEUED_GAS_PER_EPOCH,
    maxSequencedGasPerEpoch: MAX_SEQUENCED_GAS_PER_EPOCH,
    gasRateLimitEpochLength: GAS_RATE_LIMIT_EPOCH_IN_SECONDS,
  }
}
