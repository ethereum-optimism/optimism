import { assert, should } from '../setup'

/* External Imports */
import debug from 'debug'
import { logError } from '@pigi/core-utils'

/* Internal Imports */
import { Batch, DB, DEL_BATCH_TYPE, PUT_BATCH_TYPE } from '../../src/types/db'
import { newInMemoryDB } from '../../src/app'

const log = debug('db', true)

describe('RangeDB', () => {
  let db: DB

  beforeEach(async () => {
    db = newInMemoryDB()
  })

  describe('batch', () => {
    it('should process put batch correctly', async () => {
      const testBuff: Buffer = Buffer.from(`test`)
      const testBuff2: Buffer = Buffer.from(`test2`)
      const batch: Batch[] = [
        {
          type: PUT_BATCH_TYPE,
          key: testBuff,
          value: testBuff,
        },
        {
          type: PUT_BATCH_TYPE,
          key: testBuff2,
          value: testBuff2,
        },
      ]

      try {
        await db.batch(batch)
      } catch (e) {
        logError(log, `Error processing put batch`, e)
        throw e
      }

      const res1: Buffer = await db.get(testBuff)
      res1.should.eql(
        testBuff,
        `Expected ${res1.toString()} to equal ${testBuff.toString()}`
      )

      const res2: Buffer = await db.get(testBuff2)
      res2.should.eql(
        testBuff2,
        `Expected ${res2.toString()} to equal ${testBuff2.toString()}`
      )
    })

    it('should process del batch correctly', async () => {
      const testBuff: Buffer = Buffer.from(`test`)
      const testBuff2: Buffer = Buffer.from(`test2`)

      await db.put(testBuff, testBuff)
      await db.put(testBuff2, testBuff2)

      const batch: Batch[] = [
        {
          type: DEL_BATCH_TYPE,
          key: testBuff,
        },
        {
          type: DEL_BATCH_TYPE,
          key: testBuff2,
        },
      ]

      try {
        await db.batch(batch)
      } catch (e) {
        logError(log, `Error processing put batch`, e)
        assert.fail()
      }

      const res1: Buffer = await db.get(testBuff)
      should.not.exist(res1, `${res1} should have been deleted`)

      const res2: Buffer = await db.get(testBuff2)
      should.not.exist(res2, `${res2} should have been deleted`)
    })

    it('should process mixed batch correctly', async () => {
      const testBuff: Buffer = Buffer.from(`test`)
      const testBuff2: Buffer = Buffer.from(`test2`)

      await db.put(testBuff, testBuff)
      await db.put(testBuff2, testBuff2)

      const testBuff3: Buffer = Buffer.from(`test2`)

      const batch: Batch[] = [
        {
          type: DEL_BATCH_TYPE,
          key: testBuff,
        },
        {
          type: PUT_BATCH_TYPE,
          key: testBuff2,
          value: testBuff3,
        },
        {
          type: PUT_BATCH_TYPE,
          key: testBuff3,
          value: testBuff3,
        },
      ]

      try {
        await db.batch(batch)
      } catch (e) {
        logError(log, `Error processing put batch`, e)
        assert.fail()
      }

      const res1: Buffer = await db.get(testBuff)
      should.not.exist(res1, `${res1} should have been deleted`)

      const res2: Buffer = await db.get(testBuff2)
      res2.should.eql(
        testBuff3,
        `${res2.toString()} should have been updated to ${testBuff3.toString()}`
      )

      const res3: Buffer = await db.get(testBuff3)
      res3.should.eql(
        testBuff3,
        `${res3.toString()} should have been set to ${testBuff3.toString()}`
      )
    })
  })
})
