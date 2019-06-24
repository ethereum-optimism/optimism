import '../../setup'

/* Internal Imports */
import { keccak256 } from '../../../src/app'

describe('Ethereum Utils', () => {
  describe('keccak256', () => {
    it('should return the keccak256 hash of a value', () => {
      const value = Buffer.from('1234', 'hex')
      const hash = keccak256(value)

      const expected = Buffer.from(
        '56570de287d73cd1cb6092bb8fdee6173974955fdef345ae579ee9f475ea7432',
        'hex'
      )
      hash.should.deep.equal(expected)
    })

    it('should return the keccak256 of the empty string', () => {
      const value = Buffer.from('', 'hex')
      const hash = keccak256(value)

      const expected = Buffer.from(
        'c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470',
        'hex'
      )
      hash.should.deep.equal(expected)
    })
  })
})
