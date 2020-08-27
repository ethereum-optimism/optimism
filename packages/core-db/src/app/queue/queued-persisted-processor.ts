/* External Imports */
import { getLogger, logError, Logger, sleep } from '@eth-optimism/core-utils'

import * as AsyncLock from 'async-lock'

/* Internal Imports */
import {
  QueuedPersistedProcessor,
  SequentialProcessingItem,
  SequentialProcessingDataService,
} from '../../types'

const log: Logger = getLogger('base-persisted-queue')
const lockKey: string = 'lock_key'

export abstract class BaseQueuedPersistedProcessor<T>
  implements QueuedPersistedProcessor<T> {
  private readonly processingLock: AsyncLock

  protected constructor(
    private readonly processingDataService: SequentialProcessingDataService,
    private readonly persistenceKey: string,
    private readonly startIndex: number = 0,
    private readonly retrySleepDelayMillis: number = 1000
  ) {
    this.processingLock = new AsyncLock()
  }

  /**
   * @inheritDoc
   */
  public async add(index: number, item: T): Promise<void> {
    await this.processingDataService.persistItem(
      index,
      await this.serializeItem(item),
      this.persistenceKey
    )

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

    await this.processingDataService.updateToProcessed(
      index,
      this.persistenceKey
    )

    setTimeout(() => {
      this.handleIfExists(index + 1)
    }, 0)
  }

  /**
   * @inheritDoc
   */
  public async getLastIndexProcessed(): Promise<number> {
    const last: number = await this.processingDataService.getLastIndexProcessed(
      this.persistenceKey
    )
    return Math.max(last, this.startIndex - 1)
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
      log.debug(message)
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
    const item: SequentialProcessingItem = await this.processingDataService.fetchItem(
      index,
      this.persistenceKey
    )

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

    return this.handleIfReady(index, await this.deserializeItem(item.data))
  }
}
