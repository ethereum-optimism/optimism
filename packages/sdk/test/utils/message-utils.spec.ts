import { BigNumber } from 'ethers'

import { expect } from '../setup'
import { migratedWithdrawalGasLimit } from '../../src/utils/message-utils'

describe('Message Utils', () => {
  describe('migratedWithdrawalGasLimit', () => {
    it('should have a max of 25 million', () => {
      const data = '0x' + 'ff'.repeat(15_000_000)
      const result = migratedWithdrawalGasLimit(data)
      expect(result).to.eq(BigNumber.from(25_000_000))
    })

    it('should work for mixes of zeros and ones', () => {
      const tests = [
        { input: '0x', result: BigNumber.from(200_000) },
        { input: '0xff', result: BigNumber.from(200_000 + 16) },
        { input: '0xff00', result: BigNumber.from(200_000 + 16 + 4) },
        { input: '0x00', result: BigNumber.from(200_000 + 4) },
        { input: '0x000000', result: BigNumber.from(200_000 + 4 + 4 + 4) },
      ]

      for (const test of tests) {
        const result = migratedWithdrawalGasLimit(test.input)
        expect(result).to.eq(test.result)
      }
    })
  })
})
