import '../setup'

/* External Imports */
import { hexStrToNumber, keccak256, TestUtils } from '@eth-optimism/core-utils'

import { Wallet } from 'ethers'
import { JsonRpcProvider } from 'ethers/providers'

/* Internal Imports */
import { BlockBatches, RollupTransaction } from '../../src/types'
import { BlockBatchSubmitter } from '../../src/app'
import { verifyMessage } from 'ethers/utils'

interface Payload {
  method: string
  params: any
}

class MockedProvider extends JsonRpcProvider {
  public readonly sent: Payload[]

  constructor() {
    super()
    this.sent = []
  }

  public async send(method: string, params: any): Promise<any> {
    this.sent.push({ method, params })
    return 'dope.'
  }
}

const timestamp: number = 123
const timestamp2: number = 1234

const blockNumber: number = 0
const blockNumber2: number = 1

const nonce: number = 0
const gasLimit: number = 10_000
const sender: string = Wallet.createRandom().address
const target: string = Wallet.createRandom().address
const calldata: string = keccak256(Buffer.from('calldata').toString('hex'))
const rollupTx: RollupTransaction = {
  gasLimit,
  nonce,
  sender,
  target,
  calldata,
}

const nonce2: number = 1
const gasLimit2: number = 20_000
const sender2: string = Wallet.createRandom().address
const target2: string = Wallet.createRandom().address
const calldata2: string = keccak256(Buffer.from('calldata 2').toString('hex'))
const rollupTx2: RollupTransaction = {
  gasLimit: gasLimit2,
  nonce: nonce2,
  sender: sender2,
  target: target2,
  calldata: calldata2,
}

const rollupTxsEqual = (
  one: RollupTransaction,
  two: RollupTransaction
): boolean => {
  return JSON.stringify(one) === JSON.stringify(two)
}

const deserializeBlockBatches = (serialized: string): BlockBatches => {
  return JSON.parse(serialized, (k, v) => {
    switch (k) {
      case 'blockNumber':
      case 'timestamp':
      case 'gasLimit':
      case 'nonce':
        return hexStrToNumber(v)
      default:
        return v
    }
  })
}

describe('L2 Transaction Batch Submitter', () => {
  let blockBatchSubmitter: BlockBatchSubmitter
  let mockedSendProvider: MockedProvider
  let wallet: Wallet

  beforeEach(async () => {
    mockedSendProvider = new MockedProvider()
    wallet = Wallet.createRandom().connect(mockedSendProvider)
    blockBatchSubmitter = new BlockBatchSubmitter(wallet)
  })

  it('should handle undefined batch properly', async () => {
    await TestUtils.assertThrowsAsync(async () => {
      await blockBatchSubmitter.handleBlockBatches(undefined)
    })
  })

  it('should handle batch with undefined transactions properly', async () => {
    await blockBatchSubmitter.handleBlockBatches({
      timestamp,
      blockNumber,
      batches: undefined,
    })

    mockedSendProvider.sent.length.should.equal(
      0,
      'Should not have sent anything!'
    )
  })

  it('should handle batch with empty transactions properly', async () => {
    await blockBatchSubmitter.handleBlockBatches({
      timestamp,
      blockNumber,
      batches: [],
    })

    mockedSendProvider.sent.length.should.equal(
      0,
      'Should not have sent anything!'
    )
  })

  it('should send single-tx batch properly', async () => {
    await blockBatchSubmitter.handleBlockBatches({
      timestamp,
      blockNumber,
      batches: [[rollupTx]],
    })

    mockedSendProvider.sent.length.should.equal(1, 'Should have sent tx!')
    mockedSendProvider.sent[0].method.should.equal(
      BlockBatchSubmitter.sendBlockBatchesMethod,
      'Sent to incorrect Web3 method!'
    )
    Array.isArray(mockedSendProvider.sent[0].params).should.equal(
      true,
      'Incorrect params type!'
    )
    const paramsArray = mockedSendProvider.sent[0].params as string[]
    paramsArray.length.should.equal(2, 'Incorrect params length')
    const [payloadStr, signature] = paramsArray

    const blockBatches: BlockBatches = deserializeBlockBatches(payloadStr)

    blockBatches.timestamp.should.equal(timestamp, 'Incorrect timestamp!')
    blockBatches.batches.length.should.equal(1, 'Incorrect num batches!')
    blockBatches.batches[0].length.should.equal(1, 'Incorrect num txs!')
    rollupTxsEqual(blockBatches.batches[0][0], rollupTx).should.equal(
      true,
      'Incorrect transaction received!'
    )

    verifyMessage(payloadStr, signature).should.equal(
      wallet.address,
      'IncorrectSignature!'
    )
  })

  it('should send multi-tx batch properly', async () => {
    await blockBatchSubmitter.handleBlockBatches({
      timestamp,
      blockNumber,
      batches: [[rollupTx, rollupTx2]],
    })

    mockedSendProvider.sent.length.should.equal(1, 'Should have sent tx!')
    mockedSendProvider.sent[0].method.should.equal(
      BlockBatchSubmitter.sendBlockBatchesMethod,
      'Sent to incorrect Web3 method!'
    )
    Array.isArray(mockedSendProvider.sent[0].params).should.equal(
      true,
      'Incorrect params type!'
    )
    const paramsArray = mockedSendProvider.sent[0].params as string[]
    paramsArray.length.should.equal(2, 'Incorrect params length')
    const [payloadStr, signature] = paramsArray

    const blockBatches: BlockBatches = deserializeBlockBatches(payloadStr)

    blockBatches.timestamp.should.equal(timestamp, 'Incorrect timestamp!')
    blockBatches.batches.length.should.equal(1, 'Incorrect num batches!')
    blockBatches.batches[0].length.should.equal(
      2,
      'Incorrect num transactions!'
    )
    rollupTxsEqual(blockBatches.batches[0][0], rollupTx).should.equal(
      true,
      'Incorrect transaction received!'
    )

    rollupTxsEqual(blockBatches.batches[0][1], rollupTx2).should.equal(
      true,
      'Incorrect transaction 2 received!'
    )

    verifyMessage(payloadStr, signature).should.equal(
      wallet.address,
      'IncorrectSignature!'
    )
  })
})
