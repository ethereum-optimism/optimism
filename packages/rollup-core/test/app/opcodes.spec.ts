import '../setup'
import * as assert from 'assert'

import { Opcode } from '../../src/types'

describe('Opcode', () => {
  describe('Parsing', () => {
    it('parses properly by opcode name', () => {
      for (const opcode of Opcode.ALL_OP_CODES) {
        opcode.should.eql(
          Opcode.parseByName(opcode.name),
          `Could not parse opcode by name '${opcode.name}' from opcode.name!`
        )
      }
    })
    it('parses properly by opcode code buffer', () => {
      for (const opcode of Opcode.ALL_OP_CODES) {
        opcode.should.eql(
          Opcode.parseByCode(opcode.code),
          `Could not parse opcode ${
            opcode.name
          } by code buffer '${opcode.code.toString('hex')}' from opcode.code!`
        )
      }
    })
    it('parses properly by opcode number', () => {
      for (const opcode of Opcode.ALL_OP_CODES) {
        const opNum: number = parseInt(opcode.code.toString('hex'), 16)
        opcode.should.eql(
          Opcode.parseByNumber(opNum),
          `Could not parse opcode ${opcode.name} by code number [${opNum}]!`
        )
      }
    })
    it('returns undfined when not parseable', () => {
      assert(
        Opcode.parseByNumber(999) === undefined,
        'Opcode 999 parsed when it should not be!'
      )
      assert(
        Opcode.parseByNumber(undefined) === undefined,
        'Opcode undefined parsed when it should not be!'
      )

      assert(
        Opcode.parseByName(undefined) === undefined,
        'Opcode with undefined name parsed when it should not be!'
      )
      assert(
        Opcode.parseByName('') === undefined,
        'Opcode with empty name parsed when it should not be!'
      )
      assert(
        Opcode.parseByName('derp') === undefined,
        'Opcode with name "derp" parsed when it should not be!'
      )

      assert(
        Opcode.parseByCode(undefined) === undefined,
        'Opcode with undefined code parsed when it should not be!'
      )
      assert(
        Opcode.parseByCode(Buffer.from('')) === undefined,
        'Opcode with empty code parsed when it should not be!'
      )
      assert(
        Opcode.parseByCode(Buffer.from('derp')) === undefined,
        'Opcode with "derp" code parsed when it should not be!'
      )
    })
  })
})
