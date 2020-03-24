/* External Imports */
import { BigNumber, ZERO } from '@eth-optimism/core-utils'
import debug from 'debug'
import MemDown from 'memdown'

/* Internal Imports */
import { dbRootPath } from '../setup'
import {BaseDB, RangeEntry, RangeBucket, getLevelInstance} from '../../src'

const log = debug('test:info:range-db')

const addDefaultRangesToDB = async (rangeDB) => {
  // Generate some ranges
  const ranges = []
  for (let i = 0; i < 10; i++) {
    const start = new BigNumber('' + i * 10, 'hex')
    const end = new BigNumber('' + (i + 1) * 10, 'hex')
    ranges.push({
      start,
      end,
    })
  }
  // Put them in our DB
  for (const range of ranges) {
    log(range.start.toString(16))
    await rangeDB.put(range.start, range.end, Buffer.from('Hello'))
  }
  return ranges
}

class StringRangeEntry {
  public stringRangeEntry
  constructor(rangeEntry: RangeEntry) {
    this.stringRangeEntry = {
      start: rangeEntry.start.toString('hex'),
      end: rangeEntry.end.toString('hex'),
      value: rangeEntry.value.toString(),
    }
  }
}

const testPutResults = async (
  db: RangeBucket,
  putContents: any[],
  expectedResults: any[]
): Promise<void> => {
  // First put the ranges
  putRanges(db, putContents)
  // Now check that they were added correctly
  const res = await db.get(ZERO, new BigNumber('100000000000', 'hex'))
  for (let i = 0; i < res.length; i++) {
    compareResult(res[i], expectedResults[i])
  }
}

const putRanges = async (
  db: RangeBucket,
  putContents: any[]
): Promise<void> => {
  for (const putContent of putContents) {
    await db.put(
      new BigNumber(putContent.start, 'hex'),
      new BigNumber(putContent.end, 'hex'),
      Buffer.from(putContent.value)
    )
  }
}

const compareResult = (res: any, expectedResult: any): void => {
  const strResult = new StringRangeEntry(res)
  strResult.stringRangeEntry.should.deep.equal(expectedResult)
}

describe('RangeDB', () => {
  const db = getLevelInstance(dbRootPath + 'rangeTest')
  let prefixCounter = 0
  let rangeDB

  beforeEach(async () => {
    const baseDB = new BaseDB(new MemDown('') as any)
    rangeDB = baseDB.rangeBucket(Buffer.from([prefixCounter++]))
  })

  it('allows puts on a range & get should return the range value which was put', async () => {
    const start = 0
    const end = 10
    await rangeDB.put(
      new BigNumber(start),
      new BigNumber(end),
      Buffer.from('Hello')
    )
    const res = await rangeDB.get(new BigNumber(start), new BigNumber(end))
    new BigNumber(res[0].start, 'hex').toNumber().should.equal(start)
    new BigNumber(res[0].end, 'hex').toNumber().should.equal(end)
  })

  it('returns an empty array if the db is empty', async () => {
    const getStart = 4
    const getEnd = 8
    const res = await rangeDB.get(
      new BigNumber(getStart),
      new BigNumber(getEnd)
    )
    res.length.should.equal(0)
  })

  it('returns a range which surrounds the range which you are getting', async () => {
    // This covers the case where the DB has one element of range 0-10, and you get 3-4, then it
    // should return the entire element which "surrounds" your get query.
    const start = 0
    const end = 10
    const getStart = 4
    const getEnd = 8
    await rangeDB.put(
      new BigNumber(start),
      new BigNumber(end),
      Buffer.from('Hello')
    )
    const res = await rangeDB.get(
      new BigNumber(getStart),
      new BigNumber(getEnd)
    )
    new BigNumber(res[0].start, 'hex').toNumber().should.equal(start)
    new BigNumber(res[0].end, 'hex').toNumber().should.equal(end)
    res.length.should.equal(1)
  })

  it('allows gets on all of the values that have been put', async () => {
    // Add some ranges to our db
    const ranges = await addDefaultRangesToDB(rangeDB)
    // Get them from our DB
    const gottenRanges = await rangeDB.get(
      ranges[0].start,
      ranges[ranges.length - 1].end
    )
    // Compare them to the ranges we put & got and make sure they are equal
    for (let i = 0; i < ranges.length; i++) {
      const start = ranges[i].start.toString(16)
      const end = ranges[i].end.toString(16)
      const gottenStart = gottenRanges[i].start.toString(16)
      const gottenEnd = gottenRanges[i].end.toString(16)
      log('Put start:', start, ' -- Got start:', gottenStart)
      log('Put end:', end, ' -- Got end:', gottenEnd)
      gottenStart.should.equal(start)
      gottenEnd.should.equal(end)
    }
    gottenRanges.length.should.equal(ranges.length)
  })

  it('allows gets a subset of the values that have been put', async () => {
    // Add some ranges to our db
    const ranges = await addDefaultRangesToDB(rangeDB)
    // This time get the ranges 22-
    const gottenRanges = await rangeDB.get(
      ranges[2].start.add(new BigNumber(2)),
      ranges[ranges.length - 2].end.sub(new BigNumber(2))
    )
    // Compare them to the ranges we put & got and make sure they are equal
    for (let i = 2; i < ranges.length - 1; i++) {
      const start = ranges[i].start.toString(16)
      const end = ranges[i].end.toString(16)
      const gottenStart = gottenRanges[i - 2].start.toString(16)
      const gottenEnd = gottenRanges[i - 2].end.toString(16)
      log('Put start:', start, ' -- Got start:', gottenStart)
      log('Put end:', end, ' -- Got end:', gottenEnd)
      gottenStart.should.equal(start)
      gottenEnd.should.equal(end)
    }
  })

  it('returns nothing when querying in between two other values', async () => {
    // Values added to the database: [0,10) & [20,30).
    // We will query [10,20) and it should return nothing.
    const start1 = new BigNumber('0', 'hex')
    const end1 = new BigNumber('10', 'hex')
    const start2 = new BigNumber('20', 'hex')
    const end2 = new BigNumber('30', 'hex')
    // Put range 1
    await rangeDB.put(start1, end1, Buffer.from('Hello'))
    // Put range 2
    await rangeDB.put(start2, end2, Buffer.from('world!'))
    // Check that if we query in between we don't get anything
    const res = await rangeDB.get(end1, start2)
    res.length.should.equal(0)
  })

  it('splits ranges which has been put in the middle of another range', async () => {
    // Surrounding: [10, 100), Inner: [50, 60), should result in [10, 50), [50, 60), [60, 100)
    const surroundingStart = new BigNumber('10', 'hex')
    const surroundingEnd = new BigNumber('100', 'hex')
    const innerStart = new BigNumber('50', 'hex')
    const innerEnd = new BigNumber('60', 'hex')
    // Put our surrounding ranges
    await rangeDB.put(surroundingStart, surroundingEnd, Buffer.from('Hello'))
    // Check that our range was added
    const res = await rangeDB.get(innerStart, innerEnd)
    // Now put the inner range
    await rangeDB.put(innerStart, innerEnd, Buffer.from('world!'))
    // Get all the ranges and see what we get
    const gottenRanges = await rangeDB.get(surroundingStart, surroundingEnd)
    // Print all the ranges
    for (const range of gottenRanges) {
      log(
        'start:',
        range.start.toString(16),
        '- end:',
        range.end.toString(16),
        '- value:',
        range.value.toString()
      )
    }
    // Check that the start and ends are correct
    // The first segment:
    gottenRanges[0].start.toString(16).should.equal('10')
    gottenRanges[0].end.toString(16).should.equal('50')
    // The second segment:
    gottenRanges[1].start.toString(16).should.equal('50')
    gottenRanges[1].end.toString(16).should.equal('60')
    // The third segment:
    gottenRanges[2].start.toString(16).should.equal('60')
    gottenRanges[2].end.toString(16).should.equal('100')
  })

  it('splits `put(0, 100, x), put(50, 150, y)` into (0, 50, x), (50, 150, y)', async () => {
    await testPutResults(
      rangeDB,
      [
        { start: '0', end: '100', value: 'x1' },
        { start: '50', end: '150', value: 'y1' },
      ],
      [
        { start: '0', end: '50', value: 'x1' },
        { start: '50', end: '150', value: 'y1' },
      ]
    )
  })

  it('splits `put(50, 150, x), put(0, 100, y)` into (0, 50, x), (50, 150, y)', async () => {
    await testPutResults(
      rangeDB,
      [
        { start: '50', end: '150', value: 'x2' },
        { start: '0', end: '100', value: 'y2' },
      ],
      [
        { start: '0', end: '100', value: 'y2' },
        { start: '100', end: '150', value: 'x2' },
      ]
    )
  })

  it('splits `put(0, 100, x), put(0, 100, y)` into (0, 100, y)', async () => {
    await testPutResults(
      rangeDB,
      [
        { start: '0', end: '100', value: 'x3' },
        { start: '0', end: '100', value: 'y3' },
      ],
      [{ start: '0', end: '100', value: 'y3' }]
    )
  })

  it('splits `put(0, 100, x), put(100, 200, y), put(50, 150, z)` into (0, 50, x), (50, 150, z), (150, 200, y)', async () => {
    await testPutResults(
      rangeDB,
      [
        { start: '0', end: '100', value: 'x4' },
        { start: '100', end: '200', value: 'y4' },
        { start: '50', end: '150', value: 'z4' },
      ],
      [
        { start: '0', end: '50', value: 'x4' },
        { start: '50', end: '150', value: 'z4' },
        { start: '150', end: '200', value: 'y4' },
      ]
    )
  })

  describe('iterator()', () => {
    it('allows nextRange() to be called by the iterator returning a RangeEntry instead of a KV', async () => {
      const testRanges = {
        inputs: [
          { start: '0', end: '100', value: 'x' },
          { start: '100', end: '200', value: 'y' },
          { start: '200', end: '225', value: 'z' },
        ],
        expectedResults: [
          { start: '0', end: '100', value: 'x' },
          { start: '100', end: '200', value: 'y' },
          { start: '200', end: '225', value: 'z' },
        ],
      }

      // Put our ranges
      await putRanges(rangeDB, testRanges.inputs)
      // Use a range iterator to get values we expect
      const it = rangeDB.iterator()
      const range0 = await it.nextRange()
      const range1 = await it.nextRange()
      const range2 = await it.nextRange()
      compareResult(range0, testRanges.expectedResults[0])
      compareResult(range1, testRanges.expectedResults[1])
      compareResult(range2, testRanges.expectedResults[2])
    })
  })
})
