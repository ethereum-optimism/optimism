import { bytecodeToBuffer, EVMBytecode, Opcode } from '@pigi/rollup-core'
import { bufferUtils, getLogger } from '@pigi/core-utils'
import { getPUSHOpcode } from './helpers'

const log = getLogger('jump-replacement')

let jumpReplacementLength: number
export const getJumpReplacementBytecodeLength = (): number => {
  if (jumpReplacementLength === undefined) {
    jumpReplacementLength = bytecodeToBuffer(getJumpReplacementBytecode(0))
      .length
  }
  return jumpReplacementLength
}

let jumpiReplacementLength: number
export const getJumpiReplacementBytecodeLength = (): number => {
  if (jumpiReplacementLength === undefined) {
    jumpiReplacementLength = bytecodeToBuffer(getJumpiReplacementBytecode(0))
      .length
  }
  return jumpiReplacementLength
}

/**
 * Gets the replacement bytecode for a JUMP operation, given the provided
 * index of the footer switch statement JUMPDEST.
 * See: https://github.com/op-optimism/optimistic-rollup/wiki/Transpiler#jump-transpilation-approach
 * for more information on why this is necessary and how replacement occurs.
 *
 * @param footerSwitchJumpdestIndex The index of the footer JUMPDEST.
 * @returns The EVMBytecode to replace JUMP EVMBytecode with.
 */
export const getJumpReplacementBytecode = (
  footerSwitchJumpdestIndex: number
): EVMBytecode => {
  const indexBuffer: Buffer = bufferUtils.numberToBuffer(
    footerSwitchJumpdestIndex
  )
  return [
    {
      opcode: getPUSHOpcode(indexBuffer.length),
      consumedBytes: indexBuffer,
    },
    {
      opcode: Opcode.JUMP,
      consumedBytes: undefined,
    },
  ]
}

/**
 * Gets the replacement bytecode for a JUMPI operation, given the provided
 * index of the footer switch statement JUMPDEST.
 * See: https://github.com/op-optimism/optimistic-rollup/wiki/Transpiler#jump-transpilation-approach
 * for more information on why this is necessary and how replacement occurs.
 *
 * @param footerSwitchJumpdestIndex The index of the footer JUMPDEST.
 * @returns The EVMBytecode to replace JUMPI EVMBytecode with.
 */
export const getJumpiReplacementBytecode = (
  footerSwitchJumpdestIndex: number
): EVMBytecode => {
  const indexBuffer: Buffer = bufferUtils.numberToBuffer(
    footerSwitchJumpdestIndex
  )
  return [
    {
      opcode: Opcode.SWAP1,
      consumedBytes: undefined,
    },
    {
      opcode: getPUSHOpcode(indexBuffer.length),
      consumedBytes: indexBuffer,
    },
    {
      opcode: Opcode.JUMPI,
      consumedBytes: undefined,
    },
    {
      opcode: Opcode.POP,
      consumedBytes: undefined,
    },
  ]
}

/**
 * Gets the success jumpdest for the footer switch statement. This will be jumped to when the
 * switch statement finds a match. It is responsible for getting rid of extra stack arguments
 * that the footer switch statement adds.
 *
 * @returns The success bytecode.
 */
const getJumpIndexSwitchStatementSuccessJumpdestBytecode = (): EVMBytecode => {
  return [
    // This JUMPDEST is hit on successful switch match
    { opcode: Opcode.JUMPDEST, consumedBytes: undefined },
    // Swaps the duped pre-transpilation JUMPDEST with the post-transpilation JUMPDEST
    { opcode: Opcode.SWAP1, consumedBytes: undefined },
    // Pops the pre-transpilation JUMPDEST
    { opcode: Opcode.POP, consumedBytes: undefined },
    // Jumps to the post-transpilation JUMPDEST
    { opcode: Opcode.JUMP, consumedBytes: undefined },
  ]
}

/**
 * Gets the EVMBytecode to read a pre-transpilation JUMPDEST index off of the stack and
 * JUMP to the associated post-transpilation JUMPDEST.
 * See: https://github.com/op-optimism/optimistic-rollup/wiki/Transpiler#jump-transpilation-approach
 * for more information on why this is necessary and how replacement occurs.
 *
 * @param jumpdestIndexesBefore The array of of pre-transpilation JUMPDEST indexes.
 * @param jumpdestIndexesAfter The array of of post-transpilation JUMPDEST indexes.
 * @param indexOfThisBlock The index in the bytecode that this block will be at.
 * @returns The JUMP switch statement bytecode.
 */
export const getJumpIndexSwitchStatementBytecode = (
  jumpdestIndexesBefore: number[],
  jumpdestIndexesAfter: number[],
  indexOfThisBlock: number
): EVMBytecode => {
  const successJumpIndex: Buffer = bufferUtils.numberToBuffer(indexOfThisBlock)

  const footerBytecode: EVMBytecode = [
    ...getJumpIndexSwitchStatementSuccessJumpdestBytecode(),
    // Switch statement jumpdest
    { opcode: Opcode.JUMPDEST, consumedBytes: undefined },
  ]
  for (let i = 0; i < jumpdestIndexesBefore.length; i++) {
    log.debug(
      `Adding bytecode to replace ${jumpdestIndexesBefore[i]} with ${jumpdestIndexesAfter[i]}`
    )
    const beforeBuffer: Buffer = bufferUtils.numberToBuffer(
      jumpdestIndexesBefore[i]
    )
    const afterBuffer: Buffer = bufferUtils.numberToBuffer(
      jumpdestIndexesAfter[i]
    )
    footerBytecode.push(
      ...[
        {
          opcode: Opcode.DUP1,
          consumedBytes: undefined,
        },
        {
          opcode: getPUSHOpcode(beforeBuffer.length),
          consumedBytes: beforeBuffer,
        },
        {
          opcode: Opcode.EQ,
          consumedBytes: undefined,
        },
        {
          // push ACTUAL jumpdest
          opcode: getPUSHOpcode(afterBuffer.length),
          consumedBytes: afterBuffer,
        },
        {
          // swap actual jumpdest with EQ result so stack is [eq result, actual jumpdest, duped compare jumpdest, ...]
          opcode: Opcode.SWAP1,
          consumedBytes: undefined,
        },
        {
          // push loop exit jumpdest
          opcode: getPUSHOpcode(successJumpIndex.length),
          consumedBytes: successJumpIndex,
        },
        {
          opcode: Opcode.JUMPI,
          consumedBytes: undefined,
        },
        // pop ACTUAL jumpdest because this is not a match
        {
          opcode: Opcode.POP,
          consumedBytes: undefined,
        },
      ]
    )
  }
  // If pre-transpile JUMPDEST index is not found, revert.
  footerBytecode.push({ opcode: Opcode.REVERT, consumedBytes: undefined })
  return footerBytecode
}

/**
 * Gets the expected index of the successful switch match JUMPDEST in the footer switch statement.
 *
 * @param transpiledBytecode The transpiled bytecode in question.
 * @returns The expected index of the JUMPDEST for successful footer switch matches.
 */
const getFooterSwitchStatementSuccessJumpdestIndex = (
  transpiledBytecode: EVMBytecode
): number => {
  let length: number = 0
  for (const opcodeAndBytes of transpiledBytecode) {
    if (opcodeAndBytes.opcode === Opcode.JUMP) {
      length += getJumpReplacementBytecodeLength()
    } else if (opcodeAndBytes.opcode === Opcode.JUMPI) {
      length += getJumpiReplacementBytecodeLength()
    } else {
      length += 1 + opcodeAndBytes.opcode.programBytesConsumed
    }
  }
  return length
}

/**
 * Gets the expected index of the footer JUMP switch statement, given EVMBytecode
 * that will *only* change by replacing JUMP, JUMPI, and JUMPDEST with the appropriate
 * EVMBytecode.
 *
 * @param bytecode The bytecode in question.
 * @returns The expected index of the JUMPDEST for the footer JUMP switch statement.
 */
export const getExpectedFooterSwitchStatementJumpdestIndex = (
  bytecode: EVMBytecode
): number => {
  let length: number = 0
  for (const opcodeAndBytes of bytecode) {
    if (opcodeAndBytes.opcode === Opcode.JUMP) {
      length += getJumpReplacementBytecodeLength()
    } else if (opcodeAndBytes.opcode === Opcode.JUMPI) {
      length += getJumpiReplacementBytecodeLength()
    } else {
      length += 1 + opcodeAndBytes.opcode.programBytesConsumed
    }
  }
  length += bytecodeToBuffer(
    getJumpIndexSwitchStatementSuccessJumpdestBytecode()
  ).length
  return length
}
