import { dbRootPath } from '../setup'

/* External Imports */
import debug from 'debug'
const log = debug('test:info:state-manager')
import level = require('level')

// import BigNum = require('bn.js')

/* Internal Imports */
import { OwnershipState } from '../../src/state-manager/state'

describe.only('merkle-index-tree', () => {
  const db = level(dbRootPath)
  const test = new OwnershipState(db)
  test.applyTransaction(Buffer.from('testing testing 123'))
})
