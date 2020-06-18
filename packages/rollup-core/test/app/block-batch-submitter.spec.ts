import '../setup'

/* External Imports */
import { hexStrToNumber, keccak256, TestUtils, numberToHexString } from '@eth-optimism/core-utils'

import { Wallet } from 'ethers'
import { JsonRpcProvider } from 'ethers/providers'

/* Internal Imports */
import { RollupTransaction } from '../../src/types'
import { BlockBatchSubmitter, GAS_LIMIT } from '../../src/app'
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
const sender: string = Wallet.createRandom().address
const target: string = Wallet.createRandom().address
const calldata: string = keccak256(Buffer.from('calldata').toString('hex'))
const gasLimit: string = numberToHexString(GAS_LIMIT)
const rollupTx: RollupTransaction = {
  nonce,
  sender,
  target,
  calldata,
  gasLimit
}

const nonce2: number = 1
const sender2: string = Wallet.createRandom().address
const target2: string = Wallet.createRandom().address
const calldata2: string = keccak256(Buffer.from('calldata 2').toString('hex'))
const rollupTx2: RollupTransaction = {
  nonce: nonce2,
  sender: sender2,
  target: target2,
  calldata: calldata2,
  gasLimit
}

const rollupTxsEqual = (
  one: RollupTransaction,
  two: RollupTransaction
): boolean => {
  return JSON.stringify(one) === JSON.stringify(two)
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
    paramsArray.length.should.equal(3, 'Incorrect params length')
    const [timestampStr, batchesStr, signature] = paramsArray

    hexStrToNumber(timestampStr).should.equal(timestamp, 'Incorrect timestamp!')
    const parsedBatches = JSON.parse(batchesStr) as any[]
    parsedBatches.length.should.equal(1, 'Incorrect num batches!')
    parsedBatches[0].length.should.equal(1, 'Incorrect num txs!')
    parsedBatches[0][0].nonce = hexStrToNumber(parsedBatches[0][0].nonce)
    rollupTxsEqual(parsedBatches[0][0], rollupTx).should.equal(
      true,
      'Incorrect transaction received!'
    )

    verifyMessage(batchesStr, signature).should.equal(
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
    paramsArray.length.should.equal(3, 'Incorrect params length')
    const [timestampStr, txsStr, signature] = paramsArray

    hexStrToNumber(timestampStr).should.equal(timestamp, 'Incorrect timestamp!')
    const parsedBatches = JSON.parse(txsStr) as any[]
    parsedBatches.length.should.equal(1, 'Incorrect num batches!')
    parsedBatches[0].length.should.equal(2, 'Incorrect num transactions!')
    parsedBatches[0] = parsedBatches[0].map((x) => {
      x.nonce = hexStrToNumber(x.nonce)
      return x
    })
    rollupTxsEqual(parsedBatches[0][0], rollupTx).should.equal(
      true,
      'Incorrect transaction received!'
    )

    rollupTxsEqual(parsedBatches[0][1], rollupTx2).should.equal(
      true,
      'Incorrect transaction 2 received!'
    )

    verifyMessage(txsStr, signature).should.equal(
      wallet.address,
      'IncorrectSignature!'
    )
  })
})
