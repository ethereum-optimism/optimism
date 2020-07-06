import '../setup'

/* External Imports */
import {
  hexStrToNumber,
  keccak256FromUtf8,
  TestUtils,
} from '@eth-optimism/core-utils'

import { Wallet } from 'ethers'
import { JsonRpcProvider } from 'ethers/providers'

/* Internal Imports */
import { BlockBatches, RollupTransaction } from '../../src/types'
import { DefaultL2NodeService } from '../../src/app'
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

const l1TxHash: string = keccak256FromUtf8('tx 1')
const batchNumber: number = 1

const nonce: number = 0
const gasLimit: number = 10_000
const sender: string = Wallet.createRandom().address
const target: string = Wallet.createRandom().address
const calldata: string = keccak256FromUtf8('calldata')
const rollupTx: RollupTransaction = {
  batchIndex: 1,
  gasLimit,
  nonce,
  sender,
  target,
  calldata,
  l1Timestamp: timestamp,
  l1BlockNumber: blockNumber,
  l1TxHash,
  queueOrigin: 1,
}

const nonce2: number = 1
const gasLimit2: number = 20_000
const sender2: string = Wallet.createRandom().address
const target2: string = Wallet.createRandom().address
const calldata2: string = keccak256FromUtf8('calldata 2')
const rollupTx2: RollupTransaction = {
  batchIndex: 2,
  gasLimit: gasLimit2,
  nonce: nonce2,
  sender: sender2,
  target: target2,
  calldata: calldata2,
  l1Timestamp: timestamp,
  l1BlockNumber: blockNumber,
  l1TxHash,
  queueOrigin: 1,
}

const deserializeBlockBatches = (serialized: string): BlockBatches => {
  return JSON.parse(serialized, (k, v) => {
    switch (k) {
      case 'blockNumber':
      case 'timestamp':
      case 'gasLimit':
      case 'nonce':
      case 'batchIndex':
      case 'l1BlockNumber':
      case 'l1Timestamp':
      case 'queueOrigin':
        return hexStrToNumber(v)
      default:
        return v
    }
  })
}

describe('L2 Node Service', () => {
  let l2NodeService: DefaultL2NodeService
  let mockedSendProvider: MockedProvider
  let wallet: Wallet

  beforeEach(async () => {
    mockedSendProvider = new MockedProvider()
    wallet = Wallet.createRandom().connect(mockedSendProvider)
    l2NodeService = new DefaultL2NodeService(wallet)
  })

  it('should handle undefined batch properly', async () => {
    await TestUtils.assertThrowsAsync(async () => {
      await l2NodeService.sendBlockBatches(undefined)
    })
  })

  it('should handle batch with undefined transactions properly', async () => {
    await l2NodeService.sendBlockBatches({
      batchNumber,
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
    await l2NodeService.sendBlockBatches({
      batchNumber,
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
    await l2NodeService.sendBlockBatches({
      batchNumber,
      timestamp,
      blockNumber,
      batches: [[rollupTx]],
    })

    mockedSendProvider.sent.length.should.equal(1, 'Should have sent tx!')
    mockedSendProvider.sent[0].method.should.equal(
      DefaultL2NodeService.sendBlockBatchesMethod,
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
    blockBatches.batches[0][0].should.deep.equal(
      rollupTx,
      'Incorrect transaction received!'
    )

    verifyMessage(payloadStr, signature).should.equal(
      wallet.address,
      'IncorrectSignature!'
    )
  })

  it('should send multi-tx batch properly', async () => {
    await l2NodeService.sendBlockBatches({
      batchNumber,
      timestamp,
      blockNumber,
      batches: [[rollupTx, rollupTx2]],
    })

    mockedSendProvider.sent.length.should.equal(1, 'Should have sent tx!')
    mockedSendProvider.sent[0].method.should.equal(
      DefaultL2NodeService.sendBlockBatchesMethod,
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
    blockBatches.batches[0][0].should.deep.equal(
      rollupTx,
      'Incorrect transaction received!'
    )

    blockBatches.batches[0][1].should.deep.equal(
      rollupTx2,
      'Incorrect transaction 2 received!'
    )

    verifyMessage(payloadStr, signature).should.equal(
      wallet.address,
      'IncorrectSignature!'
    )
  })
})
