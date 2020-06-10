import '../setup'

/* External Imports */
import { hexStrToBuf, TestUtils, ZERO_ADDRESS } from '@eth-optimism/core-utils'
import { Opcode, EVMOpcode, EVMBytecode } from '@eth-optimism/rollup-core'

/* Internal imports */
import { OpcodeReplacer, InvalidBytesConsumedError } from '../../src/types'
import { OpcodeReplacerImpl } from '../../src/tools/transpiler'

const zeroAddrBuf: Buffer = hexStrToBuf(ZERO_ADDRESS)

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

  describe('Mandatory Replacements and Replacement Checking', () => {
    let replacer: OpcodeReplacer
    beforeEach(() => {
      replacer = new OpcodeReplacerImpl(ZERO_ADDRESS)
    })

    const assertReplaced = (r: OpcodeReplacer, opcode: EVMOpcode): void => {
      const shouldReplace = r.shouldReplaceOpcode(opcode)
      shouldReplace.should.eq(true)

      const res = r.replaceIfNecessary({
        opcode,
        consumedBytes: undefined,
      })

      const callCount: number = res.filter((x) => x.opcode === Opcode.CALL)
        .length
      const pushEMAddrCount: number = res.filter(
        (x) => x.opcode === Opcode.PUSH20 && x.consumedBytes.equals(zeroAddrBuf)
      ).length

      res.length.should.be.gt(0, 'Should return replacement!')
      callCount.should.eq(1, 'Should call EM!')
      pushEMAddrCount.should.eq(1, 'Should push EM address for call!')
      if (opcode !== Opcode.CALL) {
        const origOpcodeCount: number = res.filter(
          (x) => x.opcode === Opcode.ADDRESS
        ).length
        origOpcodeCount.should.eq(0, 'Should replace opcode!')
      }
    }

    it('replaces ADDRESS', async () => {
      assertReplaced(replacer, Opcode.ADDRESS)
    })

    it('replaces CALL', async () => {
      assertReplaced(replacer, Opcode.CALL)
    })

    it('replaces CALLER', async () => {
      assertReplaced(replacer, Opcode.CALLER)
    })

    it('replaces CREATE', async () => {
      assertReplaced(replacer, Opcode.CREATE)
    })

    it('replaces CREATE2', async () => {
      assertReplaced(replacer, Opcode.CREATE2)
    })

    it('replaces DELEGATECALL', async () => {
      assertReplaced(replacer, Opcode.DELEGATECALL)
    })

    it('replaces EXTCODECOPY', async () => {
      assertReplaced(replacer, Opcode.EXTCODECOPY)
    })

    it('replaces EXTCODEHASH', async () => {
      assertReplaced(replacer, Opcode.EXTCODEHASH)
    })

    it('replaces EXTCODESIZE', async () => {
      assertReplaced(replacer, Opcode.EXTCODESIZE)
    })

    it('replaces ORIGIN', async () => {
      assertReplaced(replacer, Opcode.ORIGIN)
    })

    it('replaces SLOAD', async () => {
      assertReplaced(replacer, Opcode.SLOAD)
    })

    it('replaces SSTORE', async () => {
      assertReplaced(replacer, Opcode.SSTORE)
    })

    it('replaces STATICCALL', async () => {
      assertReplaced(replacer, Opcode.STATICCALL)
    })

    it('replaces TIMESTAMP', async () => {
      assertReplaced(replacer, Opcode.TIMESTAMP)
    })
  })

  describe('Discretionary Replacements', () => {
    it('returns the EVMOpcode as EVMBytecode if no replacement specified', () => {
      const cfg: Map<EVMOpcode, EVMBytecode> = new Map<
        EVMOpcode,
        EVMBytecode
      >().set(Opcode.ADD, [{ opcode: Opcode.MUL, consumedBytes: undefined }])

      const replacer = new OpcodeReplacerImpl(ZERO_ADDRESS, cfg)

      const replacedBytecode: EVMBytecode = replacer.replaceIfNecessary({
        opcode: Opcode.MUL, // different opcode
        consumedBytes: undefined,
      })
      const expected: EVMBytecode = [
        {
          opcode: Opcode.MUL,
          consumedBytes: undefined,
        },
      ]
      replacedBytecode.should.deep.equal(expected)
    })

    it('correctly parses and replaces a single opcode with another', () => {
      const cfg: Map<EVMOpcode, EVMBytecode> = new Map<
        EVMOpcode,
        EVMBytecode
      >().set(Opcode.ADD, [{ opcode: Opcode.MUL, consumedBytes: undefined }])

      const replacer = new OpcodeReplacerImpl(ZERO_ADDRESS, cfg)

      const replacedBytecode: EVMBytecode = replacer.replaceIfNecessary({
        opcode: Opcode.ADD,
        consumedBytes: undefined,
      })

      replacedBytecode.should.deep.equal(cfg.get(Opcode.ADD))
    })

    it('correctly parses and replaces a single opcode with two others', () => {
      const cfg: Map<EVMOpcode, EVMBytecode> = new Map<
        EVMOpcode,
        EVMBytecode
      >().set(Opcode.ADD, [
        { opcode: Opcode.MUL, consumedBytes: undefined },
        { opcode: Opcode.MUL, consumedBytes: undefined },
      ])
      const replacer = new OpcodeReplacerImpl(ZERO_ADDRESS, cfg)

      const replacedBytecode: EVMBytecode = replacer.replaceIfNecessary({
        opcode: Opcode.ADD,
        consumedBytes: undefined,
      })

      replacedBytecode.should.deep.equal(cfg.get(Opcode.ADD))
    })

    it('correctly parses and replaces a single PUSH1', () => {
      const cfg: Map<EVMOpcode, EVMBytecode> = new Map<
        EVMOpcode,
        EVMBytecode
      >().set(Opcode.ADD, [
        { opcode: Opcode.PUSH1, consumedBytes: hexStrToBuf('0x00') },
      ])
      const replacer = new OpcodeReplacerImpl(ZERO_ADDRESS, cfg)

      const replacedBytecode: EVMBytecode = replacer.replaceIfNecessary({
        opcode: Opcode.ADD,
        consumedBytes: undefined,
      })

      replacedBytecode.should.deep.equal(cfg.get(Opcode.ADD))
    })

    it('correctly identifies when a PUSH2 is followed by wrong num bytes and throws', () => {
      const cfg: Map<EVMOpcode, EVMBytecode> = new Map<
        EVMOpcode,
        EVMBytecode
      >().set(Opcode.ADD, [
        { opcode: Opcode.PUSH2, consumedBytes: hexStrToBuf('0x00') },
      ])
      TestUtils.assertThrows(() => {
        new OpcodeReplacerImpl(ZERO_ADDRESS, cfg)
      }, InvalidBytesConsumedError)
    })

    it('correctly parses and replaces a push for the execution manager', () => {
      const cfg: Map<EVMOpcode, EVMBytecode> = new Map<
        EVMOpcode,
        EVMBytecode
      >().set(Opcode.ADD, [
        {
          opcode: Opcode.PUSH20,
          consumedBytes: OpcodeReplacerImpl.EX_MGR_PLACEHOLDER,
        },
      ])
      const executionManagerAddress = ZERO_ADDRESS
      const replacer = new OpcodeReplacerImpl(executionManagerAddress, cfg)

      const replacedBytecode: EVMBytecode = replacer.replaceIfNecessary({
        opcode: Opcode.ADD,
        consumedBytes: undefined,
      })
      const expected: EVMBytecode = [
        {
          opcode: Opcode.PUSH20,
          consumedBytes: hexStrToBuf(executionManagerAddress),
        },
      ]
      replacedBytecode.should.deep.equal(expected)
    })
  })
})
