import {
  Address,
  bufferToBytecode,
  EVMBytecode,
  EVMOpcode,
  formatBytecode,
  Opcode,
} from '@pigi/rollup-core/build/index'
import {
  EvmIntrospectionUtil,
  ExecutionResultComparison,
} from '../src/types/vm'
import { should } from './setup'
import { bufferUtils } from '@pigi/core-utils/build'

export const whitelistedOpcodes: EVMOpcode[] = [
  Opcode.PUSH1,
  Opcode.PUSH4,
  Opcode.PUSH29,
  Opcode.PUSH32,
  Opcode.MSTORE,
  Opcode.CALLDATALOAD,
  Opcode.SWAP1,
  Opcode.SWAP2,
  Opcode.SWAP3,
  Opcode.DIV,
  Opcode.DUP1,
  Opcode.DUP2,
  Opcode.DUP3,
  Opcode.DUP4,
  Opcode.EQ,
  Opcode.JUMPI,
  Opcode.JUMP,
  Opcode.JUMPDEST,
  Opcode.STOP,
  Opcode.ADD,
  Opcode.MUL,
  Opcode.POP,
  Opcode.MLOAD,
  Opcode.SUB,
  Opcode.RETURN,
]

export const stateManagerAddress: Address =
  '0x0000000000000000000000000000000000000000'

export const validBytecode: EVMBytecode = [
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('00', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('01', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('02', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('03', 'hex') },
  { opcode: Opcode.ADD, consumedBytes: undefined },
  { opcode: Opcode.MUL, consumedBytes: undefined },
  { opcode: Opcode.EQ, consumedBytes: undefined },
  { opcode: Opcode.RETURN, consumedBytes: undefined },
]

export const singleNonWhitelisted: EVMBytecode = [
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('00', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('01', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('02', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('03', 'hex') },

  { opcode: Opcode.SSTORE, consumedBytes: undefined },

  { opcode: Opcode.ADD, consumedBytes: undefined },
  { opcode: Opcode.MUL, consumedBytes: undefined },
  { opcode: Opcode.EQ, consumedBytes: undefined },
  { opcode: Opcode.RETURN, consumedBytes: undefined },
]

export const multipleNonWhitelisted: EVMBytecode = [
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('00', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('01', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('02', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('03', 'hex') },

  { opcode: Opcode.SSTORE, consumedBytes: undefined },

  { opcode: Opcode.ADD, consumedBytes: undefined },
  { opcode: Opcode.MUL, consumedBytes: undefined },
  { opcode: Opcode.EQ, consumedBytes: undefined },

  { opcode: Opcode.SLOAD, consumedBytes: undefined },

  { opcode: Opcode.RETURN, consumedBytes: undefined },
]

export const invalidBytesConsumedBytecode: EVMBytecode = [
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('00', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('01', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('02', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('03', 'hex') },
  { opcode: Opcode.ADD, consumedBytes: undefined },
  { opcode: Opcode.MUL, consumedBytes: undefined },
  { opcode: Opcode.EQ, consumedBytes: undefined },
  { opcode: Opcode.RETURN, consumedBytes: undefined },
  { opcode: Opcode.PUSH1, consumedBytes: undefined },
]

export const multipleErrors: EVMBytecode = [
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('00', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('01', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('02', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('03', 'hex') },
  { opcode: Opcode.ADD, consumedBytes: undefined },
  { opcode: Opcode.MUL, consumedBytes: undefined },
  { opcode: Opcode.EQ, consumedBytes: undefined },
  { opcode: Opcode.RETURN, consumedBytes: undefined },
  { opcode: Opcode.SLOAD, consumedBytes: undefined },
  { opcode: Opcode.PUSH1, consumedBytes: undefined },
]

export const invalidOpcode: Buffer = Buffer.from('5d', 'hex')

export const assertExecutionEqual = async (
  evmUtil: EvmIntrospectionUtil,
  firstBytecode: Buffer,
  secondBytecode: Buffer
): Promise<void> => {
  const res: ExecutionResultComparison = await evmUtil.getExecutionResultComparison(
    firstBytecode,
    secondBytecode
  )

  const firstEvmBytecode: EVMBytecode = bufferToBytecode(firstBytecode)
  const secondEvmBytecode: EVMBytecode = bufferToBytecode(secondBytecode)
  should.exist(
    res,
    `Got undefined result checking for discrepancies between \n${formatBytecode(
      firstEvmBytecode
    )}\n\nand\n\n${formatBytecode(secondEvmBytecode)}.`
  )

  res.resultsDiffer.should.equal(
    false,
    `Execution result differs between\n${formatBytecode(
      firstEvmBytecode
    )}\n\nand\n\n${formatBytecode(secondEvmBytecode)}.\n${JSON.stringify(res)}`
  )
}

export const returnNumberBytecode = (num: number = 1): EVMBytecode => {
  return [
    {
      opcode: Opcode.PUSH32,
      consumedBytes: bufferUtils.numberToBuffer(32),
    },
    {
      opcode: Opcode.PUSH1,
      consumedBytes: Buffer.from('60', 'hex'),
    },
    {
      opcode: Opcode.PUSH32,
      consumedBytes: bufferUtils.numberToBuffer(num),
    },
    {
      opcode: Opcode.PUSH1,
      consumedBytes: Buffer.from('60', 'hex'),
    },
    {
      opcode: Opcode.PUSH1,
      consumedBytes: Buffer.from('80', 'hex'),
    },
    {
      opcode: Opcode.PUSH1,
      consumedBytes: Buffer.from('40', 'hex'),
    },
    {
      opcode: Opcode.MSTORE,
      consumedBytes: undefined,
    },
    {
      opcode: Opcode.MSTORE,
      consumedBytes: undefined,
    },
    {
      opcode: Opcode.RETURN,
      consumedBytes: undefined,
    },
  ]
}

export const voidBytecode: EVMBytecode = [
  {
    opcode: Opcode.PUSH1,
    consumedBytes: Buffer.from('ff', 'hex'),
  },
]

export const memoryAndStackBytecode: EVMBytecode = [
  {
    opcode: Opcode.PUSH1,
    consumedBytes: Buffer.from('ff', 'hex'),
  },
  {
    opcode: Opcode.PUSH32,
    consumedBytes: bufferUtils.numberToBuffer(1),
  },
  {
    opcode: Opcode.PUSH1,
    consumedBytes: Buffer.from('60', 'hex'),
  },
  {
    opcode: Opcode.MSTORE,
    consumedBytes: undefined,
  },
  {
    opcode: Opcode.POP,
    consumedBytes: undefined,
  },
]
