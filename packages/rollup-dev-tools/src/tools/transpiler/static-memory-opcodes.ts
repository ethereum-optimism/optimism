/* External Imports */
import {
  Opcode,
  EVMOpcode,
  EVMOpcodeAndBytes,
  EVMBytecode,
  isValidOpcodeAndBytes,
  Address,
} from '@pigi/rollup-core'
import {
  bufToHexString,
  remove0x,
  getLogger,
  isValidHexAddress,
  hexStrToBuf,
  bufferUtils,
  BigNumber,
} from '@pigi/core-utils'
import { ADDRCONFIG } from 'dns'
import { POINT_CONVERSION_HYBRID } from 'constants'
import {
  getPUSHIntegerOp,
  getPUSHBuffer,
  staticStashMemoryInStack,
  getDUPNOp,
  staticUnstashMemoryFromStack,
  getSWAPNOp,
} from './memory-substitution'

import * as abi from 'ethereumjs-abi'

const log = getLogger(`static-memory-opcodes`)

// for the cases where calldata is fixed and known
//   export const callContractAndReturnWordToStack = (
//       addressToCall: Address,
//       methodName: string,
//   ): EVMBytecode => {
//       const totalMemory
//   }

export const storeStackElementsAsMemoryWords = (
  memoryIndexToStoreAt: number,
  numStackElementsToStore: number
): EVMBytecode => {
  let op: EVMBytecode = []
  for (let i = 0; i < numStackElementsToStore; i++) {
    op = op.concat([
      // push storage index
      getPUSHIntegerOp(
        (numStackElementsToStore - i - 1) * 32 + memoryIndexToStoreAt
      ),
      // store the stack item
      { opcode: Opcode.MSTORE, consumedBytes: undefined },
    ])
  }
  return op
}

// uses contiguous memory space starting at memoryIndexToUse
export const callContractWithStackElementsAndReturnWordToMemory = (
  address: Address,
  methodName: string,
  numStackArgumentsToPass: number,
  memoryIndexToUse: number
): [EVMBytecode, number] => {
  const methodData: Buffer = abi.methodID(methodName, [])

  const callDataMemoryLength: number =
    methodData.byteLength + 32 * numStackArgumentsToPass
  // MLOAD is 32 bytes w/ big endian, e.g. MSTOREing 4 bytes means those begin at e.g. 32 - 4 = 28 for
  // in other words, we do 1+numStackArgumentsToPass because the methodId takes up 1 whole word to store, but we pass
  const callDataMemoryOffset: number =
    memoryIndexToUse + 32 * (1 + numStackArgumentsToPass) - callDataMemoryLength
  // due to the above, this line is equivalent to memoryIndexToUse + 32.  However, leaving explicit as is hopefully more readable
  const returnDataMemoryIndex: number =
    memoryIndexToUse + callDataMemoryLength + callDataMemoryOffset
  const returnDataMemoryLength: number = 32

  const operation: EVMBytecode = [
    // Store method ID for callData
    getPUSHBuffer(methodData),
    getPUSHIntegerOp(memoryIndexToUse), // this is not callDataMemoryOffset; see comments above.
    {
      opcode: Opcode.MSTORE,
      consumedBytes: undefined,
    },
    // Store stack elements as 32-byte words for calldata
    // index + 32 because first word is methodId
    ...storeStackElementsAsMemoryWords(
      memoryIndexToUse + 32,
      numStackArgumentsToPass
    ),
    // CALL
    // ret length
    getPUSHIntegerOp(returnDataMemoryLength),
    // ret offset
    getPUSHIntegerOp(returnDataMemoryIndex),
    // calldata args length
    getPUSHIntegerOp(callDataMemoryLength),
    // calldata args offset
    getPUSHIntegerOp(callDataMemoryOffset),
    // value (0 ETH always!)
    {
      opcode: Opcode.PUSH1,
      consumedBytes: hexStrToBuf('0x00'),
    },
    // address
    getPUSHBuffer(hexStrToBuf(address)),
    // Gas
    {
      opcode: Opcode.PUSH32,
      consumedBytes: Buffer.from('00'.repeat(16) + 'ff'.repeat(16), 'hex'),
    },
    // execute the call!
    {
      opcode: Opcode.CALL,
      consumedBytes: undefined,
    },
    // POP success
    {
      opcode: Opcode.POP,
      consumedBytes: undefined,
    },
  ]

  const totalBytesMemoryUsed: number =
    callDataMemoryLength + returnDataMemoryLength
  return [operation, totalBytesMemoryUsed]
}

export const callContractWithStackElementsAndReturnWordToStack = (
  address: Address,
  methodName: string,
  numStackArgumentsToPass: number,
  memoryIndexToUse: number
): EVMBytecode => {
  let callAndReturnToMemory: EVMBytecode
  let bytesMemoryUsed: number
  ;[
    callAndReturnToMemory,
    bytesMemoryUsed,
  ] = callContractWithStackElementsAndReturnWordToMemory(
    address,
    methodName,
    numStackArgumentsToPass,
    memoryIndexToUse
  )

  // todo change return so that the ceil thing doesn't have to be done
  const returnedWordMemoryIndex: number =
    memoryIndexToUse + Math.ceil(bytesMemoryUsed / 32) * 32 - 32

  const numWordsToStash: number = Math.ceil(bytesMemoryUsed / 32)
  return [
    ...staticStashMemoryInStack(memoryIndexToUse, numWordsToStash),
    ...duplicateStackAbove(numWordsToStash, numStackArgumentsToPass),
    ...callContractWithStackElementsAndReturnWordToMemory(
      address,
      methodName,
      numStackArgumentsToPass,
      memoryIndexToUse
    )[0],
    getPUSHIntegerOp(returnedWordMemoryIndex),
    {
      opcode: Opcode.MLOAD,
      consumedBytes: undefined,
    },
    ...duplicateStackAbove(1, numWordsToStash),
    ...staticUnstashMemoryFromStack(memoryIndexToUse, numWordsToStash),
    getSWAPNOp(numWordsToStash + numStackArgumentsToPass),
    ...POPNTimes(numWordsToStash + numStackArgumentsToPass),
    { opcode: Opcode.RETURN, consumedBytes: undefined },
  ]
}

// todo add a proper test
export const duplicateStackAbove = (
  numStackElementsToIgnore: number,
  numStackElementsToDuplicate: number
): EVMBytecode => {
  let op: EVMBytecode = []
  for (let i = 0; i < numStackElementsToDuplicate; i++) {
    op.push(getDUPNOp(numStackElementsToIgnore + numStackElementsToDuplicate))
  }
  return op
}

export const POPNTimes = (numStackElementsToPop: number): EVMBytecode => {
  let op: EVMBytecode = new Array(numStackElementsToPop)
  op.fill({
    opcode: Opcode.POP,
    consumedBytes: undefined,
  })
  return op
}
