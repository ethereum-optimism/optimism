import '../setup'

import { SequentialProcessingDataService } from '../../src/types/queue'
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
  })
})
