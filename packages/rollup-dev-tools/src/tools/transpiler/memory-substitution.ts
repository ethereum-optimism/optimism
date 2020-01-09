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
  BigNumber,
} from '@pigi/core-utils'
import { ADDRCONFIG } from 'dns'
import { POINT_CONVERSION_HYBRID } from 'constants'

const log = getLogger(`memory-substitution-gen`)

/**
 * Returns a piece of bytecode which dynamically stashes a specified number of words from memory onto the stack.
 * Assumes that the first element of the stack is the memory index to load from.
 *
 * Stack before this operation: index, X, Y, Z
 * Stack after this operation: index, memory[(wordsToStash - 1)*32 + index], memory[(wordsToStash - 2)*32 + index], ..., memory[index] X, Y, Z
 * (Note that each MLOAD pulls 32 byte words from memory, each stack element above is a word)
 *
 * Memory after this operation: unaffected
 *
 * @param wordsToStash The number of 32-byte words from the memory to stash
 * @returns Btyecode which results in the stash operation described above.
 */
export const dynamicStashMemoryInStack = (
  wordsToStash: number
): EVMBytecode => {
  let stashOperation: EVMBytecode = []
  // For each word to stash...
  for (let i = 0; i < wordsToStash; i++) {
    stashOperation = stashOperation.concat([
      // duplicate the mmory index which is expected as first element on stack
      {
        opcode: Opcode.DUP1,
        consumedBytes: undefined,
      },
      // ADD the word number to the memory index to get index for this word
      getPUSHIntegerOp(i * 32), // 32 because memory is byte indexed but loaded in 32 byte words
      {
        opcode: Opcode.ADD,
        consumedBytes: undefined,
      },
      // MLOAD the word from memory
      {
        opcode: Opcode.MLOAD,
        consumedBytes: undefined,
      },
      // Swap the loaded word so that memory index is first on stack for next iteration
      {
        opcode: Opcode.SWAP1,
        consumedBytes: undefined,
      },
    ])
  }
  return stashOperation
}

/**
 * Returns a piece of bytecode which dynamically unstashes a specified number of words from the stack back into memory.
 * Assumes that the first element of the stack is the memory index to load from.
 *
 * Stack before this operation: [index, M_wordsToUnstash, M_(wordsToUnstash-1), ... M_1, X, Y, Z, ...]
 * Stack after this operation: [index, X, Y, Z, ...]
 * (Note that each MSTORE puts 32 byte words from memory, each M_* stack element above is a word)
 *
 * Memory after this operation: memory[index + 32 * n] = M_n (for n = 0 through wordsToUnstash)
 *
 * @param wordsToStash The number of 32-byte words from the stack to unstash
 * @returns Btyecode which results in the unstash operation described above.
 */
export const dynamicUnstashMemoryFromStack = (
  wordsToUnstash: number
): EVMBytecode => {
  let unstashOperation: EVMBytecode = []
  // The only trickiness here is that the stash operation for memory = A, B, C --> stack = C, B, A
  // So we store in reeverse order, starting with index + wordsToUnstash and work back to index + 0.

  // For each word to unstash...
  for (let i = 0; i < wordsToUnstash; i++) {
    unstashOperation = unstashOperation.concat([
      // duplicate the memory index, expected as first thing on the stack.
      {
        opcode: Opcode.DUP1,
        consumedBytes: undefined,
      },
      // ADD the max words to unstash
      getPUSHIntegerOp((wordsToUnstash - 1) * 32),
      {
        opcode: Opcode.ADD,
        consumedBytes: undefined,
      },
      // SUBtract the current word we're going to unstash
      getPUSHIntegerOp(i * 32),
      {
        opcode: Opcode.SWAP1,
        consumedBytes: undefined,
      },
      {
        opcode: Opcode.SUB,
        consumedBytes: undefined,
      },
      // DUP the word we're going to unstash from the stack.
      // Stack looks like: [index + numWords - i, index, C, B, A, ...]
      // So the index of C is 3, index of B is 4, etc...
      getDUPNOp(3 + i),
      // Swap so stack is now [index, wordToUnstash]
      {
        opcode: Opcode.SWAP1,
        consumedBytes: undefined,
      },
      // Store
      {
        opcode: Opcode.MSTORE,
        consumedBytes: undefined,
      },
    ])
  }
  // Now all that's left is to CLEANUP THE STACK
  // For dynamic index, we don't want to delete the index, in case it's needed for other operations.
  // Stack should look like it started, [index, C, B, A, ...]  so this SWAP makes it [A, C, B, index, ...]
  unstashOperation.push(getSWAPNOp(wordsToUnstash))
  // Now that the unstashed words are at the front of the stack, POP them all (numWords times)
  unstashOperation = unstashOperation.concat(
    new Array<EVMOpcodeAndBytes>(wordsToUnstash).fill({
      opcode: Opcode.POP,
      consumedBytes: undefined,
    })
  )
  return unstashOperation
}

/**
 * Returns a piece of bytecode which stashes a specified number of words from memory, at the specified byte index, onto the stack.
 *
 * Stack before this operation: X, Y, Z
 * Stack after this operation: memory[(wordsToStash - 1)*32 + memoryIndex], memory[(wordsToStash - 2)*32 + mmoryIndex], ..., memory[memoryIndex] X, Y, Z
 * (Note that each MLOAD pulls 32 byte words from memory, each stack element above is a word)
 *
 * Memory after this operation: unaffected
 *
 * @param memoryIndex The byte index in the memory to stash
 * @param numWords The number of 32-byte words from the memory to stash
 * @returns Btyecode which results in the stash operation described above.
 */

export const staticStashMemoryInStack = (
  memoryIndex: number,
  numWords: number
): EVMBytecode => {
  // we just use the dynamic operation, PUSHing the memoryIndex to the stack beforehand, and POPing the memoryIndex once the operation is complete.
  return [
    getPUSHIntegerOp(memoryIndex),
    ...dynamicStashMemoryInStack(numWords),
    {
      opcode: Opcode.POP,
      consumedBytes: undefined,
    },
  ]
}

/**
 * Returns a piece of bytecode which unstashes a specified number of words from the stack back into memory at the specified byte index.
 * Assumes that the first element of the stack is the memory index to load from.
 *
 * Stack before this operation: [M_wordsToUnstash, M_(wordsToUnstash-1), ... M_1, X, Y, Z, ...]
 * Stack after this operation: [X, Y, Z, ...]
 * (Note that each MSTORE puts 32 byte words from memory, each M_* stack element above is a word)
 *
 * Memory after this operation: memory[memoryIndex + 32 * n] = M_n (for n = 0 through numWords)
 *
 * @param memoryIndex The byte index in the memory to unstash
 * @param numWords The number of 32-byte words from the stack to unstash to memory
 * @returns Btyecode which results in the unstash operation described above.
 */
export const staticUnstashMemoryFromStack = (
  memoryIndex: number,
  numWords: number
): EVMBytecode => {
  // we just use the dynamic operation, PUSHing the memoryIndex to the stack beforehand, and POPing the memoryIndex once operation is complete.
  return [
    getPUSHIntegerOp(memoryIndex),
    ...dynamicUnstashMemoryFromStack(numWords),
    {
      opcode: Opcode.POP,
      consumedBytes: undefined,
    },
  ]
}

// constructs an operation which PUSHes the given integer to the stack.
export const getPUSHIntegerOp = (theInt: number): EVMOpcodeAndBytes => {
  const intAsBuffer: Buffer = new BigNumber(theInt).toBuffer()
  return getPUSHBuffer(intAsBuffer)
}

// Returns a PUSH operation for the given bytes
export const getPUSHBuffer = (toPush: Buffer): EVMOpcodeAndBytes => {
  const numBytesToPush: number = toPush.byteLength
  // TODO: error if length exceeds 32
  return {
    opcode: Opcode.parseByNumber(96 + numBytesToPush - 1), // PUSH1 is 96 in decimal
    consumedBytes: toPush,
  }
}

// returns DUPN operation for the specified N.
export const getDUPNOp = (indexToDUP: number): EVMOpcodeAndBytes => {
  // TODO: error if index is too big
  return {
    opcode: Opcode.parseByNumber(128 + indexToDUP - 1),
    consumedBytes: undefined,
  }
}
// returns SWAPN operation for the specified N.
export const getSWAPNOp = (indexToSWAP: number): EVMOpcodeAndBytes => {
  // TODO: error if index is too big
  return {
    opcode: Opcode.parseByNumber(144 + indexToSWAP - 1),
    consumedBytes: undefined,
  }
}
