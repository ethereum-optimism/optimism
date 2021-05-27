import { expect } from '../setup'
import * as fees from '../../src/fees'
import { BigNumber } from 'ethers'

const hundredBillion = 10 ** 11
const million = 10 ** 6

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

  describe('Round Gas Price', () => {
    const roundGasPriceTests = [
      { input: 10, expect: hundredBillion, name: 'simple' },
      {
        input: hundredBillion + 1,
        expect: 2 * hundredBillion,
        name: 'one-over',
      },
      { input: hundredBillion, expect: hundredBillion, name: 'exact' },
      { input: hundredBillion - 1, expect: hundredBillion, name: 'one-under' },
      { input: 3, expect: hundredBillion, name: 'small' },
      { input: 2, expect: hundredBillion, name: 'two' },
      { input: 1, expect: hundredBillion, name: 'one' },
      { input: 0, expect: 0, name: 'zero' },
    ]

    for (const test of roundGasPriceTests) {
      it(`should pass for ${test.name} case`, () => {
        const got = fees.roundGasPrice(test.input)
        const expected = BigNumber.from(test.expect)
        expect(got).to.deep.equal(expected)
      })
    }
  })

  describe('Rollup Fees', () => {
    const rollupFeesTests = [
      {
        name: 'simple',
        dataLen: 10,
        l1GasPrice: hundredBillion,
        l2GasPrice: hundredBillion,
        l2GasLimit: 437118,
        error: false,
      },
      {
        name: 'zero-l2-gasprice',
        dataLen: 10,
        l1GasPrice: hundredBillion,
        l2GasPrice: 0,
        l2GasLimit: 196205,
        error: false,
      },
      {
        name: 'one-l2-gasprice',
        dataLen: 10,
        l1GasPrice: hundredBillion,
        l2GasPrice: 1,
        l2GasLimit: 196205,
        error: true,
      },
      {
        name: 'zero-l1-gasprice',
        dataLen: 10,
        l1GasPrice: 0,
        l2GasPrice: hundredBillion,
        l2GasLimit: 196205,
        error: false,
      },
      {
        name: 'one-l1-gasprice',
        dataLen: 10,
        l1GasPrice: 1,
        l2GasPrice: hundredBillion,
        l2GasLimit: 23255,
        error: true,
      },
      {
        name: 'zero-gasprices',
        dataLen: 10,
        l1GasPrice: 0,
        l2GasPrice: 0,
        l2GasLimit: 23255,
        error: false,
      },
      {
        name: 'bad-l2-gasprice',
        dataLen: 10,
        l1GasPrice: 0,
        l2GasPrice: hundredBillion - 1,
        l2GasLimit: 23255,
        error: true,
      },
      {
        name: 'bad-l1-gasprice',
        dataLen: 10,
        l1GasPrice: hundredBillion - 1,
        l2GasPrice: hundredBillion,
        l2GasLimit: 44654,
        error: true,
      },
      // The largest possible gaslimit that can be represented
      // is 0x04ffffff which is plenty high enough to cover the
      // L2 gas limit
      {
        name: 'max-gaslimit',
        dataLen: 10,
        l1GasPrice: hundredBillion,
        l2GasPrice: hundredBillion,
        l2GasLimit: 0x4ffffff,
        error: false,
      },
      {
        name: 'larger-divisor',
        dataLen: 10,
        l1GasPrice: 0,
        l2GasLimit: 10,
        l2GasPrice: 0,
        error: false,
      },
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
