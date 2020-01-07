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
    BigNumber
  } from '@pigi/core-utils'
import { ADDRCONFIG } from 'dns'
import { POINT_CONVERSION_HYBRID } from 'constants'

const log = getLogger(`memory-substitution-gen`)

  // assumes the first element on the stack is the memory offset to start pulling from
  export const dynamicStashMemoryInStack = function(
      wordsToStash: number
  ): EVMBytecode {
    let stashOperation: EVMBytecode = []
    for (let i = 0; i < wordsToStash; i++) {
        stashOperation = stashOperation.concat([
            {
                opcode: Opcode.DUP1,
                consumedBytes: undefined
            },
            getPUSHIntegerOp(i * 32),
            {
                opcode: Opcode.ADD,
                consumedBytes: undefined
            },
            {
                opcode: Opcode.MLOAD,
                consumedBytes: undefined
            },
            {
                opcode: Opcode.SWAP1,
                consumedBytes: undefined
            }
        ])
    }
    return stashOperation
  }

  // assumes the first element on the stack is the memory offset to start pushing to
  export const dynamicUnstashMemoryFromStack = function (
      wordsToUnstash: number
  ): EVMBytecode {
    let unstashOperation: EVMBytecode = []
    for (let i = 0; i < wordsToUnstash; i++) {
        unstashOperation = unstashOperation.concat([
            {
                opcode: Opcode.DUP1,
                consumedBytes: undefined
            },
            getPUSHIntegerOp((wordsToUnstash - 1) * 32),
            {
                opcode: Opcode.ADD,
                consumedBytes: undefined
            },
            getPUSHIntegerOp(i * 32),
            {
                opcode: Opcode.SWAP1,
                consumedBytes: undefined
            },
            {
                opcode: Opcode.SUB,
                consumedBytes: undefined
            },
            getDUPNOp(3 + i),
            {
                opcode: Opcode.SWAP1,
                consumedBytes: undefined
            },
            {
                opcode: Opcode.MSTORE,
                consumedBytes: undefined
            }
        ])
    }
    return unstashOperation
  }

  export const staticStashMemoryInStack = function(memoryIndex: number, numWords: number): EVMBytecode {
      return [
          getPUSHIntegerOp(memoryIndex),
          ...dynamicStashMemoryInStack(numWords),
          {
              opcode: Opcode.POP,
              consumedBytes: undefined
          }
      ]
  }

  export const staticUnstashMemoryFromStack = function(memoryIndex: number, numWords: number): EVMBytecode {
    return [
        getPUSHIntegerOp(memoryIndex),
        ...dynamicUnstashMemoryFromStack(numWords),
        {
            opcode: Opcode.POP,
            consumedBytes: undefined
        }
    ]
}


  export const getPUSHIntegerOp = function(theInt: number): EVMOpcodeAndBytes {
    const intAsBuffer: Buffer = new BigNumber(theInt).toBuffer()
    const numBytesToPush = intAsBuffer.length
    // todo error if length exceeds 32
    return {
        opcode: Opcode.parseByNumber(96 + numBytesToPush - 1), // PUSH1 is 96 in decimal
        consumedBytes: intAsBuffer
    }
  }

  export const getDUPNOp = function(indexToDUP: number): EVMOpcodeAndBytes {
      // todo error if index is too big
      return {
          opcode: Opcode.parseByNumber(128 + indexToDUP - 1),
          consumedBytes: undefined
      }
  }