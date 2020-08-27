/**
 * Defines an interface for all classes that require ordered processing of items that may
 * or may not arrive in order and must be resilient through application failure.
 */
export interface QueuedPersistedProcessor<T> {
  /**
   * Persists and queues the provided item with the associated index,
   * adding it to the processor queue.
   *
   * @param index The index in question.
   * @param item The item to be added.
   */
  add(index: number, item: T): Promise<void>

  /**
   * Marks the item with the provided index as successfully processed, letting
   * the queued processor know that the next item may be processed.
   *
   * Note: The processor will not advance to the next item until this is called.
   *
   * @param index The index being marked as processed.
   */
  markProcessed(index: number): Promise<void>

  /**
   * Gets the last index processed by this processor.
   */
  getLastIndexProcessed(): Promise<number>
}
