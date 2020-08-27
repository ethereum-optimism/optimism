import '../setup'

import {
  SequentialProcessingDataService,
  SequentialProcessingItem,
} from '../../src/types/queue'
import { Row } from '../../src/types/db'
import {
  DefaultSequentialProcessingDataService,
  PostgresDB,
} from '../../src/app'

const sequenceKey = 'test'
const testItem = 'test item'
const tableName = 'sequential_processing'

describe('DB Test: SequentialProcessingDataService (Requires Postgres!)', () => {
  let postgres: PostgresDB
  let dataService: SequentialProcessingDataService
  before(() => {
    postgres = new PostgresDB('0.0.0.0', 5432, 'test', 'test', 'rollup')
    dataService = new DefaultSequentialProcessingDataService(postgres)
  })

  beforeEach(async () => {
    await postgres.execute(`DELETE FROM sequential_processing`)
  })

  after(async () => {
    await postgres.execute(`DELETE FROM sequential_processing`)
  })

  describe('persistItem()', async () => {
    it('should successfully persist item', async () => {
      await dataService.persistItem(0, testItem, sequenceKey)
      const res: Row[] = await postgres.select(`SELECT * FROM ${tableName}`)

      const exists: boolean = !!res
      exists.should.equal(true, `result should exist!`)
      res.length.should.equal(1, 'Incorrect number of results!')
      res[0]['sequence_number'].should.equal('0', 'Incorrect sequence number!')
      res[0]['sequence_key'].should.equal(
        sequenceKey,
        'Incorrect sequence key!'
      )
      res[0]['data_to_process'].should.equal(testItem, 'Incorrect data!')
      res[0]['processed'].should.equal(
        false,
        'Inserted data should not be processed!'
      )
    })

    it('should not conflict on re-persisting item', async () => {
      await dataService.persistItem(0, testItem, sequenceKey)
      await dataService.persistItem(0, 'dupe to be ignored', sequenceKey)
      const res: Row[] = await postgres.select(`SELECT * FROM ${tableName}`)

      const exists: boolean = !!res
      exists.should.equal(true, `result should exist!`)
      res.length.should.equal(1, 'Incorrect number of results!')
      res[0]['sequence_number'].should.equal('0', 'Incorrect sequence number!')
      res[0]['sequence_key'].should.equal(
        sequenceKey,
        'Incorrect sequence key!'
      )
      res[0]['data_to_process'].should.equal(testItem, 'Incorrect data!')
      res[0]['processed'].should.equal(
        false,
        'Inserted data should not be processed!'
      )
    })

    it('should persist as processed if set', async () => {
      await dataService.persistItem(0, testItem, sequenceKey, true)
      const res: Row[] = await postgres.select(`SELECT * FROM ${tableName}`)

      const exists: boolean = !!res
      exists.should.equal(true, `result should exist!`)
      res.length.should.equal(1, 'Incorrect number of results!')
      res[0]['sequence_number'].should.equal('0', 'Incorrect sequence number!')
      res[0]['sequence_key'].should.equal(
        sequenceKey,
        'Incorrect sequence key!'
      )
      res[0]['data_to_process'].should.equal(testItem, 'Incorrect data!')
      res[0]['processed'].should.equal(
        true,
        'Inserted data should be processed!'
      )
    })
  })

  describe('fetchItem()', async () => {
    it('should fetch persisted item', async () => {
      await dataService.persistItem(0, testItem, sequenceKey)
      const res: SequentialProcessingItem = await dataService.fetchItem(
        0,
        sequenceKey
      )

      const exists: boolean = !!res
      exists.should.equal(true, `result should exist!`)
      res.data.should.equal(testItem, 'Data mismatch!')
      res.processed.should.equal(false, 'Processed mismatch!')
    })

    it('should fetch persisted item that is processed', async () => {
      await dataService.persistItem(0, testItem, sequenceKey, true)
      const res: SequentialProcessingItem = await dataService.fetchItem(
        0,
        sequenceKey
      )

      const exists: boolean = !!res
      exists.should.equal(true, `result should exist!`)
      res.data.should.equal(testItem, 'Data mismatch!')
      res.processed.should.equal(true, 'Processed mismatch!')
    })

    it('should filter by sequenceKey persisted item that is processed', async () => {
      await dataService.persistItem(0, testItem, sequenceKey)
      const res: SequentialProcessingItem = await dataService.fetchItem(
        0,
        'derp'
      )

      const exists: boolean = !!res
      exists.should.equal(false, `result should not exist!`)
    })
  })

  describe('getLastIndexProcessed()', async () => {
    it('should return -1 by default', async () => {
      const lastProcessed: number = await dataService.getLastIndexProcessed(
        sequenceKey
      )
      lastProcessed.should.equal(-1, 'last processed mismatch!')
    })

    it('should return -1 if item is not processed', async () => {
      await dataService.persistItem(0, testItem, sequenceKey)

      const lastProcessed: number = await dataService.getLastIndexProcessed(
        sequenceKey
      )
      lastProcessed.should.equal(-1, 'last processed mismatch!')
    })

    it('should reflect last processed', async () => {
      await dataService.persistItem(0, testItem, sequenceKey, true)

      let lastProcessed: number = await dataService.getLastIndexProcessed(
        sequenceKey
      )
      lastProcessed.should.equal(0, 'last processed mismatch!')

      await dataService.persistItem(1, testItem, sequenceKey, true)
      lastProcessed = await dataService.getLastIndexProcessed(sequenceKey)
      lastProcessed.should.equal(1, 'last processed mismatch!')
    })
  })

  describe('updateToProcessed()', async () => {
    it('should not throw if there is no data', async () => {
      await dataService.updateToProcessed(23, sequenceKey)
    })

    it('should update record to processed', async () => {
      await dataService.persistItem(0, testItem, sequenceKey)
      let res: SequentialProcessingItem = await dataService.fetchItem(
        0,
        sequenceKey
      )

      let exists: boolean = !!res
      exists.should.equal(true, `result should exist!`)
      res.processed.should.equal(
        false,
        'Inserted data should not be processed!'
      )

      await dataService.updateToProcessed(0, sequenceKey)
      res = await dataService.fetchItem(0, sequenceKey)

      exists = !!res
      exists.should.equal(true, `result should exist!`)
      res.processed.should.equal(true, 'Inserted data should be processed!')
    })

    it('should not update record to processed if wrong sequence key', async () => {
      await dataService.persistItem(0, testItem, sequenceKey)
      let res: SequentialProcessingItem = await dataService.fetchItem(
        0,
        sequenceKey
      )

      let exists: boolean = !!res
      exists.should.equal(true, `result should exist!`)
      res.processed.should.equal(
        false,
        'Inserted data should not be processed!'
      )

      await dataService.updateToProcessed(0, 'derp')
      res = await dataService.fetchItem(0, sequenceKey)

      exists = !!res
      exists.should.equal(true, `result should exist!`)
      res.processed.should.equal(
        false,
        'Inserted data should not be processed!'
      )
    })

    it('should not update record to processed if wrong index', async () => {
      await dataService.persistItem(0, testItem, sequenceKey)
      let res: SequentialProcessingItem = await dataService.fetchItem(
        0,
        sequenceKey
      )

      let exists: boolean = !!res
      exists.should.equal(true, `result should exist!`)
      res.processed.should.equal(
        false,
        'Inserted data should not be processed!'
      )

      await dataService.updateToProcessed(1, sequenceKey)
      res = await dataService.fetchItem(0, sequenceKey)

      exists = !!res
      exists.should.equal(true, `result should exist!`)
      res.processed.should.equal(
        false,
        'Inserted data should not be processed!'
      )
    })
  })
})
