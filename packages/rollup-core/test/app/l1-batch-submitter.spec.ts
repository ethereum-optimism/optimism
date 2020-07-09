/* External Imports */
import {
  keccak256FromUtf8,
  sleep,
  ZERO_ADDRESS,
} from '@eth-optimism/core-utils'
import { TransactionReceipt, TransactionResponse } from 'ethers/providers'
import { Wallet } from 'ethers'

/* Internal Imports */
import { DefaultDataService } from '../../src/app/data'
import { L1BatchSubmission, L2BatchStatus } from '../../src/types/data'
import { L1BatchSubmitter } from '../../src/app/data/consumers/l1-batch-submitter'

interface BatchNumberHash {
  batchNumber: number
  txHash: string
}

class MockDataService extends DefaultDataService {
  public readonly nextBatch: L1BatchSubmission[] = []
  public readonly txBatchesSubmitted: BatchNumberHash[] = []
  public readonly txBatchesConfirmed: BatchNumberHash[] = []
  public readonly stateBatchesSubmitted: BatchNumberHash[] = []
  public readonly stateBatchesConfirmed: BatchNumberHash[] = []

  constructor() {
    super(undefined)
  }

  public async getNextBatchForL1Submission(): Promise<L1BatchSubmission> {
    return this.nextBatch.shift()
  }

  public async markTransactionBatchSubmittedToL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    this.txBatchesSubmitted.push({ batchNumber, txHash: l1TxHash })
  }

  public async markTransactionBatchConfirmedOnL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    this.txBatchesConfirmed.push({ batchNumber, txHash: l1TxHash })
  }

  public async markStateRootBatchSubmittedToL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    this.stateBatchesSubmitted.push({ batchNumber, txHash: l1TxHash })
  }

  public async markStateRootBatchConfirmedOnL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    this.stateBatchesConfirmed.push({ batchNumber, txHash: l1TxHash })
  }
}

class MockProvider {
  public confirmedTxs: Map<string, TransactionReceipt> = new Map<
    string,
    TransactionReceipt
  >()

  public async waitForTransaction(
    hash: string,
    numConfirms: number
  ): Promise<TransactionReceipt> {
    while (!this.confirmedTxs.get(hash)) {
      await sleep(100)
    }
    return this.confirmedTxs.get(hash)
  }
}

class MockCanonicalTransactionChain {
  public responses: TransactionResponse[] = []

  constructor(public readonly provider: MockProvider) {}

  public async appendSequencerBatch(
    calldata: string,
    timestamp: number
  ): Promise<TransactionResponse> {
    const response: TransactionResponse = this.responses.shift()
    if (!response) {
      throw Error('no response')
    }
    return response
  }
}

class MockStatCommitmentChain {
  public responses: TransactionResponse[] = []

  constructor(public readonly provider: MockProvider) {}

  public async appendStateBatch(
    batches: string[]
  ): Promise<TransactionResponse> {
    const response: TransactionResponse = this.responses.shift()
    if (!response) {
      throw Error('no response')
    }
    return response
  }
}

describe.only('L1 Batch Submitter', () => {
  let batchSubmitter: L1BatchSubmitter
  let dataService: MockDataService
  let canonicalProvider: MockProvider
  let canonicalTransactionChain: MockCanonicalTransactionChain
  let stateCommitmentProvider: MockProvider
  let stateCommitmentChain: MockStatCommitmentChain

  beforeEach(async () => {
    dataService = new MockDataService()
    canonicalProvider = new MockProvider()
    canonicalTransactionChain = new MockCanonicalTransactionChain(
      canonicalProvider
    )
    stateCommitmentProvider = new MockProvider()
    stateCommitmentChain = new MockStatCommitmentChain(stateCommitmentProvider)
    batchSubmitter = new L1BatchSubmitter(
      dataService,
      canonicalTransactionChain as any,
      stateCommitmentChain as any
    )
  })

  it('should not do anything if there are no batches', async () => {
    await batchSubmitter.runTask()

    dataService.txBatchesSubmitted.length.should.equal(
      0,
      'No tx batches should have been submitted!'
    )
    dataService.txBatchesConfirmed.length.should.equal(
      0,
      'No tx batches should have been confirmed!'
    )

    dataService.stateBatchesSubmitted.length.should.equal(
      0,
      'No state batches should have been submitted!'
    )
    dataService.stateBatchesConfirmed.length.should.equal(
      0,
      'No state batches should have been confirmed!'
    )
  })

  it('should not do anything if the next batch has an invalid status', async () => {
    dataService.nextBatch.push({
      l1TxBatchTxHash: undefined,
      l1StateRootBatchTxHash: undefined,
      status: L2BatchStatus.UNBATCHED,
      l2BatchNumber: 1,
      transactions: [
        {
          timestamp: 1,
          blockNumber: 2,
          transactionHash: keccak256FromUtf8('l2 tx hash'),
          transactionIndex: 0,
          to: Wallet.createRandom().address,
          from: Wallet.createRandom().address,
          nonce: 1,
          calldata: keccak256FromUtf8('some calldata'),
          stateRoot: keccak256FromUtf8('l2 state root'),
          signature: 'ab'.repeat(65),
        },
      ],
    })

    await batchSubmitter.runTask()

    dataService.txBatchesSubmitted.length.should.equal(
      0,
      'No tx batches should have been submitted!'
    )
    dataService.txBatchesConfirmed.length.should.equal(
      0,
      'No tx batches should have been confirmed!'
    )

    dataService.stateBatchesSubmitted.length.should.equal(
      0,
      'No state batches should have been submitted!'
    )
    dataService.stateBatchesConfirmed.length.should.equal(
      0,
      'No state batches should have been confirmed!'
    )
  })

  it('should send txs and roots if there is a batch', async () => {
    const hash: string = keccak256FromUtf8('l1 tx hash')
    const batchNumber: number = 1
    dataService.nextBatch.push({
      l1TxBatchTxHash: undefined,
      l1StateRootBatchTxHash: undefined,
      status: L2BatchStatus.BATCHED,
      l2BatchNumber: batchNumber,
      transactions: [
        {
          timestamp: 1,
          blockNumber: 2,
          transactionHash: keccak256FromUtf8('l2 tx hash'),
          transactionIndex: 0,
          to: Wallet.createRandom().address,
          from: Wallet.createRandom().address,
          nonce: 1,
          calldata: keccak256FromUtf8('some calldata'),
          stateRoot: keccak256FromUtf8('l2 state root'),
          signature: 'ab'.repeat(65),
        },
      ],
    })

    canonicalTransactionChain.responses.push({ hash } as any)
    stateCommitmentChain.responses.push({ hash } as any)

    await batchSubmitter.runTask()

    dataService.txBatchesSubmitted.length.should.equal(
      1,
      'No tx batches submitted!'
    )
    dataService.txBatchesSubmitted[0].txHash.should.equal(
      hash,
      'Incorrect tx hash submitted!'
    )
    dataService.txBatchesSubmitted[0].batchNumber.should.equal(
      batchNumber,
      'Incorrect tx batch number submitted!'
    )

    dataService.txBatchesConfirmed.length.should.equal(
      1,
      'No tx batches confirmed!'
    )
    dataService.txBatchesConfirmed[0].txHash.should.equal(
      hash,
      'Incorrect tx hash confirmed!'
    )
    dataService.txBatchesConfirmed[0].batchNumber.should.equal(
      batchNumber,
      'Incorrect tx batch number confirmed!'
    )

    dataService.stateBatchesSubmitted.length.should.equal(
      1,
      'No state batches submitted!'
    )
    dataService.stateBatchesSubmitted[0].txHash.should.equal(
      hash,
      'Incorrect tx hash state root confirmed!'
    )
    dataService.stateBatchesSubmitted[0].batchNumber.should.equal(
      batchNumber,
      'Incorrect tx batch number state root confirmed!'
    )

    dataService.stateBatchesConfirmed.length.should.equal(
      1,
      'No state batches confirmed!'
    )
    dataService.stateBatchesConfirmed[0].txHash.should.equal(
      hash,
      'Incorrect tx hash state root confirmed!'
    )
    dataService.stateBatchesConfirmed[0].batchNumber.should.equal(
      batchNumber,
      'Incorrect tx batch number state root confirmed!'
    )
  })

  it('should wait for tx confirmation and send roots if there is a batch in TXS_SUBMITTED status', async () => {
    const hash: string = keccak256FromUtf8('l1 tx hash')
    const batchNumber: number = 1
    dataService.nextBatch.push({
      l1TxBatchTxHash: hash,
      l1StateRootBatchTxHash: undefined,
      status: L2BatchStatus.TXS_SUBMITTED,
      l2BatchNumber: batchNumber,
      transactions: [
        {
          timestamp: 1,
          blockNumber: 2,
          transactionHash: keccak256FromUtf8('l2 tx hash'),
          transactionIndex: 0,
          to: Wallet.createRandom().address,
          from: Wallet.createRandom().address,
          nonce: 1,
          calldata: keccak256FromUtf8('some calldata'),
          stateRoot: keccak256FromUtf8('l2 state root'),
          signature: 'ab'.repeat(65),
        },
      ],
    })

    canonicalTransactionChain.responses.push({ hash } as any)
    stateCommitmentChain.responses.push({ hash } as any)

    await batchSubmitter.runTask()

    dataService.txBatchesSubmitted.length.should.equal(
      0,
      'No tx batches should have been submitted!'
    )

    dataService.txBatchesConfirmed.length.should.equal(
      1,
      'No tx batches confirmed!'
    )
    dataService.txBatchesConfirmed[0].txHash.should.equal(
      hash,
      'Incorrect tx hash confirmed!'
    )
    dataService.txBatchesConfirmed[0].batchNumber.should.equal(
      batchNumber,
      'Incorrect tx batch number confirmed!'
    )

    dataService.stateBatchesSubmitted.length.should.equal(
      1,
      'No state batches submitted!'
    )
    dataService.stateBatchesSubmitted[0].txHash.should.equal(
      hash,
      'Incorrect tx hash state root confirmed!'
    )
    dataService.stateBatchesSubmitted[0].batchNumber.should.equal(
      batchNumber,
      'Incorrect tx batch number state root confirmed!'
    )

    dataService.stateBatchesConfirmed.length.should.equal(
      1,
      'No state batches confirmed!'
    )
    dataService.stateBatchesConfirmed[0].txHash.should.equal(
      hash,
      'Incorrect tx hash state root confirmed!'
    )
    dataService.stateBatchesConfirmed[0].batchNumber.should.equal(
      batchNumber,
      'Incorrect tx batch number state root confirmed!'
    )
  })

  it('should send roots if there is a batch in TXS_CONFIRMED status', async () => {
    const hash: string = keccak256FromUtf8('l1 tx hash')
    const batchNumber: number = 1
    dataService.nextBatch.push({
      l1TxBatchTxHash: hash,
      l1StateRootBatchTxHash: undefined,
      status: L2BatchStatus.TXS_CONFIRMED,
      l2BatchNumber: batchNumber,
      transactions: [
        {
          timestamp: 1,
          blockNumber: 2,
          transactionHash: keccak256FromUtf8('l2 tx hash'),
          transactionIndex: 0,
          to: Wallet.createRandom().address,
          from: Wallet.createRandom().address,
          nonce: 1,
          calldata: keccak256FromUtf8('some calldata'),
          stateRoot: keccak256FromUtf8('l2 state root'),
          signature: 'ab'.repeat(65),
        },
      ],
    })

    canonicalTransactionChain.responses.push({ hash } as any)
    stateCommitmentChain.responses.push({ hash } as any)

    await batchSubmitter.runTask()

    dataService.txBatchesSubmitted.length.should.equal(
      0,
      'No tx batches should have been submitted!'
    )

    dataService.txBatchesConfirmed.length.should.equal(
      0,
      'No tx batches should have been confirmed!'
    )

    dataService.stateBatchesSubmitted.length.should.equal(
      1,
      'No state batches submitted!'
    )
    dataService.stateBatchesSubmitted[0].txHash.should.equal(
      hash,
      'Incorrect tx hash state root confirmed!'
    )
    dataService.stateBatchesSubmitted[0].batchNumber.should.equal(
      batchNumber,
      'Incorrect tx batch number state root confirmed!'
    )

    dataService.stateBatchesConfirmed.length.should.equal(
      1,
      'No state batches confirmed!'
    )
    dataService.stateBatchesConfirmed[0].txHash.should.equal(
      hash,
      'Incorrect tx hash state root confirmed!'
    )
    dataService.stateBatchesConfirmed[0].batchNumber.should.equal(
      batchNumber,
      'Incorrect tx batch number state root confirmed!'
    )
  })

  it('should wait for roots tx if there is a batch in ROOTS_SUBMITTED status', async () => {
    const hash: string = keccak256FromUtf8('l1 tx hash')
    const batchNumber: number = 1
    dataService.nextBatch.push({
      l1TxBatchTxHash: hash,
      l1StateRootBatchTxHash: hash,
      status: L2BatchStatus.ROOTS_SUBMITTED,
      l2BatchNumber: batchNumber,
      transactions: [
        {
          timestamp: 1,
          blockNumber: 2,
          transactionHash: keccak256FromUtf8('l2 tx hash'),
          transactionIndex: 0,
          to: Wallet.createRandom().address,
          from: Wallet.createRandom().address,
          nonce: 1,
          calldata: keccak256FromUtf8('some calldata'),
          stateRoot: keccak256FromUtf8('l2 state root'),
          signature: 'ab'.repeat(65),
        },
      ],
    })

    canonicalTransactionChain.responses.push({ hash } as any)
    stateCommitmentChain.responses.push({ hash } as any)

    await batchSubmitter.runTask()

    dataService.txBatchesSubmitted.length.should.equal(
      0,
      'No tx batches should have been submitted!'
    )

    dataService.txBatchesConfirmed.length.should.equal(
      0,
      'No tx batches should have been confirmed!'
    )

    dataService.stateBatchesSubmitted.length.should.equal(
      0,
      'No state batches should have been submitted!'
    )

    dataService.stateBatchesConfirmed.length.should.equal(
      1,
      'No state batches confirmed!'
    )
    dataService.stateBatchesConfirmed[0].txHash.should.equal(
      hash,
      'Incorrect tx hash state root confirmed!'
    )
    dataService.stateBatchesConfirmed[0].batchNumber.should.equal(
      batchNumber,
      'Incorrect tx batch number state root confirmed!'
    )
  })

  describe('waiting for confirmations', () => {
    beforeEach(() => {
      batchSubmitter = new L1BatchSubmitter(
        dataService,
        canonicalTransactionChain as any,
        stateCommitmentChain as any,
        2
      )
    })

    it('should wait for tx confirmations', async () => {
      const hash: string = keccak256FromUtf8('l1 tx hash')
      const batchNumber: number = 1
      dataService.nextBatch.push({
        l1TxBatchTxHash: hash,
        l1StateRootBatchTxHash: undefined,
        status: L2BatchStatus.TXS_SUBMITTED,
        l2BatchNumber: batchNumber,
        transactions: [
          {
            timestamp: 1,
            blockNumber: 2,
            transactionHash: keccak256FromUtf8('l2 tx hash'),
            transactionIndex: 0,
            to: Wallet.createRandom().address,
            from: Wallet.createRandom().address,
            nonce: 1,
            calldata: keccak256FromUtf8('some calldata'),
            stateRoot: keccak256FromUtf8('l2 state root'),
            signature: 'ab'.repeat(65),
          },
        ],
      })

      canonicalTransactionChain.responses.push({ hash } as any)
      stateCommitmentChain.responses.push({ hash } as any)

      batchSubmitter.runTask()

      await sleep(1000)

      dataService.txBatchesConfirmed.length.should.equal(
        0,
        'batch should not yet be confirmed'
      )

      canonicalProvider.confirmedTxs.set(hash, {} as any)

      await sleep(2_000)

      dataService.txBatchesConfirmed.length.should.equal(
        1,
        'No tx batches confirmed!'
      )
      dataService.txBatchesConfirmed[0].txHash.should.equal(
        hash,
        'Incorrect tx hash confirmed!'
      )
      dataService.txBatchesConfirmed[0].batchNumber.should.equal(
        batchNumber,
        'Incorrect tx batch number confirmed!'
      )

      // the rest omitted because they're confirmed in tests above
    })

    it('should wait for state root confirmations', async () => {
      const hash: string = keccak256FromUtf8('l1 tx hash')
      const batchNumber: number = 1
      dataService.nextBatch.push({
        l1TxBatchTxHash: hash,
        l1StateRootBatchTxHash: hash,
        status: L2BatchStatus.ROOTS_SUBMITTED,
        l2BatchNumber: batchNumber,
        transactions: [
          {
            timestamp: 1,
            blockNumber: 2,
            transactionHash: keccak256FromUtf8('l2 tx hash'),
            transactionIndex: 0,
            to: Wallet.createRandom().address,
            from: Wallet.createRandom().address,
            nonce: 1,
            calldata: keccak256FromUtf8('some calldata'),
            stateRoot: keccak256FromUtf8('l2 state root'),
            signature: 'ab'.repeat(65),
          },
        ],
      })

      stateCommitmentChain.responses.push({ hash } as any)

      batchSubmitter.runTask()

      await sleep(1000)

      dataService.stateBatchesConfirmed.length.should.equal(
        0,
        'batch should not yet be confirmed'
      )

      stateCommitmentProvider.confirmedTxs.set(hash, {} as any)

      await sleep(2_000)

      dataService.stateBatchesConfirmed.length.should.equal(
        1,
        'No state root batches confirmed!'
      )
      dataService.stateBatchesConfirmed[0].txHash.should.equal(
        hash,
        'Incorrect state root hash confirmed!'
      )
      dataService.stateBatchesConfirmed[0].batchNumber.should.equal(
        batchNumber,
        'Incorrect state root batch number confirmed!'
      )
    })
  })
})
