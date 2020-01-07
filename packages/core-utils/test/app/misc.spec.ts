import '../setup'

/* Internal Imports */
import { sleep, remove0x, add0x, getNullString, isObject } from '../../src/app'

describe('Miscellaneous Utils', () => {
  describe('sleep', () => {
    it('should sleep for a certain number of ms', async () => {
      const start = Date.now()
      await sleep(100)
      const end = Date.now()
      const diff = end - start

      diff.should.be.greaterThan(95)
      diff.should.be.lessThan(150)
    })
  })

  describe('remove0x', () => {
    it('should remove 0x from a string', () => {
      const before = '0x123'
      const after = remove0x(before)

      after.should.equal('123')
    })

    it('should do nothing for a string without 0x', () => {
      const before = '123'
      const after = remove0x(before)

      after.should.equal(before)
    })
  })

  describe('add0x', () => {
    it('should add 0x to a string', () => {
      const before = '123'
      const after = add0x(before)

      after.should.equal('0x123')
    })

    it('should do nothing for a string with 0x', () => {
      const before = '0x123'
      const after = add0x(before)

      after.should.equal(before)
    })
  })

  describe('isObject', () => {
    it('should correctly identify an object', () => {
      const obj = { hello: 'world' }

      isObject(obj).should.be.true
    })

    it('should not identify null as an object', () => {
      isObject(null).should.be.false
    })

    it('should not identify a non-object as an object', () => {
      const notobj = 'hello'

      isObject(notobj).should.be.false
    })
  })

  describe('getNullString', () => {
    it('should return a length N string', () => {
      const nullString = getNullString(10)

      nullString.should.equal('0x0000000000')
    })
  })
})
