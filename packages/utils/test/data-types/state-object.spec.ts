import '../setup'

/* External Imports */
import debug from 'debug'
const log = debug('test:info:state-object')

/* Internal Imports */
import { AbiStateObject } from '../../src/data-types/state-object'

describe('AbiStateObject', () => {
  it('should encoded & decode data without throwing', async () => {
    const stateObject = new AbiStateObject('0x2b5c5D7D87f2E6C2AC338Cb99a93B7A3aEcA823F', '0x1234')
    const stateObjectEncoding = stateObject.encoded
    const decodedStateObject = AbiStateObject.from(stateObjectEncoding)
    log('Original state object:\n', stateObject)
    log('State object encoded:\n', stateObjectEncoding)
    log('Decoded state object:\n', decodedStateObject)
    decodedStateObject.should.deep.equal(stateObject)
  })
})
