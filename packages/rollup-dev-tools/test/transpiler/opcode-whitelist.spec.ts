import '../setup'

/* External Imports */
import { Opcode, EVMOpcode } from '@pigi/rollup-core'

/* Internal imports */
import { OpcodeWhitelist } from '../../src/types/transpiler'
import { OpcodeWhitelistImpl } from '../../src/tools/transpiler'

describe('OpcodeWhitelist', () => {
  let opcodeWhitelist: OpcodeWhitelist
  const opcodes: EVMOpcode[] = [
    Opcode.ADD,
    Opcode.SUB,
    Opcode.MLOAD,
    Opcode.BALANCE,
    Opcode.BLOCKHASH,
    Opcode.INVALID,
    Opcode.CREATE,
    Opcode.PUSH1,
    Opcode.TIMESTAMP,
    Opcode.DUP1,
    Opcode.POP,
    Opcode.SSTORE,
    Opcode.MSTORE,
  ]

  beforeEach(() => {
    opcodeWhitelist = new OpcodeWhitelistImpl(opcodes)
  })

  describe('Identifies included / excluded opcodes', () => {
    it('correctly classifies included', () => {
      for (const op of opcodes) {
        opcodeWhitelist
          .isOpcodeWhitelisted(op)
          .should.eq(true, `OP: ${op.name} should be whitelisted but was not!`)

        opcodeWhitelist
          .isOpcodeWhitelistedByCodeBuffer(op.code)
          .should.eq(
            true,
            `OP: ${op.name} should be whitelisted by code buffer but was not!`
          )

        opcodeWhitelist
          .isOpcodeWhitelistedByCodeValue(parseInt(op.code.toString('hex'), 16))
          .should.eq(
            true,
            `OP: ${op.name} should be whitelisted by code value but was not!`
          )

        opcodeWhitelist
          .isOpcodeWhitelistedByName(op.name)
          .should.eq(
            true,
            `OP: ${op.name} should be whitelisted by name but was not!`
          )
      }
    })

    it('correctly classifies excluded', () => {
      for (const op of Opcode.ALL_OP_CODES.filter(
        (x) => opcodes.indexOf(x) < 0
      )) {
        opcodeWhitelist
          .isOpcodeWhitelisted(op)
          .should.eq(false, `OP: ${op.name} should not be whitelisted but was!`)

        opcodeWhitelist
          .isOpcodeWhitelistedByCodeBuffer(op.code)
          .should.eq(
            false,
            `OP: ${op.name} should not be whitelisted by code buffer but was!`
          )

        opcodeWhitelist
          .isOpcodeWhitelistedByCodeValue(parseInt(op.code.toString('hex'), 16))
          .should.eq(
            false,
            `OP: ${op.name} should not be whitelisted by code value but was!`
          )

        opcodeWhitelist
          .isOpcodeWhitelistedByName(op.name)
          .should.eq(
            false,
            `OP: ${op.name} should not be whitelisted by name but was!`
          )
      }
    })
  })

  describe('Edge cases', () => {
    it('correctly handles invalid', () => {
      opcodeWhitelist
        .isOpcodeWhitelistedByName('derp')
        .should.eq(false, `'derp' should not be whitelisted but was!`)

      opcodeWhitelist
        .isOpcodeWhitelistedByCodeBuffer(Buffer.from('derp'))
        .should.eq(false, `'derp' buffer should not be whitelisted but was!`)

      opcodeWhitelist
        .isOpcodeWhitelistedByCodeValue(999)
        .should.eq(false, `Code value 999 should not be whitelisted but was!`)
    })

    it('correctly handles undefined / empty', () => {
      opcodeWhitelist
        .isOpcodeWhitelistedByName('')
        .should.eq(false, `'' opcode string should not be whitelisted but was!`)
      opcodeWhitelist
        .isOpcodeWhitelistedByName(undefined)
        .should.eq(
          false,
          `undefined string opcode should not be whitelisted but was!`
        )

      opcodeWhitelist
        .isOpcodeWhitelistedByCodeBuffer(Buffer.from(''))
        .should.eq(false, `'' buffer should not be whitelisted but was!`)
      opcodeWhitelist
        .isOpcodeWhitelistedByCodeBuffer(undefined)
        .should.eq(false, `undefined buffer should not be whitelisted but was!`)

      opcodeWhitelist
        .isOpcodeWhitelistedByCodeValue(undefined)
        .should.eq(
          false,
          `Code value of undefined should not be whitelisted but was!`
        )
    })
  })
})
