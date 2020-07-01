/* External Imports */
import { newInMemoryDB } from '@eth-optimism/core-db'
import {
  BigNumber,
  keccak256FromUtf8,
  sleep,
  ZERO_ADDRESS,
} from '@eth-optimism/core-utils'

import {
  Block,
  JsonRpcProvider,
  Log,
  TransactionResponse,
} from 'ethers/providers'

/* Internal Imports */
import { L1ChainDataPersister } from '../../src/app/data-handling'
import { L1DataService } from '../../src/types/data'
import { LogHandlerContext, RollupTransaction } from '../../src/types'
import { CHAIN_ID } from '../../src/app'

class MockDataService implements L1DataService {
  public readonly blocks: Block[] = []
  public readonly processedBlocks: Set<string> = new Set<string>()
  public readonly blockTransactions: Map<string, TransactionResponse[]>
  public readonly stateRoots: Map<string, string[]>
  public readonly rollupTransactions: Map<string, RollupTransaction[]>

  constructor() {
    this.blocks = []
    this.processedBlocks = new Set<string>()
    this.blockTransactions = new Map<string, TransactionResponse[]>()
    this.stateRoots = new Map<string, string[]>()
    this.rollupTransactions = new Map<string, RollupTransaction[]>()
  }

  public async insertBlock(block: Block, processed: boolean): Promise<void> {
    this.blocks.push(block)
    if (processed) {
      this.processedBlocks.add(block.hash)
    }
  }

  public async insertBlockAndTransactions(
    block: Block,
    txs: TransactionResponse[],
    processed: boolean
  ): Promise<void> {
    this.blocks.push(block)
    this.blockTransactions.set(block.hash, txs)
    if (processed) {
      this.processedBlocks.add(block.hash)
    }
  }

  public async insertRollupStateRoots(
    l1TxHash: string,
    stateRoots: string[]
  ): Promise<number> {
    this.stateRoots.set(l1TxHash, stateRoots)
    return this.stateRoots.size
  }

  public async insertRollupTransactions(
    l1TxHash: string,
    rollupTransactions: RollupTransaction[]
  ): Promise<number> {
    this.rollupTransactions.set(l1TxHash, rollupTransactions)
    return this.rollupTransactions.size
  }

  public async insertTransactions(
    transactions: TransactionResponse[]
  ): Promise<void> {
    throw Error(`this shouldn't be called`)
  }

  public async updateBlockToProcessed(blockHash: string): Promise<void> {
    this.processedBlocks.add(blockHash)
  }
}

const getLog = (
  topics: string[],
  address: string,
  transactionHash: string = keccak256FromUtf8('tx hash'),
  logIndex: number = 0,
  blockNumber: number = 0,
  blockHash: string = keccak256FromUtf8('block hash')
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
  hash: string = keccak256FromUtf8('0xdeadb33f')
): TransactionResponse => {
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
    wait: (confirmations) => {
      return undefined
    },
  }
}

const getRollupTransaction = (): RollupTransaction => {
  return {
    batchIndex: -1,
    target: ZERO_ADDRESS,
    calldata: '0xdeadbeef',
    l1MessageSender: ZERO_ADDRESS,
    l1Timestamp: 0,
    l1BlockNumber: 0,
    l1TxHash: keccak256FromUtf8('0xdeadbeef'),
    nonce: 0,
    queueOrigin: 0,
  }
}

const getBlock = (hash: string, number: number = 0, timestamp: number = 1) => {
  return {
    number,
    hash,
    parentHash: keccak256FromUtf8('parent derp'),
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

class MockProvider extends JsonRpcProvider {
  public logsToReturn: Log[]
  public txsToReturn: Map<string, TransactionResponse>
  constructor() {
    super()
    this.logsToReturn = []
    this.txsToReturn = new Map<string, TransactionResponse>()
  }

  public async getLogs(filter): Promise<Log[]> {
    return this.logsToReturn
  }

  public async getTransaction(hash): Promise<TransactionResponse> {
    return this.txsToReturn.get(hash)
  }
}

const topic = 'derp'
const contractAddress = '0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef'
const defaultBlock = getBlock(keccak256FromUtf8('derp'))

const errorLogHandlerContext: LogHandlerContext = {
  topic,
  contractAddress,
  handleLog: async () => {
    throw Error('This should not have been called')
  },
}

describe('L1 Chain Data Persister', () => {
  let db
  let chainDataPersister: L1ChainDataPersister
  let dataService: MockDataService
  let provider: MockProvider
  beforeEach(async () => {
    db = newInMemoryDB()
    dataService = new MockDataService()
    provider = new MockProvider()
  })

  it('should not persist block without logs', async () => {
    chainDataPersister = await L1ChainDataPersister.create(
      db,
      dataService,
      provider,
      []
    )

    const block = getBlock(keccak256FromUtf8('derp'))
    await chainDataPersister.handle(block)

    await sleep(1_000)

    dataService.blocks.length.should.equal(
      0,
      `Inserted blocks when shouldn't have!`
    )
    dataService.blockTransactions.size.should.equal(
      0,
      `Inserted transactions when shouldn't have!`
    )
    dataService.stateRoots.size.should.equal(
      0,
      `Inserted roots when shouldn't have!`
    )
  })

  describe('Irrelevant logs', () => {
    it('should not persist block without log handler', async () => {
      chainDataPersister = await L1ChainDataPersister.create(
        db,
        dataService,
        provider,
        []
      )

      provider.logsToReturn.push(getLog(['derp'], ZERO_ADDRESS))

      const block = getBlock(keccak256FromUtf8('derp'))
      await chainDataPersister.handle(block)

      await sleep(1_000)

      dataService.blocks.length.should.equal(
        0,
        `Inserted blocks when shouldn't have!`
      )
      dataService.blockTransactions.size.should.equal(
        0,
        `Inserted transactions when shouldn't have!`
      )
      dataService.stateRoots.size.should.equal(
        0,
        `Inserted roots when shouldn't have!`
      )
    })

    it('should not persist block without logs relevant to log handler topic', async () => {
      const logHandlerContext: LogHandlerContext = {
        topic: 'not your topic',
        contractAddress: ZERO_ADDRESS,
        handleLog: async () => {
          throw Error('This should not have been called')
        },
      }
      chainDataPersister = await L1ChainDataPersister.create(
        db,
        dataService,
        provider,
        [logHandlerContext]
      )

      provider.logsToReturn.push(getLog(['derp'], ZERO_ADDRESS))

      const block = getBlock(keccak256FromUtf8('derp'))
      await chainDataPersister.handle(block)

      await sleep(1_000)

      dataService.blocks.length.should.equal(
        0,
        `Inserted blocks when shouldn't have!`
      )
      dataService.blockTransactions.size.should.equal(
        0,
        `Inserted transactions when shouldn't have!`
      )
      dataService.stateRoots.size.should.equal(
        0,
        `Inserted roots when shouldn't have!`
      )
    })

    it('should not persist block without logs relevant to log handler address', async () => {
      chainDataPersister = await L1ChainDataPersister.create(
        db,
        dataService,
        provider,
        [errorLogHandlerContext]
      )

      provider.logsToReturn.push(getLog([topic], ZERO_ADDRESS))

      await chainDataPersister.handle(defaultBlock)

      await sleep(1_000)

      dataService.blocks.length.should.equal(
        0,
        `Inserted blocks when shouldn't have!`
      )
      dataService.blockTransactions.size.should.equal(
        0,
        `Inserted transactions when shouldn't have!`
      )
      dataService.stateRoots.size.should.equal(
        0,
        `Inserted roots when shouldn't have!`
      )
    })
  })

  describe('relevant logs', () => {
    const configuredHandlerContext: LogHandlerContext = {
      ...errorLogHandlerContext,
    }
    beforeEach(async () => {
      chainDataPersister = await L1ChainDataPersister.create(
        db,
        dataService,
        provider,
        [configuredHandlerContext]
      )
    })

    it('should persist block, transaction, and rollup transactions with relevant logs', async () => {
      const rollupTxs = [getRollupTransaction()]
      configuredHandlerContext.handleLog = async (ds, l, t) => {
        await ds.insertRollupTransactions(t.hash, rollupTxs)
      }

      const tx: TransactionResponse = getTransactionResponse()
      provider.txsToReturn.set(tx.hash, tx)
      provider.logsToReturn.push(getLog([topic], contractAddress, tx.hash))

      await chainDataPersister.handle(defaultBlock)

      await sleep(1_000)

      dataService.blocks.length.should.equal(1, `Should have inserted block!`)
      dataService.blocks[0].should.deep.equal(defaultBlock, `block mismatch!`)

      dataService.blockTransactions.size.should.equal(
        1,
        `Should have inserted transaction!`
      )
      const blockTxsExist: boolean = !!dataService.blockTransactions.get(
        defaultBlock.hash
      )
      blockTxsExist.should.equal(
        true,
        `Should have inserted txs for the block!`
      )
      dataService.blockTransactions
        .get(defaultBlock.hash)
        .length.should.equal(1, `Should have inserted 1 block transaction!`)
      dataService.blockTransactions
        .get(defaultBlock.hash)[0]
        .should.deep.equal(tx, `Should have inserted block transactions!`)

      const rollupTxsExist: boolean = !!dataService.rollupTransactions.get(
        tx.hash
      )
      rollupTxsExist.should.equal(
        true,
        `Should have inserted rollup txs for the tx!`
      )
      dataService.rollupTransactions
        .get(tx.hash)
        .length.should.equal(1, `Should have inserted 1 rollup tx!`)
      dataService.rollupTransactions
        .get(tx.hash)[0]
        .should.deep.equal(rollupTxs[0], `Inserted rollup tx mismatch!`)

      dataService.processedBlocks.size.should.equal(1, `block not processed!`)
      dataService.processedBlocks
        .has(defaultBlock.hash)
        .should.equal(true, `correct block not processed!`)
    })

    it('should persist block, transaction, and state roots with relevant logs', async () => {
      const stateRoots = [keccak256FromUtf8('root')]
      configuredHandlerContext.handleLog = async (ds, l, t) => {
        await ds.insertRollupStateRoots(t.hash, stateRoots)
      }

      const tx: TransactionResponse = getTransactionResponse()
      provider.txsToReturn.set(tx.hash, tx)
      provider.logsToReturn.push(getLog([topic], contractAddress, tx.hash))

      await chainDataPersister.handle(defaultBlock)

      await sleep(1_000)

      dataService.blocks.length.should.equal(1, `Should have inserted block!`)
      dataService.blocks[0].should.deep.equal(defaultBlock, `block mismatch!`)

      dataService.blockTransactions.size.should.equal(
        1,
        `Should have inserted transaction!`
      )
      const blockTxsExist: boolean = !!dataService.blockTransactions.get(
        defaultBlock.hash
      )
      blockTxsExist.should.equal(
        true,
        `Should have inserted txs for the block!`
      )
      dataService.blockTransactions
        .get(defaultBlock.hash)
        .length.should.equal(1, `Should have inserted 1 block transaction!`)
      dataService.blockTransactions
        .get(defaultBlock.hash)[0]
        .should.deep.equal(tx, `Should have inserted block transactions!`)

      const stateRootsExist: boolean = !!dataService.stateRoots.get(tx.hash)
      stateRootsExist.should.equal(
        true,
        `Should have inserted state roots for the tx!`
      )
      dataService.stateRoots
        .get(tx.hash)
        .length.should.equal(1, `Should have inserted 1 state root!`)
      dataService.stateRoots
        .get(tx.hash)[0]
        .should.deep.equal(stateRoots[0], `Inserted state Root mismatch!`)

      dataService.processedBlocks.size.should.equal(1, `block not processed!`)
      dataService.processedBlocks
        .has(defaultBlock.hash)
        .should.equal(true, `correct block not processed!`)
    })

    it('should persist block, transaction, rollup transactions, and state roots with relevant logs -- single tx', async () => {
      const rollupTxs = [getRollupTransaction()]
      const stateRoots = [keccak256FromUtf8('root')]
      configuredHandlerContext.handleLog = async (ds, l, t) => {
        await ds.insertRollupStateRoots(t.hash, stateRoots)
      }
      const topic2 = 'derp_derp'
      chainDataPersister = await L1ChainDataPersister.create(
        db,
        dataService,
        provider,
        [
          configuredHandlerContext,
          {
            topic: topic2,
            contractAddress,
            handleLog: async (ds, l, t) => {
              await ds.insertRollupTransactions(t.hash, rollupTxs)
            },
          },
        ]
      )

      const tx: TransactionResponse = getTransactionResponse()
      provider.txsToReturn.set(tx.hash, tx)
      provider.logsToReturn.push(
        getLog([topic, topic2], contractAddress, tx.hash)
      )

      await chainDataPersister.handle(defaultBlock)

      await sleep(1_000)

      dataService.blocks.length.should.equal(1, `Should have inserted block!`)
      dataService.blocks[0].should.deep.equal(defaultBlock, `block mismatch!`)

      dataService.blockTransactions.size.should.equal(
        1,
        `Should have inserted transaction!`
      )
      const blockTxsExist: boolean = !!dataService.blockTransactions.get(
        defaultBlock.hash
      )
      blockTxsExist.should.equal(
        true,
        `Should have inserted txs for the block!`
      )
      dataService.blockTransactions
        .get(defaultBlock.hash)
        .length.should.equal(1, `Should have inserted 1 block transaction!`)
      dataService.blockTransactions
        .get(defaultBlock.hash)[0]
        .should.deep.equal(tx, `Should have inserted block transactions!`)

      const stateRootsExist: boolean = !!dataService.stateRoots.get(tx.hash)
      stateRootsExist.should.equal(
        true,
        `Should have inserted state roots for the tx!`
      )
      dataService.stateRoots
        .get(tx.hash)
        .length.should.equal(1, `Should have inserted 1 state root!`)
      dataService.stateRoots
        .get(tx.hash)[0]
        .should.deep.equal(stateRoots[0], `Inserted state Root mismatch!`)

      dataService.processedBlocks.size.should.equal(1, `block not processed!`)
      dataService.processedBlocks
        .has(defaultBlock.hash)
        .should.equal(true, `correct block not processed!`)

      const rollupTxsExist: boolean = !!dataService.rollupTransactions.get(
        tx.hash
      )
      rollupTxsExist.should.equal(
        true,
        `Should have inserted rollup txs for the tx!`
      )
      dataService.rollupTransactions
        .get(tx.hash)
        .length.should.equal(1, `Should have inserted 1 rollup tx!`)
      dataService.rollupTransactions
        .get(tx.hash)[0]
        .should.deep.equal(rollupTxs[0], `Inserted rollup tx mismatch!`)
    })

    it('should persist block, transaction, rollup transactions, and state roots with relevant logs -- separate txs', async () => {
      const rollupTxs = [getRollupTransaction()]
      const stateRoots = [keccak256FromUtf8('root')]
      configuredHandlerContext.handleLog = async (ds, l, t) => {
        await ds.insertRollupStateRoots(tx.hash, stateRoots)
      }
      const topic2 = 'derp_derp'
      chainDataPersister = await L1ChainDataPersister.create(
        db,
        dataService,
        provider,
        [
          configuredHandlerContext,
          {
            topic: topic2,
            contractAddress,
            handleLog: async (ds, l, t) => {
              await ds.insertRollupTransactions(t.hash, rollupTxs)
            },
          },
        ]
      )

      const tx: TransactionResponse = getTransactionResponse()
      const tx2: TransactionResponse = getTransactionResponse(
        keccak256FromUtf8('tx2')
      )
      provider.txsToReturn.set(tx.hash, tx)
      provider.txsToReturn.set(tx2.hash, tx2)
      provider.logsToReturn.push(
        getLog([topic], contractAddress, tx.hash),
        getLog([topic2], contractAddress, tx2.hash)
      )

      await chainDataPersister.handle(defaultBlock)

      await sleep(1_000)

      dataService.blocks.length.should.equal(1, `Should have inserted block!`)
      dataService.blocks[0].should.deep.equal(defaultBlock, `block mismatch!`)

      dataService.blockTransactions.size.should.equal(
        1,
        `Should have inserted transactions for 1 block!`
      )
      const blockTxsExist: boolean = !!dataService.blockTransactions.get(
        defaultBlock.hash
      )
      blockTxsExist.should.equal(
        true,
        `Should have inserted txs for the block!`
      )
      dataService.blockTransactions
        .get(defaultBlock.hash)
        .length.should.equal(2, `Should have inserted 2 block transactions!`)
      dataService.blockTransactions
        .get(defaultBlock.hash)[0]
        .should.deep.equal(tx, `Should have inserted block transaction 1!`)
      dataService.blockTransactions
        .get(defaultBlock.hash)[1]
        .should.deep.equal(tx2, `Should have inserted block transaction 2!`)

      const stateRootsExist: boolean = !!dataService.stateRoots.get(tx.hash)
      stateRootsExist.should.equal(
        true,
        `Should have inserted state roots for the tx!`
      )
      dataService.stateRoots
        .get(tx.hash)
        .length.should.equal(1, `Should have inserted 1 state root!`)
      dataService.stateRoots
        .get(tx.hash)[0]
        .should.deep.equal(stateRoots[0], `Inserted state Root mismatch!`)

      dataService.processedBlocks.size.should.equal(1, `block not processed!`)
      dataService.processedBlocks
        .has(defaultBlock.hash)
        .should.equal(true, `correct block not processed!`)

      const rollupTxsExist: boolean = !!dataService.rollupTransactions.get(
        tx2.hash
      )
      rollupTxsExist.should.equal(
        true,
        `Should have inserted rollup txs for the tx!`
      )
      dataService.rollupTransactions
        .get(tx2.hash)
        .length.should.equal(1, `Should have inserted 1 rollup tx!`)
      dataService.rollupTransactions
        .get(tx2.hash)[0]
        .should.deep.equal(rollupTxs[0], `Inserted rollup tx mismatch!`)
    })

    describe('multiple blocks', () => {
      it('should only persist relevant block, transaction, and rollup transactions with relevant logs', async () => {
        const rollupTxs = [getRollupTransaction()]
        configuredHandlerContext.handleLog = async (ds, l, t) => {
          await ds.insertRollupTransactions(tx.hash, rollupTxs)
        }

        const tx: TransactionResponse = getTransactionResponse()
        provider.txsToReturn.set(tx.hash, tx)

        const blockOne = getBlock(keccak256FromUtf8('first'))

        await chainDataPersister.handle(blockOne)

        await sleep(1_000)

        provider.logsToReturn.push(getLog([topic], contractAddress, tx.hash))

        const blockTwo = { ...defaultBlock }
        blockTwo.number = 1
        await chainDataPersister.handle(blockTwo)

        await sleep(1_000)

        dataService.blocks.length.should.equal(1, `Should have inserted block!`)
        dataService.blocks[0].should.deep.equal(blockTwo, `block mismatch!`)

        dataService.blockTransactions.size.should.equal(
          1,
          `Should have transactions for a single block!`
        )
        const blockOneTxsExist: boolean = !!dataService.blockTransactions.get(
          blockOne.hash
        )
        blockOneTxsExist.should.equal(
          false,
          `Should not have inserted txs for blockOne!`
        )

        const blockTwoTxsExist: boolean = !!dataService.blockTransactions.get(
          blockTwo.hash
        )
        blockTwoTxsExist.should.equal(
          true,
          `Should have inserted txs for blockTwo!`
        )

        dataService.blockTransactions
          .get(blockTwo.hash)
          .length.should.equal(1, `Should have inserted 1 block transaction!`)
        dataService.blockTransactions
          .get(blockTwo.hash)[0]
          .should.deep.equal(tx, `Should have inserted block transactions!`)

        const rollupTxsExist: boolean = !!dataService.rollupTransactions.get(
          tx.hash
        )
        rollupTxsExist.should.equal(
          true,
          `Should have inserted rollup txs for the tx!`
        )
        dataService.rollupTransactions
          .get(tx.hash)
          .length.should.equal(1, `Should have inserted 1 rollup tx!`)
        dataService.rollupTransactions
          .get(tx.hash)[0]
          .should.deep.equal(rollupTxs[0], `Inserted rollup tx mismatch!`)

        dataService.processedBlocks.size.should.equal(1, `block not processed!`)
        dataService.processedBlocks
          .has(blockTwo.hash)
          .should.equal(true, `correct block not processed!`)
      })
    })
  })
})
