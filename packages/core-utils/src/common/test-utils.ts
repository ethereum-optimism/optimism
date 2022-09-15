import { expect } from 'chai'
import { BigNumber } from '@ethersproject/bignumber'

import { sleep } from './misc'

interface deviationRanges {
  percentUpperDeviation?: number
  percentLowerDeviation?: number
  absoluteUpperDeviation?: number
  absoluteLowerDeviation?: number
}

export const awaitCondition = async (
  cond: () => Promise<boolean>,
  rate = 1000,
  attempts = 10
) => {
  for (let i = 0; i < attempts; i++) {
    const ok = await cond()
    if (ok) {
      return
    }

    await sleep(rate)
  }

  throw new Error('Timed out.')
}

/**
 * Assert that a number lies within a custom defined range of the target.
 */
export const expectApprox = (
  actual: BigNumber | number,
  target: BigNumber | number,
  {
    percentUpperDeviation,
    percentLowerDeviation,
    absoluteUpperDeviation,
    absoluteLowerDeviation,
  }: deviationRanges
): void => {
  actual = BigNumber.from(actual)
  target = BigNumber.from(target)

  // Ensure at least one deviation parameter is defined
  const nonNullDeviations =
    percentUpperDeviation ||
    percentLowerDeviation ||
    absoluteUpperDeviation ||
    absoluteLowerDeviation
  if (!nonNullDeviations) {
    throw new Error(
      'Must define at least one parameter to limit the deviation of the actual value.'
    )
  }

  // Upper bound calculation.
  let upper: BigNumber
  // Set the two possible upper bounds if and only if they are defined.
  const upperPcnt: BigNumber = !percentUpperDeviation
    ? null
    : target.mul(100 + percentUpperDeviation).div(100)
  const upperAbs: BigNumber = !absoluteUpperDeviation
    ? null
    : target.add(absoluteUpperDeviation)

  if (upperPcnt && upperAbs) {
    // If both are set, take the lesser of the two upper bounds.
    upper = upperPcnt.lte(upperAbs) ? upperPcnt : upperAbs
  } else {
    // Else take whichever is not undefined or set to null.
    upper = upperPcnt || upperAbs
  }

  // Lower bound calculation.
  let lower: BigNumber
  // Set the two possible lower bounds if and only if they are defined.
  const lowerPcnt: BigNumber = !percentLowerDeviation
    ? null
    : target.mul(100 - percentLowerDeviation).div(100)
  const lowerAbs: BigNumber = !absoluteLowerDeviation
    ? null
    : target.sub(absoluteLowerDeviation)
  if (lowerPcnt && lowerAbs) {
    // If both are set, take the greater of the two lower bounds.
    lower = lowerPcnt.gte(lowerAbs) ? lowerPcnt : lowerAbs
  } else {
    // Else take whichever is not undefined or set to null.
    lower = lowerPcnt || lowerAbs
  }

  // Apply the assertions if they are non-null.
  if (upper) {
    expect(
      actual.lte(upper),
      `Actual value (${actual}) is greater than the calculated upper bound of (${upper})`
    ).to.be.true
  }
  if (lower) {
    expect(
      actual.gte(lower),
      `Actual value (${actual}) is less than the calculated lower bound of (${lower})`
    ).to.be.true
  }
}
