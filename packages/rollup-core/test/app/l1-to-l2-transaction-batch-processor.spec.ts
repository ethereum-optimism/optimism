import '../setup'

/* External Imports */
import { newInMemoryDB } from '@eth-optimism/core-db'
import { add0x, keccak256, sleep, ZERO_ADDRESS } from '@eth-optimism/core-utils'
import * as BigNumber from 'bn.js'

/* Internal Imports */
import {
  L1ToL2Transaction,
  L1ToL2TransactionBatch,
  L1ToL2TransactionBatchListener,
  L1ToL2TransactionLogParserContext,
} from '../../src/types'
import { CHAIN_ID, L1TransactionBatchProcessor } from '../../src/app'
import { Wallet } from 'ethers'
import {
  BaseProvider,
  Filter,
  JsonRpcProvider,
  Log,
  TransactionResponse,
} from 'ethers/providers'
import { FilterByBlock } from 'ethers/providers/abstract-provider'

class DummyListener implements L1ToL2TransactionBatchListener {
  public readonly receivedTransactionBatches: L1ToL2TransactionBatch[] = []

  public async handleL1ToL2TransactionBatch(
    transactionBatch: L1ToL2TransactionBatch
  ): Promise<void> {
    this.receivedTransactionBatches.push(transactionBatch)
  }
}

class MockedProvider extends BaseProvider {
  public logsToReturn: Log[]
  public transactionsByHash: Map<string, TransactionResponse>

  constructor() {
    super(99)
    this.logsToReturn = []
    this.transactionsByHash = new Map<string, TransactionResponse>()
  }

  public async getLogs(filter: Filter | FilterByBlock): Promise<Log[]> {
    return this.logsToReturn
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
  blockHash: string = getHashFromString('block hash'),
  transactionIndex: number = 1
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

// add calldata as the key and transactions that are mock-parsed from the calldata as the values
const calldataParseMap: Map<string, L1ToL2Transaction[]> = new Map<
  string,
  L1ToL2Transaction[]
>()
const mapParser = async (l, t) => calldataParseMap.get(t.data)

const batchTxTopic: string = 'abcd'
const batchTxContractAddress: string = '0xabcdef'
const batchTxLogContext: L1ToL2TransactionLogParserContext = {
  topic: batchTxTopic,
  contractAddress: batchTxContractAddress,
  parseL2Transactions: mapParser,
}

const singleTxTopic: string = '9999'
const singleTxContractAddress: string = '0x999999'
const singleTxLogContext: L1ToL2TransactionLogParserContext = {
  topic: singleTxTopic,
  contractAddress: singleTxContractAddress,
  parseL2Transactions: mapParser,
}

const nonce: number = 0
const sender: string = Wallet.createRandom().address
const target: string = Wallet.createRandom().address
const calldata: string = keccak256(Buffer.from('calldata').toString('hex'))
const l1ToL2Tx: L1ToL2Transaction = {
  nonce,
  sender,
  target,
  calldata,
}

const nonce2: number = 2
const sender2: string = Wallet.createRandom().address
const target2: string = Wallet.createRandom().address
const calldata2: string = keccak256(Buffer.from('calldata 2').toString('hex'))
const l1ToL2Tx2: L1ToL2Transaction = {
  nonce: nonce2,
  sender: sender2,
  target: target2,
  calldata: calldata2,
}

describe('L1 to L2 Transaction Batch Processor', () => {
  let l1ToL2TransactionBatchProcessor: L1TransactionBatchProcessor
  let db
  let listener: DummyListener
  let mockedLogsProvider: MockedProvider

  beforeEach(async () => {
    db = newInMemoryDB()
    listener = new DummyListener()
    mockedLogsProvider = new MockedProvider()
    l1ToL2TransactionBatchProcessor = await L1TransactionBatchProcessor.create(
      db,
      mockedLogsProvider,
      [singleTxLogContext, batchTxLogContext],
      [listener]
    )
  })

  it('should handle empty block properly', async () => {
    await l1ToL2TransactionBatchProcessor.handle({
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

    await sleep(1_000)

    listener.receivedTransactionBatches.length.should.equal(
      0,
      'Should not have received a transaction batch'
    )
  })

  it('should handle block with single log properly', async () => {
    const timestamp = 123
    const blockNumber = 0
    const txHash = getHashFromString('derp derp derp')
    calldataParseMap.set(calldata, [l1ToL2Tx])

    mockedLogsProvider.transactionsByHash.set(
      txHash,
      getTransactionResponse(timestamp, calldata, txHash)
    )

    mockedLogsProvider.logsToReturn.push(
      getLog([singleTxTopic], singleTxContractAddress, txHash)
    )

    await l1ToL2TransactionBatchProcessor.handle(
      getBlock(timestamp, blockNumber)
    )

    await sleep(1_000)

    listener.receivedTransactionBatches.length.should.equal(
      1,
      'Should have received a transaction batch'
    )
    listener.receivedTransactionBatches[0].timestamp.should.equal(
      timestamp,
      'Timestamp mismatch'
    )
    listener.receivedTransactionBatches[0].blockNumber.should.equal(
      blockNumber,
      'Block number mismatch'
    )
    listener.receivedTransactionBatches[0].transactions.length.should.equal(
      1,
      'Num transactions mismatch'
    )
    listener.receivedTransactionBatches[0].transactions[0].should.eq(
      l1ToL2Tx,
      'tx mismatch'
    )
  })

  it('should handle block with multiple logs on same topic properly', async () => {
    const timestamp = 123
    const blockNumber = 0
    const txHash = getHashFromString('derp derp derp')
    calldataParseMap.set(calldata, [l1ToL2Tx])

    mockedLogsProvider.transactionsByHash.set(
      txHash,
      getTransactionResponse(timestamp, calldata, txHash)
    )

    mockedLogsProvider.logsToReturn.push(
      getLog([singleTxTopic], singleTxContractAddress, txHash, 1)
    )

    const txHash2 = getHashFromString('derp derp derp2')
    calldataParseMap.set(calldata2, [l1ToL2Tx2])

    mockedLogsProvider.transactionsByHash.set(
      txHash2,
      getTransactionResponse(timestamp, calldata2, txHash2)
    )

    mockedLogsProvider.logsToReturn.push(
      getLog([singleTxTopic], singleTxContractAddress, txHash2, 2)
    )

    await l1ToL2TransactionBatchProcessor.handle(
      getBlock(timestamp, blockNumber)
    )

    await sleep(1_000)

    listener.receivedTransactionBatches.length.should.equal(
      1,
      'Should have received a transaction batch'
    )
    listener.receivedTransactionBatches[0].timestamp.should.equal(
      timestamp,
      'Timestamp mismatch'
    )
    listener.receivedTransactionBatches[0].blockNumber.should.equal(
      blockNumber,
      'Block number mismatch'
    )
    listener.receivedTransactionBatches[0].transactions.length.should.equal(
      2,
      'Num transactions mismatch'
    )

    listener.receivedTransactionBatches[0].transactions[0].should.eq(
      l1ToL2Tx,
      'tx mismatch'
    )
    listener.receivedTransactionBatches[0].transactions[1].should.eq(
      l1ToL2Tx2,
      'tx2 mismatch'
    )
  })

  it('should handle block with multiple logs on differnt topics properly', async () => {
    const timestamp = 123
    const blockNumber = 0
    const txHash = getHashFromString('derp derp derp')
    calldataParseMap.set(calldata, [l1ToL2Tx])

    mockedLogsProvider.transactionsByHash.set(
      txHash,
      getTransactionResponse(timestamp, calldata, txHash)
    )

    mockedLogsProvider.logsToReturn.push(
      getLog([singleTxTopic], singleTxContractAddress, txHash, 1)
    )

    const txHash2 = getHashFromString('derp derp derp2')
    calldataParseMap.set(calldata2, [l1ToL2Tx2])

    mockedLogsProvider.transactionsByHash.set(
      txHash2,
      getTransactionResponse(timestamp, calldata2, txHash2)
    )

    mockedLogsProvider.logsToReturn.push(
      getLog([batchTxTopic], batchTxContractAddress, txHash2, 2)
    )

    await l1ToL2TransactionBatchProcessor.handle(
      getBlock(timestamp, blockNumber)
    )

    await sleep(1_000)

    listener.receivedTransactionBatches.length.should.equal(
      1,
      'Should have received a transaction batch'
    )
    listener.receivedTransactionBatches[0].timestamp.should.equal(
      timestamp,
      'Timestamp mismatch'
    )
    listener.receivedTransactionBatches[0].blockNumber.should.equal(
      blockNumber,
      'Block number mismatch'
    )
    listener.receivedTransactionBatches[0].transactions.length.should.equal(
      2,
      'Num transactions mismatch'
    )

    listener.receivedTransactionBatches[0].transactions[0].should.eq(
      l1ToL2Tx,
      'tx mismatch'
    )
    listener.receivedTransactionBatches[0].transactions[1].should.eq(
      l1ToL2Tx2,
      'tx2 mismatch'
    )
  })
})
