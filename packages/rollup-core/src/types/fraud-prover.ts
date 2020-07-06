export interface FraudProver {
  /**
   * Handles fraud proving for the item at `batchNumber` and `batchIndex`
   *
   * @param batchNumber The batch number of the fraudulent entry.
   * @param batchIndex The batch index of the fraudulent entry.
   */
  proveFraud(batchNumber: number, batchIndex: number): Promise<void>
}
