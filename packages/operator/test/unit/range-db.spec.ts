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

  it('should allow puts on a range', async() => {
    const start = new BigNum('10', 'hex')
    const end = new BigNum('100', 'hex')
    rangeDB.put(start, end, Buffer.from('hello!'))
    rangeDB.get(start, end.addn(10))
    log('success!')
  })
})
