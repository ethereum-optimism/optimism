/* External Imports */
import {
  add0x,
  keccak256FromUtf8,
  remove0x,
  ZERO_ADDRESS,
} from '@eth-optimism/core-utils'

import { ethers, Wallet } from 'ethers'
import { Log } from 'ethers/providers'
import { TransactionResponse } from 'ethers/providers/abstract-provider'

/* Internal Imports */
import {
  L1ToL2TxEnqueuedLogHandler,
  DefaultDataService,
  RollupTransaction,
  QueueOrigin,
  CalldataTxEnqueuedLogHandler,
  L1ToL2BatchAppendedLogHandler,
  SafetyQueueBatchAppendedLogHandler,
  SequencerBatchAppendedLogHandler,
  StateBatchAppendedLogHandler,
  CHAIN_ID,
} from '../../src'
import {
  arrayify,
  joinSignature,
  keccak256,
  parseTransaction,
  recoverAddress,
  resolveProperties,
  serializeTransaction,
  Transaction,
  UnsignedTransaction,
  verifyMessage,
} from 'ethers/utils'

const abi = new ethers.utils.AbiCoder()

const createLog = (
  data: string,
  txHash: string = keccak256FromUtf8('tx hash'),
  txIndex: number = 0,
  txLogIndex: number = 0
): Log => {
  return {
    blockNumber: 1,
    blockHash: keccak256FromUtf8('block'),
    transactionHash: txHash,
    transactionIndex: txIndex,
    transactionLogIndex: txLogIndex,
    address: ZERO_ADDRESS,
    data,
    logIndex: txLogIndex,
    topics: [],
  }
}

const createTx = (
  calldata: string,
  txHash: string = keccak256FromUtf8('tx hash'),
  blockNumber: number = 1,
  timestamp: number = 1
): TransactionResponse => {
  return {
    hash: txHash,
    blockHash: keccak256FromUtf8('block'),
    blockNumber,
    timestamp,
    from: Wallet.createRandom().address,
    to: Wallet.createRandom().address,
    nonce: 1,
    data: calldata,
    chainId: 108,
    confirmations: 0,
    gasLimit: undefined,
    gasPrice: undefined,
    value: undefined,
    wait: undefined,
  }
}

class MockDataService extends DefaultDataService {
  public createdL1ToL2Batches: number = 0
  public createdSafetyQueueBatches: number = 0
  public rollupTransactionsInserted: RollupTransaction[][] = []
  public txHashToRollupRootsInserted: Map<string, string[]> = new Map<
    string,
    string[]
  >()

  public txHashBatchesCreated: Set<string> = new Set<string>()

  constructor() {
    super(undefined)
  }

  public async queueNextGethSubmission(
    queueOrigins: number[]
  ): Promise<number> {
    if (queueOrigins.length !== 1) {
      throw Error(
        `There should only be 1 queue origin in filter but received ${
          queueOrigins.length
        }: ${queueOrigins.join(',')}`
      )
    }
    if (queueOrigins[0] === QueueOrigin.L1_TO_L2_QUEUE) {
      return ++this.createdL1ToL2Batches
    } else {
      return ++this.createdSafetyQueueBatches
    }
  }

  public async insertL1RollupTransactions(
    l1TxHash: string,
    rollupTransactions: RollupTransaction[],
    createBatch: boolean = false
  ): Promise<number> {
    this.rollupTransactionsInserted.push(rollupTransactions)
    if (createBatch) {
      this.txHashBatchesCreated.add(l1TxHash)
      return this.txHashBatchesCreated.size
    }
    return undefined
  }

  public async insertL1RollupStateRoots(
    l1TxHash: string,
    stateRoots: string[]
  ): Promise<number> {
    this.txHashToRollupRootsInserted.set(l1TxHash, stateRoots)
    return undefined
  }
}

const wallet = Wallet.createRandom()

const getTxSignature = async (
  to: string,
  nonce: string,
  gasLimit: string,
  data: string,
  w: Wallet = wallet
): Promise<string> => {
  const trans = {
    to: add0x(to),
    nonce: add0x(nonce),
    gasLimit: add0x(gasLimit),
    data: add0x(data),
    value: 0,
    chainId: CHAIN_ID,
  }

  const sig = await w.sign(trans)
  const t: Transaction = parseTransaction(sig)

  return add0x(`${t.r}${remove0x(t.s)}${t.v.toString(16)}`)
}

describe('Log Handlers', () => {
  let dataService: MockDataService
  beforeEach(() => {
    dataService = new MockDataService()
  })

  it('should parse and insert L1ToL2Tx', async () => {
    const sender: string = 'aa'.repeat(20)
    const target: string = 'bb'.repeat(20)
    const gasLimit: string = '00'.repeat(32)
    const calldata: string = 'abcd'.repeat(40)

    const l = createLog(
      abi.encode(
        ['address', 'address', 'uint32', 'bytes'],
        [sender, target, gasLimit, calldata].map((el) => {
          return add0x(el)
        })
      )
    )
    const tx = createTx('00'.repeat(64))

    await L1ToL2TxEnqueuedLogHandler(dataService, l, tx)

    dataService.rollupTransactionsInserted.length.should.equal(
      1,
      `No tx batch inserted!`
    )
    dataService.rollupTransactionsInserted[0].length.should.equal(
      1,
      `No tx inserted!`
    )
    const received: RollupTransaction =
      dataService.rollupTransactionsInserted[0][0]

    received.l1BlockNumber.should.equal(tx.blockNumber, 'Block number mismatch')
    received.l1Timestamp.should.equal(tx.timestamp, 'Timestamp mismatch')
    received.l1TxHash.should.equal(l.transactionHash, 'Tx hash mismatch')
    received.l1TxIndex.should.equal(l.transactionIndex, 'Tx index mismatch')
    received.l1TxLogIndex.should.equal(l.logIndex, 'Tx log index mismatch')
    received.queueOrigin.should.equal(
      QueueOrigin.L1_TO_L2_QUEUE,
      'Queue Origin mismatch'
    )
    received.indexWithinSubmission.should.equal(0, 'Batch index mismatch')
    received.sender.should.equal(l.address, 'Sender mismatch')
    remove0x(received.l1MessageSender)
      .toLowerCase()
      .should.equal(sender, 'L1 Message Sender mismatch')
    remove0x(received.target)
      .toLowerCase()
      .should.equal(target, 'Target mismatch')
    received.gasLimit.should.equal(0, 'Gas Limit mismatch')
    remove0x(received.calldata).should.equal(calldata, 'Calldata mismatch')

    dataService.txHashBatchesCreated.size.should.equal(
      0,
      'Should not have created batch!'
    )
  })

  it('should parse and insert Slow Queue Tx', async () => {
    const target: string = 'bb'.repeat(20)
    const nonce: string = '00'.repeat(32)
    const gasLimit: string = '00'.repeat(31) + '01'
    const calldata: string = 'abcd'.repeat(40)

    const signature = await getTxSignature(target, nonce, gasLimit, calldata)

    const data = `0x22222222${target}${nonce}${gasLimit}${remove0x(
      signature
    )}${calldata}`

    const l = createLog('00'.repeat(64))
    const tx = createTx(data)

    await CalldataTxEnqueuedLogHandler(dataService, l, tx)

    dataService.rollupTransactionsInserted.length.should.equal(
      1,
      `No tx batch inserted!`
    )
    dataService.rollupTransactionsInserted[0].length.should.equal(
      1,
      `No tx inserted!`
    )
    const received: RollupTransaction =
      dataService.rollupTransactionsInserted[0][0]

    received.l1BlockNumber.should.equal(tx.blockNumber, 'Block number mismatch')
    received.l1Timestamp.should.equal(tx.timestamp, 'Timestamp mismatch')
    received.l1TxHash.should.equal(l.transactionHash, 'Tx hash mismatch')
    received.l1TxIndex.should.equal(l.transactionIndex, 'Tx index mismatch')
    received.l1TxLogIndex.should.equal(l.logIndex, 'Tx log index mismatch')
    received.queueOrigin.should.equal(
      QueueOrigin.SAFETY_QUEUE,
      'Queue Origin mismatch'
    )
    received.indexWithinSubmission.should.equal(0, 'Batch index mismatch')
    remove0x(received.sender).should.equal(
      remove0x(wallet.address),
      'Sender mismatch'
    )
    remove0x(received.target).should.equal(target, 'Target mismatch')
    received.nonce.should.equal(0, 'Nonce mismatch')
    received.gasLimit.should.equal(1, 'Gas Limit mismatch')
    remove0x(received.signature).should.equal(
      remove0x(signature),
      'Signature mismatch'
    )
    remove0x(received.calldata).should.equal(calldata, 'Calldata mismatch')

    dataService.txHashBatchesCreated.size.should.equal(
      0,
      'Should not have created batch!'
    )
  })

  it('should append L1ToL2Batch on L1ToL2BatchAppendedLogHandler call', async () => {
    dataService.createdL1ToL2Batches.should.equal(
      0,
      'starting batch count should be 0!'
    )
    await L1ToL2BatchAppendedLogHandler(
      dataService,
      createLog(''),
      createTx('')
    )
    dataService.createdL1ToL2Batches.should.equal(1, 'batch not created!')
  })

  it('should append L1ToL2Batch on L1ToL2BatchAppendedLogHandler call', async () => {
    dataService.createdSafetyQueueBatches.should.equal(
      0,
      'starting batch count should be 0!'
    )
    await SafetyQueueBatchAppendedLogHandler(
      dataService,
      createLog(''),
      createTx('')
    )
    dataService.createdSafetyQueueBatches.should.equal(1, 'batch not created!')
  })

  it('should parse and insert Sequencer Batch', async () => {
    const timestamp = 1

    const target: string = 'bb'.repeat(20)
    const nonce: string = '00'.repeat(32)
    const gasLimit: string = '00'.repeat(31) + '01'
    const calldata: string = 'abcd'.repeat(40)

    const signature = await getTxSignature(target, nonce, gasLimit, calldata)

    let data = `0x${target}${nonce}${gasLimit}${remove0x(signature)}${calldata}`
    data = abi.encode(['bytes[]', 'uint256'], [[data, data, data], timestamp])

    const l = createLog('00'.repeat(64))
    const tx = createTx(`0x22222222${remove0x(data)}`)

    await SequencerBatchAppendedLogHandler(dataService, l, tx)

    dataService.rollupTransactionsInserted.length.should.equal(
      1,
      `No tx batch inserted!`
    )
    dataService.rollupTransactionsInserted[0].length.should.equal(
      3,
      `Tx inserted count mismatch!`
    )
    for (let i = 0; i < dataService.rollupTransactionsInserted[0].length; i++) {
      const received = dataService.rollupTransactionsInserted[0][i]

      received.l1BlockNumber.should.equal(
        tx.blockNumber,
        'Block number mismatch'
      )
      received.l1Timestamp.should.equal(tx.timestamp, 'Timestamp mismatch')
      received.l1TxHash.should.equal(l.transactionHash, 'Tx hash mismatch')
      received.l1TxIndex.should.equal(l.transactionIndex, 'Tx index mismatch')
      received.l1TxLogIndex.should.equal(l.logIndex, 'Tx log index mismatch')
      received.queueOrigin.should.equal(
        QueueOrigin.SEQUENCER,
        'Queue Origin mismatch'
      )
      received.indexWithinSubmission.should.equal(i, 'Batch index mismatch')
      remove0x(received.sender).should.equal(
        remove0x(wallet.address),
        'Sender mismatch'
      )
      remove0x(received.target).should.equal(target, 'Target mismatch')
      received.nonce.should.equal(0, 'Nonce mismatch')
      received.gasLimit.should.equal(1, 'Gas Limit mismatch')
      remove0x(received.signature).should.equal(
        remove0x(signature),
        'Signature mismatch'
      )
      remove0x(received.calldata).should.equal(calldata, 'Calldata mismatch')
    }

    dataService.txHashBatchesCreated.size.should.equal(
      1,
      'Should have created batch!'
    )
  })
  describe('Sequencer Batch as sequencer', () => {
    before(() => {
      process.env.IS_SEQUENCER_STACK = '1'
    })
    after(() => {
      process.env.IS_SEQUENCER_STACK = ''
    })

    it('should parse and insert Sequencer Batch without creating a geth submission', async () => {
      const timestamp = 1

      const target: string = 'bb'.repeat(20)
      const nonce: string = '00'.repeat(32)
      const gasLimit: string = '00'.repeat(31) + '01'
      const calldata: string = 'abcd'.repeat(40)

      const signature = await getTxSignature(target, nonce, gasLimit, calldata)

      let data = `0x${target}${nonce}${gasLimit}${remove0x(
        signature
      )}${calldata}`
      data = abi.encode(['bytes[]', 'uint256'], [[data, data, data], timestamp])

      const l = createLog('00'.repeat(64))
      const tx = createTx(`0x22222222${remove0x(data)}`)

      await SequencerBatchAppendedLogHandler(dataService, l, tx)

      dataService.rollupTransactionsInserted.length.should.equal(
        1,
        `No tx batch inserted!`
      )
      dataService.rollupTransactionsInserted[0].length.should.equal(
        3,
        `Tx inserted count mismatch!`
      )
      for (
        let i = 0;
        i < dataService.rollupTransactionsInserted[0].length;
        i++
      ) {
        const received = dataService.rollupTransactionsInserted[0][i]

        received.l1BlockNumber.should.equal(
          tx.blockNumber,
          'Block number mismatch'
        )
        received.l1Timestamp.should.equal(tx.timestamp, 'Timestamp mismatch')
        received.l1TxHash.should.equal(l.transactionHash, 'Tx hash mismatch')
        received.l1TxIndex.should.equal(l.transactionIndex, 'Tx index mismatch')
        received.l1TxLogIndex.should.equal(l.logIndex, 'Tx log index mismatch')
        received.queueOrigin.should.equal(
          QueueOrigin.SEQUENCER,
          'Queue Origin mismatch'
        )
        received.indexWithinSubmission.should.equal(i, 'Batch index mismatch')
        remove0x(received.sender).should.equal(
          remove0x(wallet.address),
          'Sender mismatch'
        )
        remove0x(received.target).should.equal(target, 'Target mismatch')
        received.nonce.should.equal(0, 'Nonce mismatch')
        received.gasLimit.should.equal(1, 'Gas Limit mismatch')
        remove0x(received.signature).should.equal(
          remove0x(signature),
          'Signature mismatch'
        )
        remove0x(received.calldata).should.equal(calldata, 'Calldata mismatch')
      }

      dataService.txHashBatchesCreated.size.should.equal(
        0,
        'Should not have created batch!'
      )
    })
  })

  it('should parse and insert State Batch', async () => {
    const timestamp = 1

    const roots: string[] = [
      keccak256FromUtf8('1'),
      keccak256FromUtf8('2'),
      keccak256FromUtf8('3'),
    ]
    const data = abi.encode(['bytes32[]'], [roots])

    const l = createLog('00'.repeat(64))
    const tx = createTx(`0x22222222${remove0x(data)}`)

    await StateBatchAppendedLogHandler(dataService, l, tx)

    dataService.txHashToRollupRootsInserted.size.should.equal(
      1,
      `No root batch inserted!`
    )
    dataService.txHashToRollupRootsInserted
      .get(tx.hash)
      .length.should.equal(3, `State root inserted count mismatch!`)
    for (
      let i = 0;
      i < dataService.txHashToRollupRootsInserted.get(tx.hash).length;
      i++
    ) {
      const received = dataService.txHashToRollupRootsInserted.get(tx.hash)[i]
      received.should.equal(roots[i], `Root ${i} mismatch!`)
    }
  })
})
