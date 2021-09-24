import { expect } from '../setup'

/* Imports: Internal */
import { expectApprox } from '../../src'

describe('expectApprox', async () => {
  it('should throw an error if the actual value is higher than expected', async () => {
    try {
      expectApprox(121, 100, {
        upperPercentDeviation: 20,
      })
    } catch (error) {
      expect(error.message).to.equal(
        'Actual value (121) is more than 20% greater than target (100): expected false to be true'
      )
    }
  })

  it('should throw an error if the actual value is lower than expected', async () => {
    try {
      expectApprox(79, 100, {
        upperPercentDeviation: 0,
        lowerPercentDeviation: 20,
      })
    } catch (error) {
      expect(error.message).to.equal(
        'Actual value (79) is more than 20% less than target (100): expected false to be true'
      )
    }
  })
})
