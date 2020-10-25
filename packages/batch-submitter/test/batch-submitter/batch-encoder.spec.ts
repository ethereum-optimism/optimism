import '../setup'

/* Internal Imports */
import { ctcCoder } from '../../src'
import { expect } from 'chai'

describe('BatchEncoder', () => {
  describe('eip155TxData', () => {
    it('should encode & then decode to the correct value', () => {
      const eip155TxData = {
        sig: {
          v: '01',
          r: '11'.repeat(32),
          s: '22'.repeat(32),
        },
        gasLimit: 500,
        gasPrice: 100,
        nonce: 100,
        target: '12'.repeat(20),
        data: '99'.repeat(10),
      }
      const encoded = ctcCoder.eip155TxData.encode(eip155TxData)
      const decoded = ctcCoder.eip155TxData.decode(encoded)
      expect(eip155TxData).to.deep.equal(decoded)
    })
  })

  describe('createEOATxData', () => {
    it('should encode & then decode to the correct value', () => {
      const createEOATxData = {
        sig: {
          v: '01',
          r: '11'.repeat(32),
          s: '22'.repeat(32),
        },
        messageHash: '89'.repeat(32),
      }
      const encoded = ctcCoder.createEOATxData.encode(createEOATxData)
      const decoded = ctcCoder.createEOATxData.decode(encoded)
      expect(createEOATxData).to.deep.equal(decoded)
    })
  })
})
