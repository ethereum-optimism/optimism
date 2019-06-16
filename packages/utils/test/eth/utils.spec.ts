import '../setup'

/* Internal Imports */
import { keccak256 } from '../../src/eth/utils'

describe('Ethereum Utils', () => {
  describe('keccak256', () => {
    it('should return the keccak256 hash of a value', () => {
      const value = Buffer.from('1234', 'hex')
      const hash = keccak256(value)

      const expected = Buffer.from('387a8233c96e1fc0ad5e284353276177af2186e7afa85296f106336e376669f7', 'hex')
      hash.should.deep.equal(expected)
    })

    it('should automatically add 0x if it does not exist', () => {
      const valueA = Buffer.from('123')
      const valueB = Buffer.from('0x123')
      const hashA = keccak256(valueA)
      const hashB = keccak256(valueB)

      hashA.should.deep.equal(hashB)
    })
  })
})
