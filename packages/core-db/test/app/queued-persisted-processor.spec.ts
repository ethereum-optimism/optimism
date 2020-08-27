import { should } from '../setup'

/* External Imports */
import { sleep } from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  BaseQueuedPersistedProcessor,
  InMemoryProcessingDataService,
  SequentialProcessingItem,
} from '../../src'
import { SequentialProcessingDataService } from '../../src/types/queue'

const persistenceKey: string = 'test'

class DummyQueuedPersistedProcessor extends BaseQueuedPersistedProcessor<
  string
> {
  public callMarkProcessed: boolean = true
  public throwOnceHandlingNextItem: boolean = false
  public handledQueue: string[]

  public static async create(
    dataService: SequentialProcessingDataService,
    startIndex: number = 0,
    retrySleepDelayMillis: number = 1000
  ): Promise<DummyQueuedPersistedProcessor> {
    const processor = new DummyQueuedPersistedProcessor(
      dataService,
      startIndex,
      retrySleepDelayMillis
    )
    await processor.init()
    return processor
  }

  private constructor(
    dataService: SequentialProcessingDataService,
    startIndex: number = 0,
    retrySleepDelayMillis: number = 1000
  ) {
    super(dataService, persistenceKey, startIndex, retrySleepDelayMillis)
    this.handledQueue = []
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
}

describe('Queued Persisted Processor', () => {
  let dataService: InMemoryProcessingDataService
  let processor: DummyQueuedPersistedProcessor
  const retrySleepDelayMillis: number = 100

  beforeEach(async () => {
    dataService = new InMemoryProcessingDataService()
    processor = await DummyQueuedPersistedProcessor.create(
      dataService,
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

    it('wont handle item if already processed', async () => {
      const item = 'Number 0!'
      dataService.lastProcessedIndex.set(persistenceKey, 0)
      await processor.add(0, item)
      await sleep(10)
      processor.handledQueue.length.should.equal(0, `Queue item processed!`)
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
      const secondProc = await DummyQueuedPersistedProcessor.create(dataService)
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

      const secondProc = await DummyQueuedPersistedProcessor.create(
        dataService,
        0
      )
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

      const secondProc = await DummyQueuedPersistedProcessor.create(
        dataService,
        0
      )
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

      const secondProc = await DummyQueuedPersistedProcessor.create(
        dataService,
        0
      )
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
