import { expect } from '../setup'

/* Imports: Internal */
import { getRandomAddress } from '../../src'

describe('getRandomAddress', () => {
  const random = global.Math.random

  before(async () => {
    global.Math.random = () => 0.5
  })

  after(async () => {
    global.Math.random = random
  })

  it('returns a random address string', () => {
    expect(getRandomAddress()).to.equal('0x' + '88'.repeat(20))
  })
})
