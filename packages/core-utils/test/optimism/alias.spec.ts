import { expect } from '../setup'
import { applyL1ToL2Alias, undoL1ToL2Alias } from '../../src'

describe('address aliasing utils', () => {
  describe('applyL1ToL2Alias', () => {
    it('should be able to apply the alias to a valid address', () => {
      expect(
        applyL1ToL2Alias('0x0000000000000000000000000000000000000000')
      ).to.equal('0x1111000000000000000000000000000000001111')
    })

    it('should be able to apply the alias even if the operation overflows', () => {
      expect(
        applyL1ToL2Alias('0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF')
      ).to.equal('0x1111000000000000000000000000000000001110')
    })

    it('should throw if the input is not a valid address', () => {
      expect(() => {
        applyL1ToL2Alias('0x1234')
      }).to.throw('not a valid address: 0x1234')
    })
  })

  describe('undoL1ToL2Alias', () => {
    it('should be able to undo the alias from a valid address', () => {
      expect(
        undoL1ToL2Alias('0x1111000000000000000000000000000000001111')
      ).to.equal('0x0000000000000000000000000000000000000000')
    })

    it('should be able to undo the alias even if the operation underflows', () => {
      expect(
        undoL1ToL2Alias('0x1111000000000000000000000000000000001110')
      ).to.equal('0xFFfFfFffFFfffFFfFFfFFFFFffFFFffffFfFFFfF')
    })

    it('should throw if the input is not a valid address', () => {
      expect(() => {
        undoL1ToL2Alias('0x1234')
      }).to.throw('not a valid address: 0x1234')
    })
  })
})
