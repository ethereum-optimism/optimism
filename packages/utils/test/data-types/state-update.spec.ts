import '../setup'

/* External Imports */
import BigNum = require('bn.js')
import debug from 'debug'
const log = debug('test:info:state-update')

/* Internal Imports */
import { AbiStateObject, AbiStateUpdate, AbiRange } from '../../src/data-types'

describe('AbiStateObject', () => {
  it('should encoded & decode data without throwing', async () => {
    const stateObject = new AbiStateObject('0x2b5c5D7D87f2E6C2AC338Cb99a93B7A3aEcA823F', '0x1234')
    const stateUpdate = new AbiStateUpdate(stateObject, new AbiRange(new BigNum(10), new BigNum(30)), 10, '0x3cDb4F0318a01f43dcf92eF09E10c05bF3bfc213')
    const stateUpdateEncoding = stateUpdate.encoded
    const decodedStateUpdate = AbiStateUpdate.from(stateUpdateEncoding)
    log('Original state object:\n', stateUpdate)
    log('State object encoded:\n', stateUpdateEncoding)
    log('Decoded state object:\n', decodedStateUpdate)
    log('Decoded state object encoded:\n', decodedStateUpdate.encoded)
    decodedStateUpdate.should.deep.equal(stateUpdate)
  })
})
