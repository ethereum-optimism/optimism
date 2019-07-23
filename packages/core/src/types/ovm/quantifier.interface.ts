export interface QuantifierResult {
  results: any[]
  allResultsQuantified: boolean
}

/**
 * Interface defining the contract for all Quantifiers. Quantifiers return a collection of
 * results that pass their logic.
 *
 * For example: A LessThanQuantifier would return all numbers less than the provided input.
 * PositiveIntegerLessThanQuantifier.getAllQuantified(5) => [1,2,3,4]
 */
export interface Quantifier {
  /**
   * Gets all of the results that meet the criteria of the quantifier logic and provided parameters.
   *
   * @param parameters the input indicating how the results will be quantified
   * @returns the QuantifierResult with results and our level of knowledge of the results.
   */
  getAllQuantified(parameters: any): QuantifierResult
}
