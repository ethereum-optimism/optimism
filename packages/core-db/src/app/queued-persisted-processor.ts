/* External Imports */
import { getLogger, logError, Logger, sleep } from '@eth-optimism/core-utils'

import * as AsyncLock from 'async-lock'

/* Internal Imports */
import { QueuedPersistedProcessor, RDB, Row } from '../types'
import { last } from 'ethereum-waffle/dist/utils'

const log: Logger = getLogger('base-persisted-queue')
const lockKey: string = 'lock_key'

export interface QueuedPersistedProcessorItem<T> {
  item: T
  processed: boolean
}

export abstract class BaseQueuedPersistedProcessor<T>
  implements QueuedPersistedProcessor<T> {
  private processingLock: AsyncLock

  protected constructor(
    private readonly rdb: RDB,
    private readonly persistenceKey: string,
    startIndex: number = 0,
    private readonly retrySleepDelayMillis: number = 1000
  ) {
    this.processingLock = new AsyncLock()
  }

  /**
   * @inheritDoc
   */
  public async add(index: number, item: T): Promise<void> {
    await this.persistItem(index, item)

    // purposefully not awaiting response
    this.handleIfReady(index, item)
  }

  /**
   * @inheritDoc
   *
   * Note: This is not async-safe in that if it is called at the same time for the same index,
   * the outcome is non-deterministic.
   */
  public async markProcessed(index: number): Promise<void> {
    this.log(
      `Marking index ${index} as processed for processor ${this.persistenceKey}`
    )

    await this.updateToProcessed(index)

    setTimeout(() => {
      this.handleIfExists(index + 1)
    }, 0)
  }

  /**
   * @inheritDoc
   */
  public async getLastIndexProcessed(): Promise<number> {
    const res = await this.rdb.select(
      `SELECT MAX(sequence_number) as last_processed
      FROM sequential_processing
      WHERE 
        sequence_key = ${this.persistenceKey}
        AND processed = TRUE`
    )

    if (
      !res ||
      !res.length ||
      res[0]['last_processed'] === null ||
      res[0]['last_processed'] === undefined
    ) {
      return -1
    }

    return res[0]['last_processed']
  }

  /**
   * Handles the provided item, which is the next item in the queue.
   * When handling is complete, whether sync or async, `markProcessed(...)` must be called.
   *
   * @param index The index of the item.
   * @param item The item to handle.
   */
  protected abstract handleNextItem(index: number, item: T): Promise<void>

  /**
   * Serializes the provided item of T type into a string.
   *
   * @param item The item to serialize.
   * @returns The string of the serialized item.
   */
  protected abstract serializeItem(item: T): Promise<string>

  /**
   * Deserializes the provided string, representing an item of T type.
   *
   * @param itemString The string representation of the object to deserialize.
   * @returns The deserialized item.
   */
  protected abstract deserializeItem(itemString: string): Promise<T>

  /**
   * Initializes the processor based on any previously-stored state.
   */
  protected async init(): Promise<void> {
    const lastProcessed: number = await this.getLastIndexProcessed()
    return this.handleIfExists(lastProcessed + 1)
  }

  protected async updateToProcessed(index: number): Promise<void> {
    return this.rdb.execute(
      `UPDATE sequential_processing
      SET processed = TRUE
      WHERE 
        sequence_key = '${this.persistenceKey}'
        AND sequence_number = ${index}
      `
    )
  }

  /**
   * Fetches the item with the provided index from storage if it exists.
   *
   * @param index The index in question.
   * @returns The fetched item if it exists, undefined otherwise.
   */
  protected async fetchItem(
    index: number
  ): Promise<QueuedPersistedProcessorItem<T>> {
    const res: Row[] = await this.rdb.select(
      `SELECT data_to_process, processed
      FROM sequential_processing
      WHERE
        sequence_key = '${this.persistenceKey}'
        AND sequence_number = ${index}`
    )

    if (!res || !res.length || !res[0]['data_to_process']) {
      return undefined
    }
    return {
      item: await this.deserializeItem(res[0]['data_to_process']),
      processed: !!res[0]['processed'],
    }
  }

  /**
   * Log utility that prepends logs with this specific instance's persistence key.
   *
   * @param msg The message to log.
   * @param error Whether or not this should be logged to error.
   */
  protected log(msg: string, error: boolean = false, e?: Error): void {
    const message: string = `[${this.persistenceKey}] ${msg}`
    if (error) {
      if (!!e) {
        logError(log, message, e)
      } else {
        log.error(message)
      }
    } else {
      this.log(message)
    }
  }

  /**
   * Log utility for logging errors.
   *
   * @param msg The message to log.
   * @param e The associated error, if one exists.
   */
  protected logError(msg: string, e?: Error): void {
    this.log(msg, true, e)
  }

  /**
   * Handles the provided item if it is ready for processing, namely if it is
   * the next to process and the one before it has been marked as complete.
   *
   * Note: this is not async-safe in that if it is called at the same time for the same index & item,
   * the outcome is non-deterministic.
   *
   * @param index The index in question.
   * @param item The item.
   */
  private async handleIfReady(index: number, item: T): Promise<void> {
    const shouldRetry: boolean = await this.processingLock.acquire(
      lockKey,
      async () => {
        try {
          const lastProcessed: number = await this.getLastIndexProcessed()

          if (lastProcessed !== index - 1) {
            this.log(
              `Told to process ${index} but last index processed was ${lastProcessed}, so will process.`
            )
            return false
          }

          this.log(`Handling index ${index}.`)
          await this.handleNextItem(index, item)
        } catch (e) {
          this.log(`Error handling item ${index}. Going to retry.`, true, e)
          return true
        }
      }
    )

    if (!shouldRetry) {
      return
    }

    await sleep(this.retrySleepDelayMillis)
    return this.handleIfReady(index, item)
  }

  /**
   * Handles the item with the provided index if one exists and is not already processed.
   *
   * @param index The index in question.
   */
  private async handleIfExists(index: number): Promise<void> {
    const item: QueuedPersistedProcessorItem<T> = await this.fetchItem(index)

    if (!item) {
      this.log(
        `Index ${index} not yet present. Waiting for its arrival to process it.`
      )
      return
    } else if (!!item.processed) {
      this.log(
        `Index ${index} already processed. Attempting to process ${index + 1}`
      )
      setTimeout(() => {
        this.handleIfExists(index + 1)
      }, 0)
      return
    }

    await this.handleIfReady(index, item.item)
  }

  /**
   * Stores the provided item, associating it with the provided index.
   *
   * @param index The index of the item.
   * @param item The item.
   */
  private async persistItem(index: number, item: T): Promise<void> {
    const serializedItem: string = await this.serializeItem(item)

    try {
      await this.rdb.execute(
        `INSERT INTO sequential_processing(sequence_key, sequence_number, data_to_process)
        VALUES('${this.persistenceKey}', ${index}, '${serializedItem}')
        ON CONFLICT ON CONSTRAINT sequential_processing_sequence_key_sequence_number_key DO NOTHING`
      )
    } catch (e) {
      this.log(
        `Error persisting index ${index} data: ${serializedItem}.`,
        true,
        e
      )
      throw e
    }

    this.log(`Persisted item with index ${index}: ${serializedItem}`)
  }
}
