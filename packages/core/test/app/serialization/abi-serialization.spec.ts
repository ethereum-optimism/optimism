import '../../setup'

/* External Imports */
import debug from 'debug'
const log = debug('test:info:abi-stuff')

/* Internal Imports */
import {
  AbiStateObject,
  AbiStateUpdate,
  AbiRange,
  AbiOwnershipBody,
  AbiOwnershipTransaction,
  BigNumber,
} from '../../../src/app'

describe('AbiEncoding', () => {
  it('should encoded & decode AbiStateUpdate without throwing', async () => {
    const stateObject = new AbiStateObject(
      '0x2b5c5D7D87f2E6C2AC338Cb99a93B7A3aEcA823F',
      '0x1234'
    )
    const range = new AbiRange(new BigNumber(10), new BigNumber(30))
    const stateUpdate = new AbiStateUpdate(
      stateObject,
      range,
      new BigNumber(10),
      '0x3cDb4F0318a01f43dcf92eF09E10c05bF3bfc213'
    )
    const stateUpdateEncoding = stateUpdate.encoded
    const decodedStateUpdate = AbiStateUpdate.from(stateUpdateEncoding)
    log('Original state object:\n', stateUpdate)
    log('State object encoded:\n', stateUpdateEncoding)
    log('Decoded state object:\n', decodedStateUpdate)
    log('Decoded state object encoded:\n', decodedStateUpdate.encoded)
    decodedStateUpdate.should.deep.equal(stateUpdate)
  })
  it('should encoded & decode AbiOwnershipParameters without throwing', async () => {
    const stateObject = new AbiStateObject(
      '0x2b5c5D7D87f2E6C2AC338Cb99a93B7A3aEcA823F',
      '0x1234'
    )
    const transactionBody = new AbiOwnershipBody(
      stateObject,
      new BigNumber(0),
      new BigNumber(0)
    )
    const transactionBodyEncoding = transactionBody.encoded
    const decodedTransactionBody = AbiOwnershipBody.from(
      transactionBodyEncoding
    )
    log('body encoded:\n', transactionBodyEncoding)
    log('Decoded body encoded:\n', decodedTransactionBody.encoded)
    decodedTransactionBody.should.deep.equal(transactionBody)
  })
  it('should encoded & decode AbiOwnershipTransaction without throwing', async () => {
    const stateObject = new AbiStateObject(
      '0x2b5c5D7D87f2E6C2AC338Cb99a93B7A3aEcA823F',
      '0x1234'
    )
    const transactionBody = new AbiOwnershipBody(
      stateObject,
      new BigNumber(0),
      new BigNumber(0)
    )
    const depositContract = '0x2b5c5D7D87f2E6C2AC338Cb99a93B7A3aEcA823F'
    const range = new AbiRange(new BigNumber(10), new BigNumber(30))
    const transaction = new AbiOwnershipTransaction(
      depositContract,
      range,
      transactionBody
    )
    const transactionBodyEncoding = transaction.encoded
    const decodedTransaction = AbiOwnershipTransaction.from(
      transactionBodyEncoding
    )
    decodedTransaction.should.deep.equal(transaction)
  })
})
