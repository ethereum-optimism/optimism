/* External Imports */
import { getLogger, Logger, logError } from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  SequentialProcessingDataService,
  SequentialProcessingItem,
} from '../../types/queue'
import { RDB, Row } from '../../types/db'

const log: Logger = getLogger('sequential-processing-data-service')

export class DefaultSequentialProcessingDataService
  implements SequentialProcessingDataService {
  constructor(private readonly rdb: RDB) {}

  /**
   * @inheritDoc
   */
  public async fetchItem(
    index: number,
    sequenceKey: string
  ): Promise<SequentialProcessingItem> {
    const res: Row[] = await this.rdb.select(
      `SELECT data_to_process, processed
      FROM sequential_processing
      WHERE
        sequence_key = '${sequenceKey}'
        AND sequence_number = ${index}`
    )

    if (!res || !res.length || !res[0]['data_to_process']) {
      return undefined
    }
    return {
      data: res[0]['data_to_process'],
      processed: !!res[0]['processed'],
    }
  }

  /**
   * @inheritDoc
   */
  public async getLastIndexProcessed(sequenceKey: string): Promise<number> {
    const res = await this.rdb.select(
      `SELECT MAX(sequence_number) as last_processed
      FROM sequential_processing
      WHERE 
        sequence_key = '${sequenceKey}'
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

    return parseInt(res[0]['last_processed'], 10)
  }

  /**
   * @inheritDoc
   */
  public async persistItem(
    index: number,
    itemData: string,
    sequenceKey: string,
    processed: boolean = false
  ): Promise<void> {
    try {
      await this.rdb.execute(
        `INSERT INTO sequential_processing(sequence_key, sequence_number, data_to_process, processed)
        VALUES('${sequenceKey}', ${index}, '${itemData}', ${
          processed ? 'TRUE' : 'FALSE'
        })
        ON CONFLICT ON CONSTRAINT sequential_processing_sequence_key_sequence_number_key DO NOTHING`
      )
    } catch (e) {
      logError(
        log,
        `[${sequenceKey}] Error persisting index ${index} data: ${itemData}.`,
        e
      )
      throw e
    }

    log.debug(
      `[${sequenceKey}] Persisted item with index ${index}: ${itemData}`
    )
  }

  /**
   * @inheritDoc
   */
  public async updateToProcessed(
    index: number,
    sequenceKey: string
  ): Promise<void> {
    await this.rdb.execute(
      `UPDATE sequential_processing
      SET processed = TRUE
      WHERE 
        sequence_key = '${sequenceKey}'
        AND sequence_number = ${index}`
    )
    log.debug(`[${sequenceKey}] index ${index} updated to processed.`)
  }
}

/**
 * Mock data service used to mock out the one specified above.
 */
export class InMemoryProcessingDataService
  implements SequentialProcessingDataService {
  public lastProcessedIndex: Map<string, number>
  public items: Map<string, Map<number, SequentialProcessingItem>>

  constructor() {
    this.items = new Map<string, Map<number, SequentialProcessingItem>>()
    this.lastProcessedIndex = new Map<string, number>()
  }

  public async updateToProcessed(
    index: number,
    sequenceKey: string
  ): Promise<void> {
    const map = this.items.get(sequenceKey)
    if (!map) {
      throw Error(
        `Tried to update processed for a sequence key that contains no items (${sequenceKey})`
      )
    }
    const item = map.get(index)
    if (!item) {
      throw Error(`Tried to updated index ${index} which does not exist.`)
    }
    item.processed = true
    this.lastProcessedIndex.set(sequenceKey, index)
  }

  public async persistItem(
    index: number,
    data: string,
    sequenceKey: string,
    processed: boolean = false
  ): Promise<void> {
    if (!this.items.get(sequenceKey)) {
      this.items.set(sequenceKey, new Map<number, SequentialProcessingItem>())
    }
    if (!this.items.get(sequenceKey).get(index)) {
      this.items.get(sequenceKey).set(index, { processed, data })
    }
  }

  public async fetchItem(
    index: number,
    sequenceKey: string
  ): Promise<SequentialProcessingItem> {
    if (!this.items.get(sequenceKey)) {
      return undefined
    }
    return this.items.get(sequenceKey).get(index)
  }

  public async getLastIndexProcessed(sequenceKey: string): Promise<number> {
    const index = this.lastProcessedIndex.get(sequenceKey)
    return index === undefined ? -1 : index
  }
}
