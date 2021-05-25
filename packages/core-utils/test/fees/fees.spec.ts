import { expect } from '../setup'
import * as fees from '../../src/fees'
import { BigNumber } from 'ethers'

describe('Fees', () => {
  it('should count zeros and ones', () => {
    const cases = [
      { input: Buffer.from('0001', 'hex'), zeros: 1, ones: 1 },
      { input: '0x0001', zeros: 1, ones: 1 },
      { input: '0x', zeros: 0, ones: 0 },
      { input: '0x1111', zeros: 0, ones: 2 },
    ]

    for (const test of cases) {
      const [zeros, ones] = fees.zeroesAndOnes(test.input)
      zeros.should.eq(test.zeros)
      ones.should.eq(test.ones)
    }
  })

  describe('Round L1 Gas Price', () => {
    const roundL1GasPriceTests = [
      { input: 10, expect: 10 ** 8, name: 'simple' },
      { input: 10 ** 8 + 1, expect: 2 * 10 ** 8, name: 'one-over' },
      { input: 10 ** 8, expect: 10 ** 8, name: 'exact' },
      { input: 10 ** 8 - 1, expect: 10 ** 8, name: 'one-under' },
      { input: 3, expect: 10 ** 8, name: 'small' },
      { input: 2, expect: 10 ** 8, name: 'two' },
      { input: 1, expect: 10 ** 8, name: 'one' },
      { input: 0, expect: 0, name: 'zero' },
    ]

    for (const test of roundL1GasPriceTests) {
      it(`should pass for ${test.name} case`, () => {
        const got = fees.roundL1GasPrice(test.input)
        const expected = BigNumber.from(test.expect)
        expect(got).to.deep.equal(expected)
      })
    }
  })

  describe('Round L2 Gas Price', () => {
    const roundL2GasPriceTests = [
      { input: 10, expect: 10 ** 8 + 1, name: 'simple' },
      { input: 10 ** 8 + 2, expect: 2 * 10 ** 8 + 1, name: 'one-over' },
      { input: 10 ** 8 + 1, expect: 10 ** 8 + 1, name: 'exact' },
      { input: 10 ** 8, expect: 10 ** 8 + 1, name: 'one-under' },
      { input: 3, expect: 10 ** 8 + 1, name: 'small' },
      { input: 2, expect: 10 ** 8 + 1, name: 'two' },
      { input: 1, expect: 10 ** 8 + 1, name: 'one' },
      { input: 0, expect: 1, name: 'zero' },
    ]

    for (const test of roundL2GasPriceTests) {
      it(`should pass for ${test.name} case`, () => {
        const got = fees.roundL2GasPrice(test.input)
        const expected = BigNumber.from(test.expect)
        expect(got).to.deep.equal(expected)
      })
    }
  })

  describe('Rollup Fees', () => {
    const rollupFeesTests = [
      {name: 'simple', dataLen: 10, l1GasPrice: 100_000_000, l2GasPrice: 100_000_001, l2GasLimit: 437118, error: false},
      {name: 'zero-l2-gasprice', dataLen: 10, l1GasPrice: 100_000_000, l2GasPrice: 0, l2GasLimit: 196205, error: true},
      {name: 'one-l2-gasprice', dataLen: 10, l1GasPrice: 100_000_000, l2GasPrice: 1, l2GasLimit: 196205, error: false},
      {name: 'zero-l1-gasprice', dataLen: 10, l1GasPrice: 0, l2GasPrice: 100_000_001, l2GasLimit: 196205, error: false},
      {name: 'one-l1-gasprice', dataLen: 10, l1GasPrice: 1, l2GasPrice: 23254, l2GasLimit: 23255, error: true},
    ]

    for (const test of rollupFeesTests) {
      it(`should pass for ${test.name} case`, () => {
        const data = Buffer.alloc(test.dataLen)

        let got
        let err = false
        try {
          got = fees.L2GasLimit.encode({
            data,
            l1GasPrice: test.l1GasPrice,
            l2GasPrice: test.l2GasPrice,
            l2GasLimit: test.l2GasLimit,
          })
        } catch (e) {
          err = true
        }

        expect(err).to.equal(test.error)

        if (!err) {
          const decoded = fees.L2GasLimit.decode(got)
          expect(decoded).to.deep.eq(BigNumber.from(test.l2GasLimit))
        }
      })
    }
  })
})
