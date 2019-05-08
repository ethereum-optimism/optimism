import { dbRootPath } from '../setup'

/* External Imports */
import debug from 'debug'
const log = debug('test:info:range-db')
import level = require('level')
import BigNum = require('bn.js')

/* Internal Imports */
import { RangeDB } from '../../src/state-manager/range-db'

describe.only('RangeDB', () => {
  const db = level(dbRootPath + 'rangeTest', { keyEncoding: 'binary', valueEncoding: 'binary' })
  let prefixCounter = 0
  let rangeDB

  beforeEach(async () => {
    rangeDB = new RangeDB(db, Buffer.from([prefixCounter++]))
  })

  it('should allow puts on a range & get should return the range value which was put', async() => {
    const start = 0
    const end = 10
    await rangeDB.put(new BigNum(start), new BigNum(end), Buffer.from('hello!'))
    const res = await rangeDB.get(new BigNum(start), new BigNum(end))
    new BigNum(res[0].start, 'hex').toNumber().should.equal(start)
    new BigNum(res[0].end, 'hex').toNumber().should.equal(end)
  })

  it('should return an empty array if the db is empty', async() => {
    const getStart = 4
    const getEnd = 8
    const res = await rangeDB.get(new BigNum(getStart), new BigNum(getEnd))
    res.length.should.equal(0)
  })

  it('should return a range which surrounds the range which you are getting', async() => {
    // This covers the case where the DB has one element of range 0-10, and you get 3-4, then it
    // should return the entire element which "surrounds" your get query.
    const start = 0
    const end = 10
    const getStart = 4
    const getEnd = 8
    await rangeDB.put(new BigNum(start), new BigNum(end), Buffer.from('hello!'))
    const res = await rangeDB.get(new BigNum(getStart), new BigNum(getEnd))
    new BigNum(res[0].start, 'hex').toNumber().should.equal(start)
    new BigNum(res[0].end, 'hex').toNumber().should.equal(end)
    res.length.should.equal(1)
  })

  it('should allow puts on multiple ranges', async() => {
    // Generate some ranges
    const ranges = []
    for (let i = 0; i < 10; i++) {
      const start = new BigNum('' + i*10, 'hex')
      const end = new BigNum('' + (i + 1)*10, 'hex')
      ranges.push({
        start,
        end
      })
    }
    // Put them in our DB
    for (const range of ranges) {
      log(range.start.toString(16))
      rangeDB.put(range.start, range.end, Buffer.from('hello!'))
    }
    // Get them from our DB
    const gottenRanges = await rangeDB.get(ranges[0].start, ranges[ranges.length-1].end)
    // Compare them to the ranges we put & got and make sure they are equal
    for (let i = 0; i < ranges.length; i++) {
      const start = ranges[i].start.toString(16)
      const end = ranges[i].end.toString(16)
      const gottenStart = ranges[i].start.toString(16)
      const gottenEnd = ranges[i].end.toString(16)
      log('Put start:', start, ' -- Got start:', gottenStart)
      log('Put end:', end, ' -- Got end:', gottenEnd)
      gottenStart.should.equal(start)
      gottenEnd.should.equal(end)
    }
    gottenRanges.length.should.equal(ranges.length)
  })
})
