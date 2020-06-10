/* External Imports */
import {
  Address,
  bufferToBytecode,
  EVMBytecode,
  EVMOpcode,
  EVMOpcodeAndBytes,
  formatBytecode,
  Opcode,
} from '@eth-optimism/rollup-core'
import {
  bufferUtils,
  bufToHexString,
  getLogger,
  hexStrToBuf,
  Logger,
  remove0x,
} from '@eth-optimism/core-utils'
import * as abi from 'ethereumjs-abi'
import { ethers } from 'ethers'

/* Internal Imports */
import { should } from './setup'
import {
  EvmIntrospectionUtil,
  ExecutionResultComparison,
  TranspilationResult,
  SuccessfulTranspilation,
  ErroredTranspilation,
  TranspilerImpl,
  OpcodeReplacer,
} from '../src/'

import { getPUSHBuffer, getPUSHIntegerOp } from '../src'

const log: Logger = getLogger('helpers')

export const emptyBuffer: Buffer = Buffer.from('', 'hex')
export const stateManagerAddress: Address =
  '0x0000000000000000000000000000000000000000'
export const invalidOpcode: Buffer = Buffer.from('5d', 'hex')

export const whitelistedOpcodes: EVMOpcode[] = [
  Opcode.PUSH1,
  Opcode.PUSH2,
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
  Opcode.REVERT,
  Opcode.INVALID,
]

export const validBytecode: EVMBytecode = [
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('00', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('01', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('02', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('03', 'hex') },
  { opcode: Opcode.ADD, consumedBytes: undefined },
  { opcode: Opcode.MUL, consumedBytes: undefined },
  { opcode: Opcode.EQ, consumedBytes: undefined },
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

export const invalidBytesConsumedBytecodeNoReturn: EVMBytecode = [
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('00', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('01', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('02', 'hex') },
  { opcode: Opcode.PUSH1, consumedBytes: Buffer.from('03', 'hex') },
  { opcode: Opcode.ADD, consumedBytes: undefined },
  { opcode: Opcode.MUL, consumedBytes: undefined },
  { opcode: Opcode.EQ, consumedBytes: undefined },
  { opcode: Opcode.POP, consumedBytes: undefined },
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
  { opcode: Opcode.SLOAD, consumedBytes: undefined },
  { opcode: Opcode.PUSH1, consumedBytes: undefined },
]

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

export const voidBytecodeWithPushPop: EVMBytecode = [
  ...voidBytecode,
  {
    opcode: Opcode.POP,
    consumedBytes: undefined,
  },
  ...voidBytecode,
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

export const memoryDiffersBytecode: EVMBytecode = [
  {
    opcode: Opcode.PUSH1,
    consumedBytes: Buffer.from('ff', 'hex'),
  },
  {
    opcode: Opcode.PUSH32,
    consumedBytes: bufferUtils.numberToBuffer(2),
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

export const stackDiffersBytecode: EVMBytecode = [
  {
    opcode: Opcode.PUSH1,
    consumedBytes: Buffer.from('fe', 'hex'),
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

export const setupStackAndCALL = (
  gas: number,
  callTarget: Address,
  value: number,
  argOffset: number,
  argLength: number,
  retOffset: number,
  retLength: number
): EVMBytecode => {
  return [
    getPUSHIntegerOp(retLength), // ret length
    getPUSHIntegerOp(retOffset), // ret offset; must exceed 4 * 32, TODO: need to write new memory in a loop to fix this edge case?
    getPUSHIntegerOp(argLength), // args length
    getPUSHIntegerOp(argOffset), // args offset; must exceed 4 * 32, TODO: need to write new memory in a loop to fix this edge case?
    getPUSHIntegerOp(value), // value
    getPUSHBuffer(hexStrToBuf(callTarget)), // target address
    getPUSHIntegerOp(gas), // gas
    {
      opcode: Opcode.CALL,
      consumedBytes: undefined,
    },
  ]
}

export const getBytecodeCallingContractMethod = (
  address: Address,
  methodName: string,
  returnLength: number
): EVMBytecode => {
  const methodData: Buffer = abi.methodID(methodName, [])

  const mStoreArgsOffset: Buffer = hexStrToBuf('0x60')
  // last 4 bytes since method is only 4 bytes
  const actualArgsOffset: Buffer = hexStrToBuf('0x7C')
  const retOffset: Buffer = hexStrToBuf('0x80')
  const retLengthBuffer: Buffer = bufferUtils.numberToBuffer(returnLength)

  return [
    // Store free memory index pointer
    {
      opcode: Opcode.PUSH1,
      consumedBytes: hexStrToBuf('0xe0'),
    },
    {
      opcode: Opcode.PUSH1,
      consumedBytes: hexStrToBuf('0x40'),
    },
    {
      opcode: Opcode.MSTORE,
      consumedBytes: undefined,
    },
    // Store method ID
    {
      opcode: Opcode.PUSH4,
      consumedBytes: methodData,
    },
    {
      opcode: Opcode.PUSH1,
      consumedBytes: mStoreArgsOffset,
    },
    {
      opcode: Opcode.MSTORE,
      consumedBytes: undefined,
    },
    // CALL
    // ret length
    {
      opcode: Opcode.PUSH32,
      consumedBytes: retLengthBuffer,
    },
    // ret offset
    {
      opcode: Opcode.PUSH1,
      consumedBytes: retOffset,
    },
    // args length
    {
      opcode: Opcode.PUSH1,
      consumedBytes: hexStrToBuf('0x04'),
    },
    // args offset
    {
      opcode: Opcode.PUSH1,
      consumedBytes: actualArgsOffset,
    },
    // value
    {
      opcode: Opcode.PUSH1,
      consumedBytes: hexStrToBuf('0x00'),
    },
    // address
    {
      opcode: Opcode.PUSH20,
      consumedBytes: hexStrToBuf(address),
    },
    // Gas
    {
      opcode: Opcode.PUSH32,
      consumedBytes: Buffer.from('00'.repeat(16) + 'ff'.repeat(16), 'hex'),
    },
    {
      opcode: Opcode.CALL,
      consumedBytes: undefined,
    },
    // POP success
    {
      opcode: Opcode.POP,
      consumedBytes: undefined,
    },
    // RETURN
    {
      opcode: Opcode.PUSH32,
      consumedBytes: retLengthBuffer,
    },
    {
      opcode: Opcode.PUSH1,
      consumedBytes: retOffset,
    },
    {
      opcode: Opcode.RETURN,
      consumedBytes: undefined,
    },
  ]
}

export const setMemory = (toSet: Buffer): EVMBytecode => {
  const op: EVMBytecode = []
  const numWords = Math.ceil(toSet.byteLength / 32)
  for (let i = 0; i < numWords; i++) {
    op.push(
      getPUSHBuffer(toSet.slice(i * 32, (i + 1) * 32)),
      getPUSHIntegerOp(i * 32),
      {
        opcode: Opcode.MSTORE,
        consumedBytes: undefined,
      }
    )
  }
  return op
}

export const stripAuxData = (bytecode: Buffer, buildJSON: any): Buffer => {
  const auxData = buildJSON.evm.legacyAssembly['.data']['0']['.auxdata']
  const bytecodeWithoutAuxdata: Buffer = hexStrToBuf(
    bufToHexString(bytecode).split(auxData)[0]
  )
  return bytecodeWithoutAuxdata
}

export const transpileAndDeployInitcode = async (
  contractBuildJSON: any,
  constructorParams: any[],
  constructorParamsEncoding: string[],
  transpiler: TranspilerImpl,
  evmUtil: EvmIntrospectionUtil
): Promise<Buffer> => {
  const abiCoder = new ethers.utils.AbiCoder()
  const originalDeployedBytecodeLength: number = hexStrToBuf(
    contractBuildJSON.evm.deployedBytecode.object
  ).length

  const bytecodeStripped: Buffer = stripAuxData(
    hexStrToBuf(contractBuildJSON.bytecode),
    contractBuildJSON
  )
  const deployedBytecodeStripped: Buffer = stripAuxData(
    hexStrToBuf(contractBuildJSON.evm.deployedBytecode.object),
    contractBuildJSON
  )

  const initcodeTranspilationResult: TranspilationResult = transpiler.transpile(
    bytecodeStripped,
    deployedBytecodeStripped,
    originalDeployedBytecodeLength
  )
  if (!initcodeTranspilationResult.succeeded) {
    throw new Error(
      `transpilation didn't work.  Errors: ${JSON.stringify(
        (initcodeTranspilationResult as ErroredTranspilation).errors
      )}`
    )
  }
  const transpiledInitcode: Buffer = (initcodeTranspilationResult as SuccessfulTranspilation)
    .bytecode
  const encodedConstructorParams: Buffer = Buffer.from(
    remove0x(abiCoder.encode(constructorParamsEncoding, constructorParams)),
    'hex'
  )
  log.debug(
    `deploying contract with bytecode: ${transpiledInitcode.toString('hex')}`
  )

  const deployedViaInitcode = await evmUtil.deployContract(
    transpiledInitcode,
    encodedConstructorParams
  )
  const deployedViaInitcodeAddress = deployedViaInitcode.result
  return deployedViaInitcodeAddress
}

export const getMockSSTOREReplacer = (): OpcodeReplacer => {
  return {
    shouldReplaceOpcode(op: EVMOpcode): boolean {
      return op === Opcode.SSTORE
    },
    replaceIfNecessary(opcodeAndBytes: EVMOpcodeAndBytes): EVMBytecode {
      if (opcodeAndBytes.opcode === Opcode.SSTORE) {
        return [
          getPUSHIntegerOp(1),
          {
            opcode: Opcode.POP,
            consumedBytes: undefined,
          },
          getPUSHIntegerOp(2),
          {
            opcode: Opcode.POP,
            consumedBytes: undefined,
          },
          getPUSHIntegerOp(3),
          {
            opcode: Opcode.POP,
            consumedBytes: undefined,
          },
          {
            opcode: Opcode.SSTORE,
            consumedBytes: undefined,
          },
        ]
      } else {
        return [opcodeAndBytes]
      }
    },
  }
}