import { ZERO_ADDRESS } from '@eth-optimism/core-utils'
import { EVMOpcode, Opcode } from '../types'

export const L1ToL2TransactionEventName = 'L1ToL2Transaction'
export const L1ToL2TransactionBatchEventName = 'NewTransactionBatchAdded'

export const CREATOR_CONTRACT_ADDRESS = ZERO_ADDRESS
export const GAS_LIMIT = 1_000_000_000

export const CHAIN_ID = 420

export const DEFAULT_UNSAFE_OPCODES: EVMOpcode[] = [
  Opcode.ADDRESS,
  Opcode.BALANCE,
  Opcode.BLOCKHASH,
  Opcode.CALLCODE,
  Opcode.CALLER,
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

// use whitelist-mask-generator.spec.ts to re-generate this
export const DEFAULT_OPCODE_WHITELIST_MASK =
  '0x600a0000000000000000001fffffffffffffffff0fcf000063f000013fff0fff'

export const L2_TO_L1_MESSAGE_PASSER_OVM_ADDRESS =
  '0x4200000000000000000000000000000000000000'

// See the getTransactionBatchCalldata(...) function of canonical-chain-batch-submitter.ts for more info
export const L2_ROLLUP_TX_SIZE_IN_BYTES_MINUS_CALLDATA = 150

/*
- 4 (Method ID)
- 32 (txs byte start)
- 32 (timestamp)
- 32 (block number)
- 32 (starts at index)
- 32 (number of txs bytes elements)
 */
export const L1_ROLLUP_BATCH_TX_STATIC_CALLDATA_BYTES = 164

/*
- 32 for start of each `bytes`
- 32 for length of each `bytes`
 */
export const L1_ROLLUP_BATCH_TX_BYTES_PER_L2_TX = 64

/*
- 32 for nonce
- 32 for gas price
- 32 for gas limit
- 32 for value
- 32 for R (signature)
- 32 for S (signature)
- 2 (at most) for V (signature)
- 20 for to address
 */
export const L1_ROLLUP_BATCH_TX_STATIC_OVERHEAD_BYTES = 214
