/* External Imports */
import {
  Opcode,
  EVMOpcode,
  EVMOpcodeAndBytes,
  EVMBytecode,
} from '@pigi/rollup-core'
import { getLogger, BigNumber } from '@pigi/core-utils'

const log = getLogger(`memory-substitution`)

/**
 * Returns a piece of bytecode which dynamically pushes a specified number of 32-byte words from memory onto the stack.
 * Assumes that the first element of the stack is the memory index to load from.
 *
 * Stack before this operation: index, X, Y, Z
 * Stack after this operation: index, memory[index], memory[index + 32], ..., memory[index + (wordsToPush * 32)] X, Y, Z
 * (Note that each MLOAD pulls 32 byte words from memory, each stack element above is a word)
 *
 * Memory after this operation: unaffected
 *
 * @param wordsToPush The number of 32-byte words from the memory to push to the stack
 * @returns Bytecode which results in the operation described above.
 */
export const pushMemoryOntoStack = (wordsToPush: number): EVMBytecode => {
  const bytecodes: EVMBytecode[] = []
  // For each word to push...
  for (let i = wordsToPush - 1; i >= 0; i--) {
    bytecodes.push([
      // duplicate the memory index which is expected as first element on stack
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
  return [].concat(...bytecodes)
}

/**
 * Returns a piece of bytecode which dynamically stores a specified number of words from the stack into memory.
 * Assumes that the first element of the stack is the memory index to store to.
 *
 * Stack before this operation: [index, wordsToStore_1, wordsToStore_2, ..., wordsToStore_n, X, Y, Z, ...]
 * Stack after this operation: [index, X, Y, Z, ...]
 *
 * Memory after this operation: memory[index: index + (32 * n)] = M_n (for n = 0 through wordsToStore)
 *
 * @param wordsToStore The number of 32-byte words from the stack to store.
 * @returns Bytecode which results in the operation described above.
 */
export const storeStackInMemory = (wordsToStore: number): EVMBytecode => {
  const bytecodes: EVMBytecode[] = []
  // For each word to store...
  for (let i = 0; i < wordsToStore; i++) {
    bytecodes.push([
      // swap the next element to store to first in stack.
      {
        opcode: Opcode.SWAP1,
        consumedBytes: undefined,
      },
      // duplicate the memory index which is not the second thing on the stack.
      {
        opcode: Opcode.DUP2,
        consumedBytes: undefined,
      },
      // ADD the max words to store, subtracting the current word we're going to store
      getPUSHIntegerOp(i * 32),
      {
        opcode: Opcode.ADD,
        consumedBytes: undefined,
      },
      // Store
      {
        opcode: Opcode.MSTORE,
        consumedBytes: undefined,
      },
    ])
  }

  return [].concat(...bytecodes)
}

/**
 * Returns a piece of bytecode which pushes a specified number of words from memory, at the specified byte index, onto the stack.
 *
 * Stack before this operation: X, Y, Z
 * Stack after this operation: memory[(wordsToPush - 1)*32 + memoryIndex], memory[(wordsToPush - 2)*32 + mmoryIndex], ..., memory[memoryIndex] X, Y, Z
 * (Note that each MLOAD pulls 32 byte words from memory, each stack element above is a word)
 *
 * Memory after this operation: unaffected
 *
 * @param memoryIndex The byte index in the memory to load from
 * @param wordsToPush The number of 32-byte words from the memory to load
 * @returns Bytecode which results in the operation described above.
 */

export const pushMemoryAtIndexOntoStack = (
  memoryIndex: number,
  wordsToPush: number
): EVMBytecode => {
  // we just use the dynamic operation, PUSHing the memoryIndex to the stack beforehand, and POPing the memoryIndex once the operation is complete.
  return [
    getPUSHIntegerOp(memoryIndex),
    ...pushMemoryOntoStack(wordsToPush),
    {
      opcode: Opcode.POP,
      consumedBytes: undefined,
    },
  ]
}

/**
 * Returns a piece of bytecode which sotres a specified number of words from the stack back into memory at the specified byte index.
 * Assumes that the first element of the stack is the memory index to load from.
 *
 * Stack before this operation: [M_wordsToStore, M_(wordsToStore-1), ... M_1, X, Y, Z, ...]
 * Stack after this operation: [X, Y, Z, ...]
 * (Note that each MSTORE puts 32 byte words from memory, each M_* stack element above is a word)
 *
 * Memory after this operation: memory[memoryIndex + 32 * n] = M_n (for n = 0 through numWords)
 *
 * @param memoryIndex The byte index in the memory to store
 * @param numWords The number of 32-byte words from the stack to store to memory
 * @returns Bytecode which results in the operation described above.
 */
export const storeStackInMemoryAtIndex = (
  memoryIndex: number,
  numWords: number
): EVMBytecode => {
  // we just use the dynamic operation, PUSHing the memoryIndex to the stack beforehand, and POPing the memoryIndex once operation is complete.
  return [
    getPUSHIntegerOp(memoryIndex),
    ...storeStackInMemory(numWords),
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
    opcode: getPUSHOpcode(numBytesToPush), // PUSH1 is 96 in decimal
    consumedBytes: toPush,
  }
}

// gets the RAW PUSHN EVMOpcode based on N.
export const getPUSHOpcode = (numBytes: number): EVMOpcode => {
  return Opcode.parseByNumber(96 + numBytes - 1) // PUSH1 is 96 in decimal
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
