import { should } from '../setup'

/* External Imports */
import { sleep } from '@eth-optimism/core-utils'

/* Internal Imports */
import { BaseQueuedPersistedProcessor, newInMemoryDB } from '../../src/app'
import { DB } from '../../src/types/db'

class DummyQueuedPersistedProcessor extends BaseQueuedPersistedProcessor<
  string
> {
  public handledQueue: string[]
  public static async create(
    db: DB,
    persistenceKey: string
  ): Promise<DummyQueuedPersistedProcessor> {
    const processor = new DummyQueuedPersistedProcessor(db, persistenceKey)
    await processor.init()
    return processor
  }

  private constructor(db: DB, persistenceKey: string) {
    super(db, persistenceKey)
    this.handledQueue = []
  }

  protected async handleNextItem(index: number, item: string): Promise<void> {
    this.handledQueue.push(item)
  }

  protected async serializeItem(item: string): Promise<Buffer> {
    return Buffer.from(item, 'utf-8')
  }

  protected async deserializeItem(itemBuffer: Buffer): Promise<string> {
    return itemBuffer.toString('utf-8')
  }
}

describe.only('Queued Persisted Processor', () => {
  let db: DB
  let processor: DummyQueuedPersistedProcessor
  const persistenceKey: string = 'derp'

  beforeEach(async () => {
    db = newInMemoryDB()
    processor = await DummyQueuedPersistedProcessor.create(db, persistenceKey)
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
      await sleep(10)
      processor.handledQueue.length.should.equal(
        1,
        `Incorrect number processed!`
      )
      processor.handledQueue[0].should.equal(first, `Incorrect item processed!`)

      await processor.markProcessed(0)
      await sleep(10)

      processor.handledQueue.length.should.equal(
        2,
        `Incorrect number processed!`
      )
      processor.handledQueue[1].should.equal(
        second,
        `Incorrect item processed!`
      )
    })

    it('handles next item added after previous is acknowledged', async () => {
      const first = 'Number 0!'
      const second = 'Number 1!'
      await processor.add(0, first)
      await sleep(10)
      processor.handledQueue.length.should.equal(
        1,
        `Incorrect number processed!`
      )
      processor.handledQueue[0].should.equal(first, `Incorrect item processed!`)

      await processor.markProcessed(0)
      await sleep(10)

      processor.handledQueue.length.should.equal(
        1,
        `Incorrect number processed!`
      )

      await processor.add(1, second)
      await sleep(10)
      processor.handledQueue.length.should.equal(
        2,
        `Incorrect number processed!`
      )
      processor.handledQueue[1].should.equal(
        second,
        `Incorrect item processed!`
      )
    })
  })

  describe('Start with existing state', () => {
    it('restarts with existing state (empty)', async () => {
      const secondProc = await DummyQueuedPersistedProcessor.create(db, persistenceKey)
      await sleep(10)
      secondProc.handledQueue.length.should.equal(0, `No items should be processed!`)

      const item = '0000'
      await secondProc.add(0, item)
      await sleep(10)
      secondProc.handledQueue.length.should.equal(1, `Item should have been processed`)
      secondProc.handledQueue[0].should.equal(item, `Incorrect item processed`)
    })
  })
})
