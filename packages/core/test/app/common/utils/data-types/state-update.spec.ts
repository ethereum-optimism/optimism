import '../../../../setup'

/* External Imports */
import BigNum = require('bn.js')
import debug from 'debug'
const log = debug('test:info:abi-stuff')

/* Internal Imports */
import { AbiStateObject, AbiStateUpdate, AbiRange, AbiOwnershipParameters, AbiOwnershipTransaction } from '../../../../../src/app/common/utils'

describe.only('AbiEncoding', () => {
  it('should encoded & decode AbiStateObject without throwing', async () => {
    const stateObject = new AbiStateObject('0x2b5c5D7D87f2E6C2AC338Cb99a93B7A3aEcA823F', '0x1234')
    const range = new AbiRange(new BigNum(10), new BigNum(30))
    const stateUpdate = new AbiStateUpdate(stateObject, range, 10, '0x3cDb4F0318a01f43dcf92eF09E10c05bF3bfc213')
    const stateUpdateEncoding = stateUpdate.encoded
    const decodedStateUpdate = AbiStateUpdate.from(stateUpdateEncoding)
    log('Original state object:\n', stateUpdate)
    log('State object encoded:\n', stateUpdateEncoding)
    log('Decoded state object:\n', decodedStateUpdate)
    log('Decoded state object encoded:\n', decodedStateUpdate.encoded)
    decodedStateUpdate.should.deep.equal(stateUpdate)
  })
  it('should encoded & decode AbiOwnershipParameters without throwing', async () => {
    const stateObject = new AbiStateObject('0x2b5c5D7D87f2E6C2AC338Cb99a93B7A3aEcA823F', '0x1234')
    const transactionParams = new AbiOwnershipParameters(stateObject, new BigNum(0), new BigNum(0))
    const transactionParamsEncoding = transactionParams.encoded
    const decodedTransactionParams = AbiOwnershipParameters.from(transactionParamsEncoding)
    log('params encoded:\n', transactionParamsEncoding)
    log('Decoded paraams encoded:\n', decodedTransactionParams.encoded)
    decodedTransactionParams.should.deep.equal(transactionParams)
  })
  it('should encoded & decode AbiOwnershipTransaction without throwing', async () => {
    const stateObject = new AbiStateObject('0x2b5c5D7D87f2E6C2AC338Cb99a93B7A3aEcA823F', '0x1234')
    const transactionParams = new AbiOwnershipParameters(stateObject, new BigNum(0), new BigNum(0))
    const depositContract = '0x2b5c5D7D87f2E6C2AC338Cb99a93B7A3aEcA823F'
    const methodId = '0x0000000000000000000000000000000000000000000000000000000000000000'
    const range = new AbiRange(new BigNum(10), new BigNum(30))
    const transaction = new AbiOwnershipTransaction(depositContract, methodId, transactionParams, range)
    const transactionParamsEncoding = transaction.encoded
    const decodedTransaction = AbiOwnershipTransaction.from(transactionParamsEncoding)
    decodedTransaction.should.deep.equal(transaction)
  })
})