import '../setup'

/* Internal Imports */
import { ctcCoder } from '../../src'

describe('BatchSubmitter', () => {
  describe('Submit', () => {
    it('should print', () => {
        console.log('Hello there!')
        const testEncoded = ctcCoder.eip155TxData.encode({
            sig: {
              v: '1',
              r: '11'.repeat(32),
              s: '11'.repeat(32)
            },
            gasLimit: 500,
            gasPrice: 100,
            nonce: 100,
            target: '0x' + '12'.repeat(20),
            data: '0x' + '99'.repeat(10)
        })
        console.log(testEncoded)
        console.log(ctcCoder.eip155TxData.decode(testEncoded))
    })
  })
})

