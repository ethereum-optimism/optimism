import { expect } from '../setup'
import { BigNumber } from 'ethers'

/* Imports: Internal */
import { getRandomHexString, toRpcHexString } from '../../src'

describe('getRandomHexString', () => {
  const random = global.Math.random

  before(async () => {
    global.Math.random = () => 0.5
  })

  after(async () => {
    global.Math.random = random
  })

  it('returns a random address string of the specified length', () => {
    expect(getRandomHexString(8)).to.equal('0x' + '88'.repeat(8))
  })
})

describe('toRpcHexString', () => {
  it('should parse 0', () => {
    expect(toRpcHexString(0)).to.deep.equal('0x0')
    expect(toRpcHexString(BigNumber.from(0))).to.deep.equal('0x0')
  })

  it('should parse non 0', () => {
    const cases = [
      { input: 2, output: '0x2' },
      { input: BigNumber.from(2), output: '0x2' },
      { input: 100, output: '0x64' },
      { input: BigNumber.from(100), output: '0x64' },
      { input: 300, output: '0x12c' },
      { input: BigNumber.from(300), output: '0x12c' },
    ]
    for (const test of cases) {
      expect(toRpcHexString(test.input)).to.deep.equal(test.output)
    }
  })
})
