/* External Imports */
import { Opcode, EVMBytecode, Address } from '@pigi/rollup-core'
import { getLogger, hexStrToBuf } from '@pigi/core-utils'
import {
  getPUSHIntegerOp,
  getPUSHBuffer,
  pushMemoryAtIndexOntoStack,
  getDUPNOp,
  storeStackInMemoryAtIndex,
  getSWAPNOp,
} from './memory-substitution'

import * as abi from 'ethereumjs-abi'

const log = getLogger(`static-memory-opcodes`)
/**
 * Stores the first `numWords` elements on the stack to memory at the specified index.
 *
 * Used to pass stack params into the Execution manager as calldata.
 *
 * @param numStackElementsToStore The number of stack elements to put in the memory
 * @param memoryIndexToStoreAt The byte index in the memory to store the stack elements to.
 * @returns Btyecode which results in the storage operation described above.
 */
export const storeStackElementsAsMemoryWords = (
  numStackElementsToStore: number,
  memoryIndexToStoreAt: number = 0
): EVMBytecode => {
  let op: EVMBytecode = []
  for (let i = 0; i < numStackElementsToStore; i++) {
    op = op.concat([
      // push storage index
      getPUSHIntegerOp(i * 32 + memoryIndexToStoreAt),
      // store the stack item
      { opcode: Opcode.MSTORE, consumedBytes: undefined },
    ])
  }
  return op
}

/**
 * Uses the contiguous memory space starting at the specified index to:
 * 1. Store the methodId to pass as calldata
 * 2. Store the stack elements to pass as calldata
 * 3. Have a single word of return data be stored at the index proceeding the above.
 *
 *
 * @param address The address to call.
 * @param methodName The human readable name of the ABI method to call
 * @param numStackArgumentsToPass The number of stack elements to pass to the address as calldata
 * @param memoryIndexToUse The memory index to use a contiguous range of
 * @returns Btyecode which results in the store-and-call operation described above.
 * @returns The total number of bytes of storage which will be used and need to be stashed beforehand if memory is to be unaffected.
 */

export const callContractWithStackElementsAndReturnWordToMemory = (
  address: Address,
  methodName: string,
  numStackArgumentsToPass: number,
  memoryIndexToUse: number = 0
): EVMBytecode => {
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

  return [
    // Store method ID for callData
    getPUSHBuffer(methodData),
    getPUSHIntegerOp(memoryIndexToUse), // this is not callDataMemoryOffset; see comments above.
    {
      opcode: Opcode.MSTORE,
      consumedBytes: undefined,
    },
    // Store the stack elements as 32-byte words for calldata.
    // index + 32 because first word is methodId
    ...storeStackElementsAsMemoryWords(
      numStackArgumentsToPass,
      memoryIndexToUse + 32
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
}

/**
 * Uses the contiguous memory space starting at the specified index to:
 * 1. Stash the original memory space into the stack to be replaced after execution.
 * 2. Store the methodId to pass as calldata
 * 3. Store the stack elements to pass as calldata
 * 4. Have a single word of return data be stored at the index proceeding the above.
 * 5. Load the returned word from memory into the stack.
 * 6. Replace the original memory by unstashing.
 *
 *
 * @param address The address to call.
 * @param methodName The human readable name of the ABI method to call
 * @param numStackArgumentsToPass The number of stack elements to pass to the address as calldata
 * @param memoryIndexToUse The memory index to use a contiguous range of
 * @returns Btyecode which results in the 32 byte word of return being pushed to the stack with original memory intact.
 */

export const callContractWithStackElementsAndReturnWordToStack = (
  address: Address,
  methodName: string,
  numStackArgumentsToPass: number,
  memoryIndexToUse: number = 0
): EVMBytecode => {
  // 1 word for method Id, 1 word for each stack argument, 1 word for return
  const numWordsToStash: number = 1 + numStackArgumentsToPass + 1 //Math.ceil(bytesMemoryUsed / 32)

  // ad 1 word for method Id, 1 word for each stack argument, and then the immediately following index will be the return val
  const returnedWordMemoryIndex: number = 32 * (1 + numStackArgumentsToPass)

  return [
    // Based on the contiguous memory space we expect to utilize, stash the original memory so it can be recovered.
    ...pushMemoryAtIndexOntoStack(memoryIndexToUse, numWordsToStash),
    // Now that the stashed memory is first on the stack, recover the original stack elements we expected to consume/pass to execution manager
    ...duplicateStackAt(numWordsToStash, numStackArgumentsToPass),
    // Do the call, with the returned word being put into memory.
    ...callContractWithStackElementsAndReturnWordToMemory(
      address,
      methodName,
      numStackArgumentsToPass,
      memoryIndexToUse
    ),
    // MLOAD the returned word into the stack
    getPUSHIntegerOp(returnedWordMemoryIndex),
    {
      opcode: Opcode.MLOAD,
      consumedBytes: undefined,
    },
    // Now that the returned value is first thing on stack, duplicate the stashed old memory so they're first on the stack.
    ...duplicateStackAt(1, numWordsToStash),
    // Now that stack is prepared, unstash the memory to its original state
    ...storeStackInMemoryAtIndex(memoryIndexToUse, numWordsToStash),
    // The above duplications need to be eliminated, but the returned word needs to be maintained.  SWAP it out of the way.
    getSWAPNOp(numWordsToStash + numStackArgumentsToPass),
    // POP the extra elements that came from the above duplications
    ...POPNTimes(numWordsToStash + numStackArgumentsToPass),
  ]
}

export const duplicateStackAt = (
  numStackElementsToIgnore: number,
  numStackElementsToDuplicate: number
): EVMBytecode => {
  // TODO: error if N is too high to DUPN
  const op: EVMBytecode = []
  for (let i = 0; i < numStackElementsToDuplicate; i++) {
    op.push(getDUPNOp(numStackElementsToIgnore + numStackElementsToDuplicate))
  }
  return op
}

export const POPNTimes = (numStackElementsToPop: number): EVMBytecode => {
  const op: EVMBytecode = new Array(numStackElementsToPop)
  op.fill({
    opcode: Opcode.POP,
    consumedBytes: undefined,
  })
  return op
}
