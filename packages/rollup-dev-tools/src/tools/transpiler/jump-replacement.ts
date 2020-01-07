import { bytecodeToBuffer, EVMBytecode, Opcode } from '@pigi/rollup-core'
import { bufferUtils, getLogger } from '@pigi/core-utils'

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

let jumpdestReplacementLength: number
export const getJumpdestReplacementBytecodeLength = (): number => {
  if (jumpdestReplacementLength === undefined) {
    jumpdestReplacementLength = bytecodeToBuffer(
      getJumpdestReplacementBytecode()
    ).length
  }
  return jumpdestReplacementLength
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
  return [
    {
      opcode: Opcode.PUSH32,
      consumedBytes: bufferUtils.numberToBuffer(footerSwitchJumpdestIndex),
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
  return [
    {
      opcode: Opcode.SWAP1,
      consumedBytes: undefined,
    },
    {
      opcode: Opcode.PUSH32,
      consumedBytes: bufferUtils.numberToBuffer(footerSwitchJumpdestIndex),
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
 * Gets the replacement bytecode for a JUMPDEST operation, given the provided
 * index of the footer switch statement JUMPDEST.
 * See: https://github.com/op-optimism/optimistic-rollup/wiki/Transpiler#jump-transpilation-approach
 * for more information on why this is necessary and how replacement occurs.
 *
 * @returns The EVMBytecode to replace JUMPDEST EVMBytecode with.
 */
export const getJumpdestReplacementBytecode = (): EVMBytecode => {
  return [
    {
      opcode: Opcode.JUMPDEST,
      consumedBytes: undefined,
    },
    {
      opcode: Opcode.POP,
      consumedBytes: undefined,
    },
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
 * @returns The JUMP switch statement bytecode.
 */
export const getJumpIndexSwitchStatementBytecode = (
  jumpdestIndexesBefore: number[],
  jumpdestIndexesAfter: number[]
): EVMBytecode => {
  const footerBytecode: EVMBytecode = [
    { opcode: Opcode.JUMPDEST, consumedBytes: undefined },
  ]
  for (let i = 0; i < jumpdestIndexesBefore.length; i++) {
    log.debug(
      `Adding bytecode to replace ${jumpdestIndexesBefore[i]} with ${jumpdestIndexesAfter[i]}`
    )
    footerBytecode.push(
      ...[
        {
          opcode: Opcode.DUP1,
          consumedBytes: undefined,
        },
        {
          opcode: Opcode.PUSH32,
          consumedBytes: bufferUtils.numberToBuffer(jumpdestIndexesBefore[i]),
        },
        {
          opcode: Opcode.EQ,
          consumedBytes: undefined,
        },
        {
          opcode: Opcode.PUSH32,
          consumedBytes: bufferUtils.numberToBuffer(jumpdestIndexesAfter[i]),
        },
        {
          opcode: Opcode.JUMPI,
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
    } else if (opcodeAndBytes.opcode === Opcode.JUMPDEST) {
      length += getJumpdestReplacementBytecodeLength()
    } else {
      length += 1 + opcodeAndBytes.opcode.programBytesConsumed
    }
  }
  return length
}
