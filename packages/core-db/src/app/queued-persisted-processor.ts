/* External Imports */
import {
  bufferUtils,
  bufToHexString,
  getLogger,
  hexStrToBuf,
  logError,
  Logger,
  sleep,
} from '@eth-optimism/core-utils'

/* Internal Imports */
import { DB, QueuedPersistedProcessor } from '../types'

const log: Logger = getLogger('base-persisted-queue')

export abstract class BaseQueuedPersistedProcessor<T>
  implements QueuedPersistedProcessor<T> {
  public static readonly NEXT_INDEX_TO_PROCESS_KEY: string =
    'NEXT_INDEX_TO_PROCESS'
  public static readonly LAST_INDEX_PROCESSED: string = 'LAST_INDEX_PROCESSED'
  public static readonly ITEM_STORAGE_KEY_PREFIX: string = 'ITEM_'

  private initialized: boolean
  private lastIndexProcessed: number
  private nextIndexToProcess: number

  protected constructor(
    private readonly db: DB,
    private readonly persistenceKey: string,
    startIndex: number = 0,
    private readonly retrySleepDelayMillis: number = 1000
  ) {
    this.initialized = false
    this.nextIndexToProcess = startIndex
    this.lastIndexProcessed = startIndex - 1
  }

  /**
   * @inheritDoc
   */
  public async add(index: number, item: T): Promise<void> {
    // Already processed this index, ignore it.
    if (index < this.nextIndexToProcess) {
      return
    }

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
    if (index <= this.lastIndexProcessed) {
      this.log(
        `Persisted queue received instruction to mark index ${index} processed, but last processed index is ${this.lastIndexProcessed}. Ignoring.`
      )
      return
    }
    if (index > this.lastIndexProcessed + 1) {
      this.logError(
        `Persisted queue [${this.persistenceKey}] received instruction to mark index ${index} as processed, but last processed index is ${this.lastIndexProcessed}. Ignoring, but this is an error!`
      )
      return
    }

    try {
      await this.setNextToProcess(index + 1)
    } catch (e) {
      this.log(`Error setting next to process to ${index + 1}!`, e)
      throw e
    }

    this.setLastProcessed(index).then(async () => {
      this.log(
        `Attempting to fetch index ${this.nextIndexToProcess} from storage`
      )
      const nextItem = await this.fetchItem(this.nextIndexToProcess)
      if (!!nextItem) {
        this.log(
          `Index ${this.nextIndexToProcess} was already stored. Handling it now.`
        )
        setTimeout(() => {
          this.handleIfReady(this.nextIndexToProcess, nextItem)
        }, 0)
      } else {
        this.log(
          `Have not received index ${this.nextIndexToProcess} yet. Waiting...`
        )
      }
    })
  }

  /**
   * @inheritDoc
   */
  public async getLastIndexProcessed(): Promise<number> {
    return this.lastIndexProcessed
  }

  /**
   * @inheritDoc
   */
  public async getNextIndexToProcess(): Promise<number> {
    return this.nextIndexToProcess
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
   * Serializes the provided item of T type into a Buffer.
   *
   * @param item The item to serialize.
   * @returns The buffer of the serialized item.
   */
  protected abstract serializeItem(item: T): Promise<Buffer>

  /**
   * Deserializes the provided Buffer, representing an item of T type.
   *
   * @param itemBuffer The buffer to deserialize.
   * @returns The deserialized item.
   */
  protected abstract deserializeItem(itemBuffer: Buffer): Promise<T>

  /**
   * Initializes the processor based on any previously-stored state.
   */
  protected async init(): Promise<void> {
    if (this.initialized) {
      return
    }

    let lastProcessedBuf
    let nextToProcessBuf
    ;[lastProcessedBuf, nextToProcessBuf] = await Promise.all([
      this.db.get(
        this.getStorageKey(BaseQueuedPersistedProcessor.LAST_INDEX_PROCESSED)
      ),
      this.db.get(
        this.getStorageKey(
          BaseQueuedPersistedProcessor.NEXT_INDEX_TO_PROCESS_KEY
        )
      ),
    ])

    if (!!lastProcessedBuf) {
      this.lastIndexProcessed = BaseQueuedPersistedProcessor.deserializeNumber(
        lastProcessedBuf
      )
      this.log(
        `Found last processed in the DB. Setting lastIndexProcessed to ${this.lastIndexProcessed}`
      )
    }
    if (!!nextToProcessBuf) {
      this.nextIndexToProcess = BaseQueuedPersistedProcessor.deserializeNumber(
        nextToProcessBuf
      )
      this.log(
        `Found last processed in the DB. Setting lastIndexProcessed to ${this.lastIndexProcessed}`
      )
    }

    const item: T = await this.fetchItem(this.lastIndexProcessed + 1)
    if (!!item) {
      // purposefully not awaiting response
      this.handleIfReady(this.lastIndexProcessed + 1, item, true)
    }

    this.initialized = true
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
   * @param allowRetries whether or not to allow the possibility that this item was sent for
   *        handling twice. This will likely only be used upon restart.
   */
  private async handleIfReady(
    index: number,
    item: T,
    allowRetries: boolean = false
  ): Promise<void> {
    if (
      index === this.lastIndexProcessed + 1 &&
      (index === this.nextIndexToProcess ||
        (allowRetries && index === this.nextIndexToProcess - 1))
    ) {
      try {
        this.log(`Handling index ${index}.`)
        await this.handleNextItem(index, item)
      } catch (e) {
        logError(log, `Error handling item ${index}. Going to retry.`, e)
        await sleep(this.retrySleepDelayMillis)
        return this.handleIfReady(index, item, true)
      }
    } else {
      this.log(
        `Cannot handle ${index} yet. last processed: ${this.lastIndexProcessed}. Next to process: ${this.nextIndexToProcess}`
      )
    }
  }

  /**
   * Stores the provided item, associating it with the provided index.
   *
   * @param index The index of the item.
   * @param item The item.
   */
  private async persistItem(index: number, item: T): Promise<void> {
    const serializedItem: Buffer = await this.serializeItem(item)
    await this.db.put(this.getStorageKeyForIndex(index), serializedItem)
    this.log(`Persisted item with index ${index}: ${serializedItem}`)
  }

  /**
   * Fetches the item with the provided index from storage if it exists.
   *
   * @param index The index in question.
   * @returns The fetched item if it exists, undefined otherwise.
   */
  private async fetchItem(index: number): Promise<T | undefined> {
    const itemBuffer: Buffer = await this.db.get(
      this.getStorageKeyForIndex(index)
    )
    if (!itemBuffer) {
      return undefined
    }
    return this.deserializeItem(itemBuffer)
  }

  /**
   * Gets the storage key for the item with the provided index.
   *
   * @param index The index in question.
   * @returns The storage key (Buffer) of the item in question.
   */
  private getStorageKeyForIndex(index: number): Buffer {
    return this.getStorageKey(
      `${BaseQueuedPersistedProcessor.ITEM_STORAGE_KEY_PREFIX}${index}`
    )
  }

  /**
   * Gets this instance's storage key for the item with the provided key.
   * This takes into account the fact that there may be multiple instances of this class
   * running at once.
   *
   * @param key The key in question.
   * @returns The storage key (Buffer) of the item in question.
   */
  private getStorageKey(key: string): Buffer {
    return Buffer.from(`${this.persistenceKey}_${key}`)
  }

  /**
   * Sets the last processed index, persisting the updated index in case of failure.
   * @param index The index to set Last Processed to.
   */
  private async setLastProcessed(index: number): Promise<void> {
    await this.db.put(
      this.getStorageKey(BaseQueuedPersistedProcessor.LAST_INDEX_PROCESSED),
      BaseQueuedPersistedProcessor.serializeNumber(index)
    )
    this.lastIndexProcessed = index
    this.log(`Last processed incremented to ${this.lastIndexProcessed}`)
  }

  /**
   * Sets the next index to process, persisting the updated index in case of failure.
   * @param index The index to set  Processed to.
   */
  protected async setNextToProcess(index: number): Promise<void> {
    await this.db.put(
      this.getStorageKey(
        BaseQueuedPersistedProcessor.NEXT_INDEX_TO_PROCESS_KEY
      ),
      BaseQueuedPersistedProcessor.serializeNumber(index)
    )
    this.nextIndexToProcess = index
    this.log(`Next to process incremented to ${this.nextIndexToProcess}`)
  }

  /**
   * Utility to make number serialization consistent.
   * This must be compatible with deserializeNumber(...).
   *
   * @param num The number to be serialized.
   * @returns The serialized number as a Buffer.
   */
  private static serializeNumber(num: number): Buffer {
    const hex: string = num.toString(16)
    return hexStrToBuf(hex.length % 2 === 0 ? hex : `0${hex}`)
  }

  /**
   * Utility to make number deserialization consistent.
   * This must be compatible with serializeNumber(...).
   *
   * @param buf The buffer to deserialize into a number
   * @returns The deserialized number.
   */
  private static deserializeNumber(buf: Buffer): number {
    return parseInt(bufToHexString(buf, false), 16)
  }
}
