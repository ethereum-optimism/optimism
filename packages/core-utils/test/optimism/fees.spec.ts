import '../setup'

import { BigNumber } from '@ethersproject/bignumber'

import { zeroesAndOnes, calldataCost } from '../../src'

describe('Fees', () => {
  it('should count zeros and ones', () => {
    const cases = [
      { input: Buffer.from('0001', 'hex'), zeros: 1, ones: 1 },
      { input: '0x0001', zeros: 1, ones: 1 },
      { input: '0x', zeros: 0, ones: 0 },
      { input: '0x1111', zeros: 0, ones: 2 },
    ]

    for (const test of cases) {
      const [zeros, ones] = zeroesAndOnes(test.input)
      zeros.should.eq(test.zeros)
      ones.should.eq(test.ones)
    }
  })

  it('should compute calldata costs', () => {
    const cases = [
      { input: '0x', output: BigNumber.from(0) },
      { input: '0x00', output: BigNumber.from(4) },
      { input: '0xff', output: BigNumber.from(16) },
      { input: Buffer.alloc(32), output: BigNumber.from(4 * 32) },
      { input: Buffer.alloc(32, 0xff), output: BigNumber.from(16 * 32) },
    ]

    for (const test of cases) {
      const cost = calldataCost(test.input)
      cost.should.deep.eq(test.output)
    }
  })
})
