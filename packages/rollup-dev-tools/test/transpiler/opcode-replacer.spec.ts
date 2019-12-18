
import '../setup'

/* External Imports */
import { Opcode, EVMOpcode, Address, EVMBytecode, EVMOpcodeAndBytes } from '@pigi/rollup-core'

/* Internal imports */
import { OpcodeReplacer } from '../../src/types/transpiler'
import { OpcodeReplacerImpl } from '../../src/transpiler'
import { openSync } from 'fs'
import { hexStrToBuf } from '@pigi/core-utils'

const ZERO_ADDRESS: Address = '0x0000000000000000000000000000000000000000'
describe('OpcodeReplacer', () => {
    const testConfig = {
        ADD: ['MUL'],
        SUB: ['MUL', 'MUL'],
        MUL: ['PUSH1', '0x00'],
        CALL: ['PUSH2', '0x00'],
        STATICCALL: ['PUSH2', '0x00'],
        CALLCODE: ['PUSH_STATE_MGR_ADDR']
    }

    describe('Initialization', () => {
        it('Should throw if given invalid state manager address', () => {
            //TODO
        })
    })
  
    describe('Replacement Parsing', () => {
      it('correctly parses and replaces a single opcode with another', () => {
        const cfg = {ADD: ['MUL']}
        const replacer = new OpcodeReplacerImpl(ZERO_ADDRESS, cfg)

          const replacedBytecode: EVMBytecode = replacer.getOpcodeReplacement({opcode: Opcode.ADD, consumedBytes: undefined})
          const expected: EVMBytecode = [
              {
                  opcode: Opcode.MUL,
                  consumedBytes: undefined
              }
          ]
          replacedBytecode.should.deep.equal(expected)
      })
      it('correctly parses and replaces a single opcode with two others', () => {
        const cfg = {ADD: ['MUL', 'MUL']}
        const replacer = new OpcodeReplacerImpl(ZERO_ADDRESS, cfg)

          const replacedBytecode: EVMBytecode = replacer.getOpcodeReplacement({opcode: Opcode.ADD, consumedBytes: undefined})
        const expected: EVMBytecode = [
            {
                opcode: Opcode.MUL,
                consumedBytes: undefined
            },
            {
                opcode: Opcode.MUL,
                consumedBytes: undefined
            }
        ]
        replacedBytecode.should.deep.equal(expected)
    })
    it('correctly parses and replaces a single PUSH1', () => {
        const cfg = {ADD: ['PUSH1', '0x00']}
        const replacer = new OpcodeReplacerImpl(ZERO_ADDRESS, cfg)

          const replacedBytecode: EVMBytecode = replacer.getOpcodeReplacement({opcode: Opcode.ADD, consumedBytes: undefined})        
          const expected: EVMBytecode = [
            {
                opcode: Opcode.PUSH1,
                consumedBytes: Buffer.from('00', 'hex')
            }
        ]
        replacedBytecode.should.deep.equal(expected)
    })
    it('correctly identifies when a PUSH is followed by wrong num bytes and throws', () => {
        const cfg = {ADD: ['PUSH2', '0x00']}
        const replacer = new OpcodeReplacerImpl(ZERO_ADDRESS, cfg)

          const replacedBytecode: EVMBytecode = replacer.getOpcodeReplacement({opcode: Opcode.ADD, consumedBytes: undefined}) 
        // TODO: check error once thrown
    })
    it('correctly parses and replaces a push for the state manager', () => {
        const cfg = {ADD: ['PUSH_STATE_MGR_ADDR']}
        const replacer = new OpcodeReplacerImpl(ZERO_ADDRESS, cfg)

          const replacedBytecode: EVMBytecode = replacer.getOpcodeReplacement({opcode: Opcode.ADD, consumedBytes: undefined}) 
        const expected: EVMBytecode = [
            {
                opcode: Opcode.PUSH20,
                consumedBytes: hexStrToBuf(ZERO_ADDRESS)
            }
        ]
        replacedBytecode.should.deep.equal(expected)
    })
})
})