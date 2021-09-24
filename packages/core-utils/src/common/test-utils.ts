import { expect } from 'chai'
import { BigNumber } from 'ethers'

interface percentDeviationRange {
  upperPercentDeviation: number
  lowerPercentDeviation?: number
}

/**
 * Assert that a number lies within a custom defined range of the target.
 */
export const expectApprox = (
  actual: BigNumber | number,
  target: BigNumber | number,
  { upperPercentDeviation, lowerPercentDeviation = 100 }: percentDeviationRange
): void => {
  actual = BigNumber.from(actual)
  target = BigNumber.from(target)

  const validDeviations =
    upperPercentDeviation >= 0 &&
    upperPercentDeviation <= 100 &&
    lowerPercentDeviation >= 0 &&
    lowerPercentDeviation <= 100
  if (!validDeviations) {
    throw new Error(
      'Upper and lower deviation percentage arguments should be between 0 and 100'
    )
  }
  const upper = target.mul(100 + upperPercentDeviation).div(100)
  const lower = target.mul(100 - lowerPercentDeviation).div(100)

  expect(
    actual.lte(upper),
    `Actual value (${actual}) is more than ${upperPercentDeviation}% greater than target (${target})`
  ).to.be.true
  expect(
    actual.gte(lower),
    `Actual value (${actual}) is more than ${lowerPercentDeviation}% less than target (${target})`
  ).to.be.true
}
