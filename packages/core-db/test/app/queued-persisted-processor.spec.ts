import { should } from '../setup'

/* External Imports */
import { sleep } from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  BaseQueuedPersistedProcessor,
  newInMemoryDB,
  QueuedPersistedProcessorItem,
} from '../../src/app'
import { DB } from '../../src/types/db'

class DummyQueuedPersistedProcessor extends BaseQueuedPersistedProcessor<
  string
> {
  public lastProcessedIndex: number = -1
  public callMarkProcessed: boolean = true
  public throwOnceHandlingNextItem: boolean = false
  public handledQueue: string[]
  public items: Map<number, QueuedPersistedProcessorItem<string>>

  public static async create(
    startIndex: number = 0,
    retrySleepDelayMillis: number = 1000
  ): Promise<DummyQueuedPersistedProcessor> {
    const processor = new DummyQueuedPersistedProcessor(
      startIndex,
      retrySleepDelayMillis
    )
    await processor.init()
    return processor
  }

  private constructor(
    startIndex: number = 0,
    retrySleepDelayMillis: number = 1000
  ) {
    super(undefined, 'test', startIndex, retrySleepDelayMillis)
    this.handledQueue = []
    this.items = new Map<number, QueuedPersistedProcessorItem<string>>()
  }

  protected async handleNextItem(index: number, item: string): Promise<void> {
    if (this.throwOnceHandlingNextItem) {
      this.throwOnceHandlingNextItem = false
      throw Error('you told me to throw in handleNextItem.')
    }
    this.handledQueue.push(item)
    if (this.callMarkProcessed) {
      return this.markProcessed(index)
    }
  }

  protected async serializeItem(item: string): Promise<string> {
    return item
  }

  protected async deserializeItem(item: string): Promise<string> {
    return item
  }

  protected async updateToProcessed(index: number): Promise<void> {
    this.items.get(index).processed = true
  }

  protected async fetchItem(
    index: number
  ): Promise<QueuedPersistedProcessorItem<string>> {
    return this.items.get(index)
  }

  public async getLastIndexProcessed(): Promise<number> {
    return this.lastProcessedIndex
  }
}

describe('Queued Persisted Processor', () => {
  let db: DB
  let processor: DummyQueuedPersistedProcessor
  const retrySleepDelayMillis: number = 100

  beforeEach(async () => {
    db = newInMemoryDB()
    processor = await DummyQueuedPersistedProcessor.create(
      0,
      retrySleepDelayMillis
    )
  })

  describe('Fresh start', () => {
    it('handles items in order', async () => {
      const item = 'Number 0!'
      await processor.add(0, item)
      await sleep(10)
      processor.handledQueue.length.should.equal(1, `Queue item not processed!`)
      processor.handledQueue[0].should.equal(item, `Incorrect item processed!`)
    })

    it('does not handle items out of order', async () => {
      const item = 'Number 1!'
      await processor.add(1, item)
      await sleep(10)
      processor.handledQueue.length.should.equal(0, `Queue item processed!`)
    })

    it('does not handle next item if previous is not acknowledged', async () => {
      processor.callMarkProcessed = false

      const first = 'Number 0!'
      await processor.add(0, first)
      await processor.add(1, 'Number 1!')
      await sleep(10)
      processor.handledQueue.length.should.equal(
        1,
        `Incorrect number processed!`
      )
      processor.handledQueue[0].should.equal(first, `Incorrect item processed!`)
    })

    it('handles next item if previous is acknowledged', async () => {
      const first = 'Number 0!'
      const second = 'Number 1!'
      await processor.add(0, first)
      await processor.add(1, second)
      await sleep(20)
      processor.handledQueue.length.should.equal(
        2,
        `Incorrect number processed!`
      )
      processor.handledQueue[0].should.equal(first, `Incorrect item processed!`)
      processor.handledQueue[1].should.equal(
        second,
        `Incorrect item processed!`
      )
    })

    it('handles next item added after previous is acknowledged', async () => {
      const first = 'Number 0!'
      const second = 'Number 1!'
      await processor.add(0, first)
      await sleep(20)

      processor.handledQueue.length.should.equal(
        1,
        `Incorrect number processed!`
      )

      await processor.add(1, second)
      await sleep(20)
      processor.handledQueue.length.should.equal(
        2,
        `Incorrect number processed!`
      )
      processor.handledQueue[1].should.equal(
        second,
        `Incorrect item processed!`
      )
    })

    it('retries processing item if handleNextItemThrows', async () => {
      const first = 'Number 0!'
      const second = 'Number 1!'
      await processor.add(0, first)
      await sleep(10)
      processor.handledQueue.length.should.equal(
        1,
        `Incorrect number processed!`
      )
      processor.handledQueue[0].should.equal(first, `Incorrect item processed!`)

      processor.throwOnceHandlingNextItem = true
      await processor.add(1, second)

      await sleep(20)

      processor.handledQueue.length.should.equal(
        1,
        `There should still only be one item processed! Should fail and retry after ${retrySleepDelayMillis} millis`
      )

      await sleep(retrySleepDelayMillis * 2)

      processor.handledQueue.length.should.equal(
        2,
        `Incorrect number processed!`
      )
      processor.handledQueue[1].should.equal(
        second,
        `Incorrect item processed!`
      )
      processor.throwOnceHandlingNextItem.should.equal(
        false,
        'Throw once config should be reset!'
      )
    })
  })

  describe('Start with existing state', () => {
    it('restarts with existing state (empty)', async () => {
      const secondProc = await DummyQueuedPersistedProcessor.create()
      await sleep(10)
      secondProc.handledQueue.length.should.equal(
        0,
        `No items should be processed!`
      )

      const item = '0000'
      await secondProc.add(0, item)
      await sleep(10)
      secondProc.handledQueue.length.should.equal(
        1,
        `Item should have been processed`
      )
      secondProc.handledQueue[0].should.equal(item, `Incorrect item processed`)
    })

    it('restarts with existing state (1 added but not acknowledged)', async () => {
      processor.callMarkProcessed = false

      const item: string = 'Number 0!'
      await processor.add(0, item)
      await sleep(10)
      processor.handledQueue.length.should.equal(
        1,
        `One item should be processed!`
      )
      processor.handledQueue[0].should.equal(item, `Incorrect item processed`)

      const secondProc = await DummyQueuedPersistedProcessor.create()
      await sleep(10)
      secondProc.handledQueue.length.should.equal(
        1,
        `One item should be processed!`
      )
      secondProc.handledQueue[0].should.equal(item, `Incorrect item processed`)
    })

    it('restarts with existing state (1 added and acknowledged)', async () => {
      const item: string = 'Number 0!'
      await processor.add(0, item)
      await sleep(10)
      processor.handledQueue.length.should.equal(
        1,
        `One item should be processed!`
      )
      processor.handledQueue[0].should.equal(item, `Incorrect item processed`)

      const secondProc = await DummyQueuedPersistedProcessor.create()
      await sleep(10)
      secondProc.handledQueue.length.should.equal(
        0,
        `No items should be processed!`
      )
    })

    it('restarts with existing state and processed new item', async () => {
      const item: string = 'Number 0!'
      await processor.add(0, item)
      await sleep(10)
      processor.handledQueue.length.should.equal(
        1,
        `One item should be processed!`
      )
      processor.handledQueue[0].should.equal(item, `Incorrect item processed`)

      const secondProc = await DummyQueuedPersistedProcessor.create()
      await sleep(10)
      secondProc.handledQueue.length.should.equal(
        0,
        `No items should be processed!`
      )

      const item2: string = '111111'
      await secondProc.add(1, item2)
      await sleep(10)

      secondProc.handledQueue.length.should.equal(
        1,
        `Second item should be processed!`
      )
      secondProc.handledQueue[0].should.equal(item2, `Incorrect item processed`)
    })
  })
})
