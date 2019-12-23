import '../setup'

/* External Imports */
import {
  Opcode as Ops,
  EVMOpcode,
  Address,
  EVMBytecode,
  EVMOpcodeAndBytes,
} from '@pigi/rollup-core'

/* Internal imports */
import { OpcodeReplacer, InvalidBytesConsumedError } from '../../src/types'
import { OpcodeReplacerImpl } from '../../src/transpiler'
import { openSync } from 'fs'
import { hexStrToBuf, TestUtils } from '@pigi/core-utils'

const ZERO_ADDRESS: Address = '0x0000000000000000000000000000000000000000'
describe('OpcodeReplacer', () => {
  describe('Initialization', () => {
    it('Should throw if given invalid execution manager address', () => {
      try {
        new OpcodeReplacerImpl('0xnotAnAddr', new Map<EVMOpcode, EVMBytecode>())
      } catch (err) {
        // Success we threw an error!
        return
      }
      throw new Error('Did not throw when expected!')
    })
    it('Should not throw if given a valid execution manager address and no Opcodes Replacements Map', () => {
      try {
        new OpcodeReplacerImpl(ZERO_ADDRESS, new Map<EVMOpcode, EVMBytecode>())
      } catch (err) {
        throw new Error(
          'Should not throw with a valid execution manager address'
        )
      }
    })
  })

  describe('Replacement Parsing', () => {
    it('returns the EVMOpcode as EVMBytecode if no replacement specified', () => {
      const cfg: Map<EVMOpcode, EVMBytecode> = new Map<
        EVMOpcode,
        EVMBytecode
      >().set(Ops.ADD, [{ opcode: Ops.MUL, consumedBytes: undefined }])

      const replacer = new OpcodeReplacerImpl(ZERO_ADDRESS, cfg)

      const replacedBytecode: EVMBytecode = replacer.replaceIfNecessary({
        opcode: Ops.MUL, // different opcode
        consumedBytes: undefined,
      })
      const expected: EVMBytecode = [
        {
          opcode: Ops.MUL,
          consumedBytes: undefined,
        },
      ]
      replacedBytecode.should.deep.equal(expected)
    })

    it('correctly parses and replaces a single opcode with another', () => {
      const cfg: Map<EVMOpcode, EVMBytecode> = new Map<
        EVMOpcode,
        EVMBytecode
      >().set(Ops.ADD, [{ opcode: Ops.MUL, consumedBytes: undefined }])

      const replacer = new OpcodeReplacerImpl(ZERO_ADDRESS, cfg)

      const replacedBytecode: EVMBytecode = replacer.replaceIfNecessary({
        opcode: Ops.ADD,
        consumedBytes: undefined,
      })

      replacedBytecode.should.deep.equal(cfg.get(Ops.ADD))
    })

    it('correctly parses and replaces a single opcode with two others', () => {
      const cfg: Map<EVMOpcode, EVMBytecode> = new Map<
        EVMOpcode,
        EVMBytecode
      >().set(Ops.ADD, [
        { opcode: Ops.MUL, consumedBytes: undefined },
        { opcode: Ops.MUL, consumedBytes: undefined },
      ])
      const replacer = new OpcodeReplacerImpl(ZERO_ADDRESS, cfg)

      const replacedBytecode: EVMBytecode = replacer.replaceIfNecessary({
        opcode: Ops.ADD,
        consumedBytes: undefined,
      })

      replacedBytecode.should.deep.equal(cfg.get(Ops.ADD))
    })

    it('correctly parses and replaces a single PUSH1', () => {
      const cfg: Map<EVMOpcode, EVMBytecode> = new Map<
        EVMOpcode,
        EVMBytecode
      >().set(Ops.ADD, [
        { opcode: Ops.PUSH1, consumedBytes: hexStrToBuf('0x00') },
      ])
      const replacer = new OpcodeReplacerImpl(ZERO_ADDRESS, cfg)

      const replacedBytecode: EVMBytecode = replacer.replaceIfNecessary({
        opcode: Ops.ADD,
        consumedBytes: undefined,
      })

      replacedBytecode.should.deep.equal(cfg.get(Ops.ADD))
    })

    it('correctly identifies when a PUSH2 is followed by wrong num bytes and throws', () => {
      const cfg: Map<EVMOpcode, EVMBytecode> = new Map<
        EVMOpcode,
        EVMBytecode
      >().set(Ops.ADD, [
        { opcode: Ops.PUSH2, consumedBytes: hexStrToBuf('0x00') },
      ])
      TestUtils.assertThrows(() => {
        new OpcodeReplacerImpl(ZERO_ADDRESS, cfg)
      }, InvalidBytesConsumedError)
    })

    it('correctly parses and replaces a push for the execution manager', () => {
      const cfg: Map<EVMOpcode, EVMBytecode> = new Map<
        EVMOpcode,
        EVMBytecode
      >().set(Ops.ADD, [
        {
          opcode: Ops.PUSH20,
          consumedBytes: OpcodeReplacerImpl.EX_MGR_PLACEHOLDER,
        },
      ])
      const executionManagerAddress = ZERO_ADDRESS
      const replacer = new OpcodeReplacerImpl(executionManagerAddress, cfg)

      const replacedBytecode: EVMBytecode = replacer.replaceIfNecessary({
        opcode: Ops.ADD,
        consumedBytes: undefined,
      })
      const expected: EVMBytecode = [
        {
          opcode: Ops.PUSH20,
          consumedBytes: hexStrToBuf(executionManagerAddress),
        },
      ]
      replacedBytecode.should.deep.equal(expected)
    })
  })
})
