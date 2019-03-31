/**
 * ExitGuard watches for invalid withdrawals of specific state objects.
 */
export interface ExitGuard {
  /**
   * Makes the ExitGuard start watching for exits
   * on a specific state object.
   * @param stateId ID of the state object.
   */
  subscribe(stateId: any): void

  /**
   * Makes the ExitGuard stop watching for exits
   * on a specific state object
   * @param stateId ID of the state object.
   */
  unsubscribe(stateId: any): void
}
