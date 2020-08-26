export interface SequentialProcessingItem {
  data: string
  processed: boolean
}

/**
 * Defines the data service used by the Queued Persisted Processor for data storage.
 */
export interface SequentialProcessingDataService {
  /**
   * Updates the record for the provided index to indicate that it has been processed.
   *
   * @param sequenceKey The key identifying the sequence for which the provided index should be updated to processed.
   * @param index The index in question
   */
  updateToProcessed(index: number, sequenceKey: string): Promise<void>

  /**
   * Persists the item in question, associating it with the given index.
   *
   * @param index The index in question.
   * @param sequenceKey The key identifying the sequence in which the provided item should be stored at the provided index.
   * @param item The item to store.
   * @param processed (optional) Whether or not the item should be stored as processed.
   */
  persistItem(
    index: number,
    item: string,
    sequenceKey: string,
    processed?: boolean
  ): Promise<void>

  /**
   * Fetches the item with the provided index, if one exists.
   *
   * @param index The index in question.
   * @param sequenceKey The key identifying the sequence in which this item belongs.
   * @returns The QueuedPersistedProcessorItem.
   */
  fetchItem(
    index: number,
    sequenceKey: string
  ): Promise<SequentialProcessingItem>

  /**
   * Gets the highest index that has been processed.
   *
   * @param sequenceKey The key identifying the sequence for which the last processed index is being requested.
   * @returns The index.
   */
  getLastIndexProcessed(sequenceKey: string): Promise<number>
}
