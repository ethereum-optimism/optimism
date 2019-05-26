/* External Imports */
import debug from 'debug'
const log = debug('test:info:ownership-predicate')
import BigNum = require('bn.js')
import { AbiStateObject } from '@pigi/utils'

/* Internal Imports */
import { OwnershipTransaction } from '../..//src/ownership/transaction'

describe('OwnershipTransaction', () => {
  it.only('should initalize', async() => {
    const newStateObject = new AbiStateObject('0x7Fa6da9966869B56Dd08cb111Efed88FDF799545', '0x00')
    const ownershipTx = new OwnershipTransaction(
      '0x7Fa6da9966869B56Dd08cb111Efed88FDF799545',
      1,
      { start: new BigNum(1), end: new BigNum(10) },
      '0x00',
      { newStateObject },
      {
        v: '0x0000000000000000000000000000000000000000000000000000000000000000',
        r: '0x0000000000000000000000000000000000000000000000000000000000000000',
        s: '0x00',
      }
    )
    const decodedOwnershipTx = OwnershipTransaction.from(ownershipTx.encoded)
    log(ownershipTx)
    log(ownershipTx.encoded)
    log(decodedOwnershipTx)
    decodedOwnershipTx.should.deep.equal(ownershipTx)
  })
})
