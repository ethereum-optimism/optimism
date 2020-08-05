/* External Imports */
import { newInMemoryDB } from '@eth-optimism/core-db'
import {
  BigNumber,
  keccak256FromUtf8,
  sleep,
  ZERO_ADDRESS,
} from '@eth-optimism/core-utils'

import {
  JsonRpcProvider,
  TransactionReceipt,
  TransactionResponse,
} from 'ethers/providers'

/* Internal Imports */
import { L2DataService, TransactionOutput } from '../../src/types'
import {
  CHAIN_ID,
  DefaultDataService,
  L2ChainDataPersister,
} from '../../src/app'

class MockDataService extends DefaultDataService {
  public readonly transactionAndRoots: TransactionOutput[] = []

  constructor() {
    super(undefined)
  }

  public async insertL2TransactionOutput(transaction: TransactionOutput) {
    this.transactionAndRoots.push(transaction)
  }

  public async tryBuildOccBatchToMatchL1Batch(
    batchNumber: number,
    batchSize: number
  ): Promise<number> {
    return undefined
  }

  public async tryBuildCanonicalChainBatchNotPresentOnL1(): Promise<number> {
    return undefined
  }
}

class MockProvider extends JsonRpcProvider {
  public txsToReturn: Map<string, TransactionResponse>
  public txReceiptsToReturn: Map<string, TransactionReceipt>
  constructor() {
    super()
    this.txsToReturn = new Map<string, TransactionResponse>()
    this.txReceiptsToReturn = new Map<string, TransactionReceipt>()
  }

  public async getTransaction(hash): Promise<TransactionResponse> {
    return this.txsToReturn.get(hash)
  }

  public async getTransactionReceipt(
    transactionHash: string
  ): Promise<TransactionReceipt> {
    return this.txReceiptsToReturn.get(transactionHash)
  }
}

const getTransactionResponse = (
  hash: string = keccak256FromUtf8('0xdeadb33f')
): any => {
  return {
    data: '0xdeadb33f',
    timestamp: 0,
    hash,
    blockNumber: 0,
    blockHash: keccak256FromUtf8('block hash'),
    gasLimit: new BigNumber(1_000_000, 10) as any,
    confirmations: 1,
    from: ZERO_ADDRESS,
    nonce: 1,
    gasPrice: undefined,
    value: undefined,
    chainId: CHAIN_ID,
    l1MessageSender: ZERO_ADDRESS,
    l1RollupTxId: 1,
    wait: (confirmations) => {
      return undefined
    },
  }
}

const getTransactionReceipt = (
  transactionHash: string = keccak256FromUtf8('0xdeadb33f'),
  root?: string
): TransactionReceipt => {
  return {
    status: 1,
    root: root || transactionHash,
    transactionHash,
    transactionIndex: 0,
    blockNumber: 0,
    blockHash: keccak256FromUtf8('block hash'),
    byzantium: false,
  }
}

const getBlock = (hash: string, txHashes: string[], number: number = 0) => {
  return {
    number,
    hash,
    parentHash: keccak256FromUtf8('parent derp'),
    timestamp: 1,
    nonce: '0x01',
    difficulty: 99999,
    gasLimit: undefined,
    gasUsed: undefined,
    miner: '',
    extraData: '',
    transactions: txHashes,
  }
}

describe('L2 Chain Data Persister', () => {
  let db
  let chainDataPersister: L2ChainDataPersister
  let dataService: MockDataService
  let provider: MockProvider
  beforeEach(async () => {
    db = newInMemoryDB()
    dataService = new MockDataService()
    provider = new MockProvider()
    chainDataPersister = await L2ChainDataPersister.create(
      db,
      dataService,
      provider
    )
  })

  it('should insert block transaction', async () => {
    const txHash = keccak256FromUtf8('tx hash')
    const block = getBlock(keccak256FromUtf8('derp'), [txHash])

    const txResponse = getTransactionResponse(txHash)
    const txReceipt = getTransactionReceipt(txHash)
    provider.txsToReturn.set(txHash, txResponse)
    provider.txReceiptsToReturn.set(txHash, txReceipt)

    await chainDataPersister.handle(block)

    await sleep(1_000)

    dataService.transactionAndRoots.length.should.equal(
      1,
      `Did not insert tx when should have!`
    )
    const expectedTxAndRoot = L2ChainDataPersister.getTransactionAndRoot(
      block,
      txResponse,
      txReceipt
    )

    dataService.transactionAndRoots[0].should.deep.equal(
      expectedTxAndRoot,
      `Did not insert tx & root!`
    )
  })

  it('should not insert block transaction if empty block', async () => {
    const block = getBlock(keccak256FromUtf8('derp'), [])
    await chainDataPersister.handle(block)

    await sleep(1_000)

    dataService.transactionAndRoots.length.should.equal(
      0,
      `Inserted tx when should not have!`
    )
  })

  it('should insert insert multiple block transactions if present', async () => {
    const txOneHash = keccak256FromUtf8('tx one hash')
    const txTwoHash = keccak256FromUtf8('tx two hash')
    const block = getBlock(keccak256FromUtf8('derp'), [txOneHash, txTwoHash])

    const txOneResponse = getTransactionResponse(txOneHash)
    const txOneReceipt = getTransactionReceipt(txOneHash)
    provider.txsToReturn.set(txOneHash, txOneResponse)
    provider.txReceiptsToReturn.set(txOneHash, txOneReceipt)

    const txTwoResponse = getTransactionResponse(txTwoHash)
    const txTwoReceipt = getTransactionReceipt(txTwoHash)
    provider.txsToReturn.set(txTwoHash, txTwoResponse)
    provider.txReceiptsToReturn.set(txTwoHash, txTwoReceipt)

    await chainDataPersister.handle(block)

    await sleep(1_000)

    dataService.transactionAndRoots.length.should.equal(
      2,
      `Did not insert txs when should have!`
    )
    let expectedTxAndRoot = L2ChainDataPersister.getTransactionAndRoot(
      block,
      txOneResponse,
      txOneReceipt
    )

    dataService.transactionAndRoots[0].should.deep.equal(
      expectedTxAndRoot,
      `Did not insert tx & root 1!`
    )

    expectedTxAndRoot = L2ChainDataPersister.getTransactionAndRoot(
      block,
      txTwoResponse,
      txTwoReceipt
    )

    dataService.transactionAndRoots[1].should.deep.equal(
      expectedTxAndRoot,
      `Did not insert tx & root 2!`
    )
  })
})
