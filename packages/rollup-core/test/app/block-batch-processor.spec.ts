import '../setup'

/* External Imports */
import { newInMemoryDB } from '@eth-optimism/core-db'
import {
  add0x,
  keccak256,
  sleep,
  TestUtils,
  ZERO_ADDRESS,
  numberToHexString,
} from '@eth-optimism/core-utils'

import { Wallet } from 'ethers'
import {
  BaseProvider,
  Filter,
  Log,
  TransactionResponse,
} from 'ethers/providers'
import { FilterByBlock } from 'ethers/providers/abstract-provider'

/* Internal Imports */
import {
  RollupTransaction,
  BlockBatches,
  BlockBatchListener,
  BatchLogParserContext,
  L1Batch,
} from '../../src/types'
import { CHAIN_ID, BlockBatchProcessor, GAS_LIMIT } from '../../src/app'

class DummyListener implements BlockBatchListener {
  public readonly receivedBlockBatches: BlockBatches[] = []

  public async handleBlockBatches(blockBatch: BlockBatches): Promise<void> {
    this.receivedBlockBatches.push(blockBatch)
  }
}

class MockedProvider extends BaseProvider {
  public logsToReturn: Log[][]
  public transactionsByHash: Map<string, TransactionResponse>

  constructor() {
    super(99)
    this.logsToReturn = []
    this.transactionsByHash = new Map<string, TransactionResponse>()
  }

  public async getLogs(filter: Filter | FilterByBlock): Promise<Log[]> {
    return this.logsToReturn.length > 0 ? this.logsToReturn.pop() : []
  }

  public async getTransaction(txHash: string): Promise<TransactionResponse> {
    return this.transactionsByHash.get(txHash)
  }
}

const getHashFromString = (s: string): string => {
  return add0x(keccak256(Buffer.from(s).toString('hex')))
}

const getLog = (
  topics: string[],
  address: string,
  transactionHash: string = getHashFromString('tx hash'),
  logIndex: number = 1,
  blockNumber: number = 1,
  blockHash: string = getHashFromString('block hash')
): Log => {
  return {
    topics,
    transactionHash,
    address,
    blockNumber,
    blockHash,
    transactionIndex: 1,
    removed: false,
    transactionLogIndex: 1,
    data: '',
    logIndex,
  }
}

const getTransactionResponse = (
  timestamp: number,
  data: string,
  hash: string,
  blockNumber: number = 1,
  blockHash: string = getHashFromString('block hash')
): TransactionResponse => {
  return {
    data,
    timestamp,
    hash,
    blockNumber,
    blockHash,
    confirmations: 1,
    from: ZERO_ADDRESS,
    nonce: 1,
    gasLimit: undefined,
    gasPrice: undefined,
    value: undefined,
    chainId: CHAIN_ID,
    wait: (confirmations) => {
      return undefined
    },
  }
}

const getBlock = (timestamp: number, number: number = 1) => {
  return {
    number,
    hash: getHashFromString('derp derp derp'),
    parentHash: getHashFromString('parent derp'),
    timestamp,
    nonce: '0x01',
    difficulty: 99999,
    gasLimit: undefined,
    gasUsed: undefined,
    miner: '',
    extraData: '',
    transactions: [],
  }
}

const throwOnParsing: L1Batch = [
  {
    nonce: -1,
    sender: ZERO_ADDRESS,
    target: ZERO_ADDRESS,
    calldata: '0xdeadbeef',
    gasLimit:'0x1234'
  },
]

// add calldata as the key and transaction batch that is mock-parsed from the calldata as the values
const calldataParseMap: Map<string, L1Batch> = new Map<string, L1Batch>()
const mapParser = async (l, t) => {
  const res = calldataParseMap.get(t.data)
  if (res === throwOnParsing) {
    throw Error('parsing error')
  }
  return res
}

const batchTxTopic: string = 'abcd'
const batchTxContractAddress: string = '0xabcdef'
const batchTxLogContext: BatchLogParserContext = {
  topic: batchTxTopic,
  contractAddress: batchTxContractAddress,
  parseL1Batch: mapParser,
}

const singleTxTopic: string = '9999'
const singleTxContractAddress: string = '0x999999'
const singleTxLogContext: BatchLogParserContext = {
  topic: singleTxTopic,
  contractAddress: singleTxContractAddress,
  parseL1Batch: mapParser,
}

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

describe('Block Batch Processor', () => {
  let blockBatchProcessor: BlockBatchProcessor
  let db
  let listener: DummyListener
  let mockedLogsProvider: MockedProvider

  beforeEach(async () => {
    db = newInMemoryDB()
    listener = new DummyListener()
    mockedLogsProvider = new MockedProvider()
    blockBatchProcessor = await BlockBatchProcessor.create(
      db,
      mockedLogsProvider,
      [singleTxLogContext, batchTxLogContext],
      [listener]
    )
  })

  describe('positive cases', () => {
    it('should handle empty block properly', async () => {
      await blockBatchProcessor.handle({
        number: 1,
        hash: getHashFromString('derp derp derp'),
        parentHash: getHashFromString('parent derp'),
        timestamp: 123,
        nonce: '0x01',
        difficulty: 99999,
        gasLimit: undefined,
        gasUsed: undefined,
        miner: '',
        extraData: '',
        transactions: [],
      })

      await sleep(100)

      listener.receivedBlockBatches.length.should.equal(
        0,
        'Should not have received a transaction batch'
      )
    })

    it('should handle block with single log properly', async () => {
      const timestamp = 123
      const blockNumber = 0
      const txHash = getHashFromString('derp derp derp')
      calldataParseMap.set(calldata, [rollupTx])

      mockedLogsProvider.transactionsByHash.set(
        txHash,
        getTransactionResponse(timestamp, calldata, txHash)
      )

      mockedLogsProvider.logsToReturn.push([
        getLog([singleTxTopic], singleTxContractAddress, txHash),
      ])

      await blockBatchProcessor.handle(getBlock(timestamp, blockNumber))

      await sleep(100)

      listener.receivedBlockBatches.length.should.equal(
        1,
        'Should have received a block batch'
      )
      listener.receivedBlockBatches[0].timestamp.should.equal(
        timestamp,
        'Timestamp mismatch'
      )
      listener.receivedBlockBatches[0].blockNumber.should.equal(
        blockNumber,
        'Block number mismatch'
      )
      listener.receivedBlockBatches[0].batches.length.should.equal(
        1,
        'Batch count mismatch'
      )
      listener.receivedBlockBatches[0].batches[0].length.should.equal(
        1,
        'transaction count mismatch'
      )
      rollupTxsEqual(
        listener.receivedBlockBatches[0].batches[0][0],
        rollupTx
      ).should.eq(true, 'tx mismatch')
    })

    it('should handle block with multiple logs on same topic as separate batches', async () => {
      const timestamp = 123
      const blockNumber = 0
      const txHash = getHashFromString('derp derp derp')
      calldataParseMap.set(calldata, [rollupTx])

      mockedLogsProvider.transactionsByHash.set(
        txHash,
        getTransactionResponse(timestamp, calldata, txHash)
      )

      const txHash2 = getHashFromString('derp derp derp2')
      calldataParseMap.set(calldata2, [rollupTx2])

      mockedLogsProvider.transactionsByHash.set(
        txHash2,
        getTransactionResponse(timestamp, calldata2, txHash2)
      )

      mockedLogsProvider.logsToReturn.push([
        getLog([singleTxTopic], singleTxContractAddress, txHash, 1),
        getLog([singleTxTopic], singleTxContractAddress, txHash2, 2),
      ])

      await blockBatchProcessor.handle(getBlock(timestamp, blockNumber))

      await sleep(100)

      listener.receivedBlockBatches.length.should.equal(
        1,
        'Should have received a block batch'
      )
      listener.receivedBlockBatches[0].timestamp.should.equal(
        timestamp,
        'Timestamp mismatch'
      )
      listener.receivedBlockBatches[0].blockNumber.should.equal(
        blockNumber,
        'Block number mismatch'
      )
      listener.receivedBlockBatches[0].batches.length.should.equal(
        2,
        'Num batches mismatch'
      )

      listener.receivedBlockBatches[0].batches[0].length.should.equal(
        1,
        'Transaction count mismatch'
      )
      listener.receivedBlockBatches[0].batches[1].length.should.equal(
        1,
        'Transaction count mismatch'
      )

      rollupTxsEqual(
        listener.receivedBlockBatches[0].batches[0][0],
        rollupTx
      ).should.eq(true, 'tx mismatch')
      rollupTxsEqual(
        listener.receivedBlockBatches[0].batches[1][0],
        rollupTx2
      ).should.eq(true, 'tx2 mismatch')
    })

    it('should handle block with multiple logs on different topics as separate batches', async () => {
      const timestamp = 123
      const blockNumber = 0
      const txHash = getHashFromString('derp derp derp')
      calldataParseMap.set(calldata, [rollupTx])

      mockedLogsProvider.transactionsByHash.set(
        txHash,
        getTransactionResponse(timestamp, calldata, txHash)
      )

      const txHash2 = getHashFromString('derp derp derp2')
      calldataParseMap.set(calldata2, [rollupTx2])

      mockedLogsProvider.transactionsByHash.set(
        txHash2,
        getTransactionResponse(timestamp, calldata2, txHash2)
      )

      mockedLogsProvider.logsToReturn.push([
        getLog([singleTxTopic], singleTxContractAddress, txHash, 1),
        getLog([batchTxTopic], batchTxContractAddress, txHash2, 2),
      ])

      await blockBatchProcessor.handle(getBlock(timestamp, blockNumber))

      await sleep(100)

      listener.receivedBlockBatches.length.should.equal(
        1,
        'Should have received a block batche'
      )
      listener.receivedBlockBatches[0].timestamp.should.equal(
        timestamp,
        'Timestamp mismatch'
      )
      listener.receivedBlockBatches[0].blockNumber.should.equal(
        blockNumber,
        'Block number mismatch'
      )
      listener.receivedBlockBatches[0].batches.length.should.equal(
        2,
        'Num batches mismatch'
      )

      listener.receivedBlockBatches[0].batches[0].length.should.equal(
        1,
        'First batch tx count mismatch'
      )
      listener.receivedBlockBatches[0].batches[1].length.should.equal(
        1,
        'Second batch tx count mismatch'
      )

      rollupTxsEqual(
        listener.receivedBlockBatches[0].batches[0][0],
        rollupTx
      ).should.eq(true, 'tx mismatch')
      rollupTxsEqual(
        listener.receivedBlockBatches[0].batches[1][0],
        rollupTx2
      ).should.eq(true, 'tx2 mismatch')
    })

    it('should make different block batches from different blocks', async () => {
      const timestamp = 123
      const blockNumber = 0
      const txHash = getHashFromString('derp derp derp')
      calldataParseMap.set(calldata, [rollupTx])

      mockedLogsProvider.transactionsByHash.set(
        txHash,
        getTransactionResponse(timestamp, calldata, txHash)
      )

      mockedLogsProvider.logsToReturn.push([
        getLog([singleTxTopic], singleTxContractAddress, txHash),
      ])

      await blockBatchProcessor.handle(getBlock(timestamp, blockNumber))

      await sleep(100)

      const timestamp2 = 1234
      const blockNumber2 = 1
      const txHash2 = getHashFromString('derp derp derp 2')
      calldataParseMap.set(calldata2, [rollupTx2])

      mockedLogsProvider.transactionsByHash.set(
        txHash2,
        getTransactionResponse(timestamp2, calldata2, txHash2)
      )

      mockedLogsProvider.logsToReturn.push([
        getLog([singleTxTopic], singleTxContractAddress, txHash2),
      ])

      await blockBatchProcessor.handle(getBlock(timestamp2, blockNumber2))

      await sleep(100)

      listener.receivedBlockBatches.length.should.equal(
        2,
        'Should have received 2 transaction batches'
      )

      listener.receivedBlockBatches[0].timestamp.should.equal(
        timestamp,
        'Timestamp mismatch'
      )
      listener.receivedBlockBatches[0].blockNumber.should.equal(
        blockNumber,
        'Block number mismatch'
      )
      listener.receivedBlockBatches[0].batches.length.should.equal(
        1,
        'Num batches mismatch'
      )
      listener.receivedBlockBatches[0].batches[0].length.should.equal(
        1,
        'Num txs mismatch'
      )

      rollupTxsEqual(
        listener.receivedBlockBatches[0].batches[0][0],
        rollupTx
      ).should.eq(true, 'tx mismatch')

      listener.receivedBlockBatches[1].timestamp.should.equal(
        timestamp2,
        'Timestamp 2 mismatch'
      )
      listener.receivedBlockBatches[1].blockNumber.should.equal(
        blockNumber2,
        'Block number 2 mismatch'
      )
      listener.receivedBlockBatches[1].batches.length.should.equal(
        1,
        'Num batches 2 mismatch'
      )
      listener.receivedBlockBatches[1].batches[0].length.should.equal(
        1,
        'Num transactions 2 mismatch'
      )

      rollupTxsEqual(
        listener.receivedBlockBatches[1].batches[0][0],
        rollupTx2
      ).should.eq(true, 'tx 2 mismatch')
    })
  })

  it('should process batches in order even if blocks are out of order', async () => {
    const timestamp2 = 1234
    const blockNumber2 = 1
    const txHash2 = getHashFromString('derp derp derp 2')
    calldataParseMap.set(calldata2, [rollupTx2])

    mockedLogsProvider.transactionsByHash.set(
      txHash2,
      getTransactionResponse(timestamp2, calldata2, txHash2)
    )

    mockedLogsProvider.logsToReturn.push([
      getLog([singleTxTopic], singleTxContractAddress, txHash2),
    ])

    await blockBatchProcessor.handle(getBlock(timestamp2, blockNumber2))

    await sleep(100)

    const timestamp = 123
    const blockNumber = 0
    const txHash = getHashFromString('derp derp derp')
    calldataParseMap.set(calldata, [rollupTx])

    mockedLogsProvider.transactionsByHash.set(
      txHash,
      getTransactionResponse(timestamp, calldata, txHash)
    )

    mockedLogsProvider.logsToReturn.push([
      getLog([singleTxTopic], singleTxContractAddress, txHash),
    ])

    await blockBatchProcessor.handle(getBlock(timestamp, blockNumber))

    await sleep(100)

    listener.receivedBlockBatches.length.should.equal(
      2,
      'Should have received 2 block batches'
    )

    listener.receivedBlockBatches[0].timestamp.should.equal(
      timestamp,
      'Timestamp mismatch'
    )
    listener.receivedBlockBatches[0].blockNumber.should.equal(
      blockNumber,
      'Block number mismatch'
    )
    listener.receivedBlockBatches[0].batches.length.should.equal(
      1,
      'Num batches mismatch'
    )
    listener.receivedBlockBatches[0].batches[0].length.should.equal(
      1,
      'Num transactions mismatch'
    )

    rollupTxsEqual(
      listener.receivedBlockBatches[0].batches[0][0],
      rollupTx
    ).should.eq(true, 'tx mismatch')

    listener.receivedBlockBatches[1].timestamp.should.equal(
      timestamp2,
      'Timestamp 2 mismatch'
    )
    listener.receivedBlockBatches[1].blockNumber.should.equal(
      blockNumber2,
      'Block number 2 mismatch'
    )
    listener.receivedBlockBatches[1].batches.length.should.equal(
      1,
      'Num batches 2 mismatch'
    )
    listener.receivedBlockBatches[1].batches[0].length.should.equal(
      1,
      'Num transactions 2 mismatch'
    )

    rollupTxsEqual(
      listener.receivedBlockBatches[1].batches[0][0],
      rollupTx2
    ).should.eq(true, 'tx 2 mismatch')
  })

  describe('Negative Cases', () => {
    it('should not produce a block batch from logs to an incorrect topic', async () => {
      const timestamp = 123
      const blockNumber = 0
      const txHash = getHashFromString('derp derp derp')
      calldataParseMap.set(calldata, [rollupTx])

      mockedLogsProvider.transactionsByHash.set(
        txHash,
        getTransactionResponse(timestamp, calldata, txHash)
      )

      mockedLogsProvider.logsToReturn.push([
        getLog(['not the right topic'], singleTxContractAddress, txHash),
      ])

      await blockBatchProcessor.handle(getBlock(timestamp, blockNumber))

      await sleep(100)

      listener.receivedBlockBatches.length.should.equal(
        0,
        'Should not have received a block batch'
      )
    })

    it('should not produce a tx batch from logs from an incorrect contract', async () => {
      const timestamp = 123
      const blockNumber = 0
      const txHash = getHashFromString('derp derp derp')
      calldataParseMap.set(calldata, [rollupTx])

      mockedLogsProvider.transactionsByHash.set(
        txHash,
        getTransactionResponse(timestamp, calldata, txHash)
      )

      mockedLogsProvider.logsToReturn.push([
        getLog([singleTxTopic], ZERO_ADDRESS, txHash),
      ])

      await blockBatchProcessor.handle(getBlock(timestamp, blockNumber))

      await sleep(100)

      listener.receivedBlockBatches.length.should.equal(
        0,
        'Should not have received a transaction batch'
      )
    })

    it('should throw if it cannot correctly parse calldata', async () => {
      const timestamp = 123
      const blockNumber = 0
      const txHash = getHashFromString('derp derp derp')
      calldataParseMap.set(calldata, throwOnParsing)

      mockedLogsProvider.transactionsByHash.set(
        txHash,
        getTransactionResponse(timestamp, calldata, txHash)
      )

      mockedLogsProvider.logsToReturn.push([
        getLog([singleTxTopic], singleTxContractAddress, txHash),
      ])

      await TestUtils.assertThrowsAsync(async () => {
        await blockBatchProcessor.handle(getBlock(timestamp, blockNumber))
      })
    })
  })
})
