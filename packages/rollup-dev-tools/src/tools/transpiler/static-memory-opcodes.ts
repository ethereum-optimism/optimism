/* External Imports */
import { Opcode, EVMBytecode, Address } from '@eth-optimism/rollup-core'
import { getLogger, hexStrToBuf } from '@eth-optimism/core-utils'
import * as abi from 'ethereumjs-abi'

/* Internal Imports */
import {
  getPUSHIntegerOp,
  getPUSHBuffer,
  pushMemoryAtIndexOntoStack,
  storeStackInMemoryAtIndex,
  getSWAPNOp,
  duplicateStackAt,
  POPNTimes,
  storeStackElementsAsMemoryWords,
} from './helpers'

const log = getLogger(`static-memory-opcodes`)

export const ovmADDRESSName: string = 'ovmADDRESS'
export const ovmCALLERName: string = 'ovmCALLER'
export const ovmEXTCODEHASHName: string = 'ovmEXTCODEHASH'
export const ovmEXTCODESIZEName: string = 'ovmEXTCODESIZE'
export const ovmORIGINName: string = 'ovmORIGIN'
export const ovmSLOADName: string = 'ovmSLOAD'
export const ovmSSTOREName: string = 'ovmSSTORE'
export const ovmTIMESTAMPName: string = 'ovmTIMESTAMP'

export const getADDRESSReplacement = (
  executionManagerAddress: Address,
  ovmADDRESSFunctionName: string = ovmADDRESSName
): EVMBytecode => {
  return callContractWithStackElementsAndReturnWordToStack(
    executionManagerAddress,
    ovmADDRESSFunctionName,
    0,
    1
  )
}

export const getCALLERReplacement = (
  executionManagerAddress: Address,
  ovmCALLERFunctionName: string = ovmCALLERName
): EVMBytecode => {
  return callContractWithStackElementsAndReturnWordToStack(
    executionManagerAddress,
    ovmCALLERFunctionName,
    0,
    1
  )
}

export const getEXTCODEHASHReplacement = (
  executionManagerAddress: Address,
  ovmEXTCODEHASHFunctionName: string = ovmEXTCODEHASHName
): EVMBytecode => {
  return callContractWithStackElementsAndReturnWordToStack(
    executionManagerAddress,
    ovmEXTCODEHASHFunctionName,
    1,
    1
  )
}

export const getEXTCODESIZEReplacement = (
  executionManagerAddress: Address,
  ovmEXTCODESIZEFunctionName: string = ovmEXTCODESIZEName
): EVMBytecode => {
  return callContractWithStackElementsAndReturnWordToStack(
    executionManagerAddress,
    ovmEXTCODESIZEFunctionName,
    1,
    1
  )
}

export const getORIGINReplacement = (
  executionManagerAddress: Address,
  ovmORIGINFunctionName: string = ovmORIGINName
): EVMBytecode => {
  return callContractWithStackElementsAndReturnWordToStack(
    executionManagerAddress,
    ovmORIGINFunctionName,
    0,
    1
  )
}

export const getSLOADReplacement = (
  executionManagerAddress: Address,
  ovmSLOADFunctionName: string = ovmSLOADName
): EVMBytecode => {
  return callContractWithStackElementsAndReturnWordToStack(
    executionManagerAddress,
    ovmSLOADFunctionName,
    1,
    1
  )
}

export const getSSTOREReplacement = (
  executionManagerAddress: Address,
  ovmSSTOREFunctionName: string = ovmSSTOREName
): EVMBytecode => {
  return callContractWithStackElementsAndReturnWordToStack(
    executionManagerAddress,
    ovmSSTOREFunctionName,
    2,
    0
  )
}

export const getTIMESTAMPReplacement = (
  executionManagerAddress: Address,
  ovmTIMESTAMPFunctionName: string = ovmTIMESTAMPName
): EVMBytecode => {
  return callContractWithStackElementsAndReturnWordToStack(
    executionManagerAddress,
    ovmTIMESTAMPFunctionName,
    0,
    1
  )
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
  numStackArgumentsToPass: number = 0,
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
  numStackValuesReturned: 1 | 0,
  memoryIndexToUse: number = 0
): EVMBytecode => {
  // 1 word for method Id, 1 word for each stack argument, 1 word for return
  const numWordsToStash: number = 1 + numStackArgumentsToPass + 1 //Math.ceil(bytesMemoryUsed / 32)

  // ad 1 word for method Id, 1 word for each stack argument, and then the immediately following index will be the return val
  const returnedWordMemoryIndex: number = 32 * (1 + numStackArgumentsToPass)

  const op = [
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
  if (numStackValuesReturned === 0) {
    // if we don't care about a return value just pop whatever was randomly grabbed from memory
    op.push({
      opcode: Opcode.POP,
      consumedBytes: undefined,
    })
  }
  return op
}
