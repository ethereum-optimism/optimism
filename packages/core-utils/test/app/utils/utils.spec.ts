import '../../setup'

/* Internal Imports */
import { keccak256 } from '../../../src/app'

describe('Ethereum Utils', () => {
  describe('keccak256', () => {
    it('should return the keccak256 hash of a value', () => {
      const value = Buffer.from('1234').toString('hex')
      const hash = keccak256(value)

      const expected =
        '387a8233c96e1fc0ad5e284353276177af2186e7afa85296f106336e376669f7'

      hash.should.equal(expected)
    })

    it('should return the keccak256 of the empty string', () => {
      const value = Buffer.from('').toString('hex')
      const hash = keccak256(value)

      const expected =
        'c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470'
      hash.should.equal(expected)
    })
  })
})
