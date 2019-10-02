/**
 * Generic listener for Ethereum events and objects.
 */
export interface EthereumListener<T> {
  /**
   * If past T objects are being synced up to the current block,
   * this callback will be invoked when the sync is completed.
   *
   * @param syncIdentifier The ID of the object type that completed syncing
   */
  onSyncCompleted(syncIdentifier?: string): Promise<void>

  /**
   * This callback will be invoked when a new T object is received from
   * a new block being mined
   * @param t The new object received from a new block being mined
   */
  handle(t: T): Promise<void>
}
