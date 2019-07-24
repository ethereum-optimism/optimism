import { QuantifierResult, Quantifier } from '../../types'

// Helper function which returns an array of numbers, starting at start, ending at end, incrementing by 1.
// Eg. [0, 1, 2,...end]
const range = (start: number, end: number): number[] => {
  return Array(end - start)
    .fill(start)
    .map((x, y) => x + y)
}

/*
 * The parameter type for `getAllQuantified(...)` in the IntegerRangeQuantifier.
 */
interface IntegerRangeParameters {
  start: number
  end: number
}

/*
 * The IntegerRangeQuantifier returns a range of integers between the start (inclusive) & end (exclusive).
 */
export class IntegerRangeQuantifier implements Quantifier {
  /**
   * Returns a QuantifierResult where results are an array of integers from 0 to withinThisRange. Eg. 3 to 6 = [3, 4, 5]
   * and `allResultsQuantified` is set to `true`--this is because we always can quantify integers in this range.
   *
   * @param withinThisRange the range of the integers we would like to return.
   */
  public getAllQuantified(withinThisRange: {
    start: number
    end: number
  }): QuantifierResult {
    if (withinThisRange.end < withinThisRange.start) {
      throw new Error('Invalid quantifier input! End is less than the start.')
    }
    return {
      results: range(withinThisRange.start, withinThisRange.end),
      allResultsQuantified: true,
    }
  }
}

/*
 * The parameter type for `getAllQuantified(...)` in the NonnegativeIntegerLessThanQuantifier
 */
type NonnegativeIntegerLessThanQuantifierParameters = number

/*
 * The NonnegativeIntegerLessThanQuantifier returns all non-negative integers less than the specified number
 */
export class NonnegativeIntegerLessThanQuantifier implements Quantifier {
  /**
   * Returns a QuantifierResult where results are an array of integers from 0 to lessThanThis. Eg. 0 to 3 = [0, 1, 2]
   * and `allResultsQuantified` is set to `true`--this is because we always can quantify integers in this range.
   *
   * @param lessThanThis the upper bound for the array. Note this is non-inclusive.
   */
  public getAllQuantified(
    lessThanThis: NonnegativeIntegerLessThanQuantifierParameters
  ): QuantifierResult {
    if (lessThanThis < 0) {
      throw new Error(
        'Invalid quantifier input! Cannot quantify negative number.'
      )
    }
    return {
      results: range(0, lessThanThis),
      allResultsQuantified: true,
    }
  }
}
