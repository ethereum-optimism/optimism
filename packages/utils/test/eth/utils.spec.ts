import '../setup'

/* Internal Imports */
import { keccak256 } from '../../src/eth/utils'

describe('Ethereum Utils', () => {
  describe('keccak256', () => {
    it('should return the keccak256 hash of a value', () => {
      const value = '0x123'
      const hash = keccak256(value)

      hash.should.equal(
        '0x667d3611273365cfb6e64399d5af0bf332ec3e5d6986f76bc7d10839b680eb58'
      )
    })

    it('should automatically add 0x if it does not exist', () => {
      const valueA = '123'
      const valueB = '0x123'
      const hashA = keccak256(valueA)
      const hashB = keccak256(valueB)

      hashA.should.equal(hashB)
    })
  })
})
