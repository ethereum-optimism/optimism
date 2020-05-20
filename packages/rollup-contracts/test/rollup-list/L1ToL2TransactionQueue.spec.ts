import '../setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Internal Imports */
import { DefaultRollupBatch } from './RLhelper'

/* Logging */
const log = getLogger('l1-to-l2-tx-queue', true)

/* Contract Imports */
import * as L1ToL2TransactionQueue from '../../build/L1ToL2TransactionQueue.json'
import * as RollupMerkleUtils from '../../build/RollupMerkleUtils.json'

/* Begin tests */
describe('L1ToL2TransactionQueue', () => {
  const provider = createMockProvider()
  const [
    wallet,
    l1ToL2TransactionPasser,
    canonicalTransactionChain,
  ] = getWallets(provider)
  let l1ToL2TxQueue
  let rollupMerkleUtils

  /* Link libraries before tests */
  before(async () => {
    rollupMerkleUtils = await deployContract(wallet, RollupMerkleUtils, [], {
      gasLimit: 6700000,
    })
  })

  /* Deploy a new RollupChain before each test */
  beforeEach(async () => {
    l1ToL2TxQueue = await deployContract(
      wallet,
      L1ToL2TransactionQueue,
      [
        rollupMerkleUtils.address,
        l1ToL2TransactionPasser.address,
        canonicalTransactionChain.address,
      ],
      {
        gasLimit: 6700000,
      }
    )
  })

  describe('enqueueBatch() ', async () => {
    it('should allow enqueue from l1ToL2TransactionPasser', async () => {
      const batch = ['0x1234']
      await l1ToL2TxQueue.connect(l1ToL2TransactionPasser).enqueueBatch(batch) // Did not throw... success!
    })
    it('should not allow enqueue from other address', async () => {
      const batch = ['0x1234']
      await l1ToL2TxQueue
        .enqueueBatch(batch)
        .should.be.revertedWith(
          'VM Exception while processing transaction: revert Message sender does not have permission to enqueue'
        )
    })
  })

  describe('dequeueBatch() ', async () => {
    it('should allow dequeue from canonicalTransactionChain', async () => {
      const batch = ['0x1234']
      const cumulativePrevElements = 0
      const batchIndex = 0
      await l1ToL2TxQueue.connect(l1ToL2TransactionPasser).enqueueBatch(batch)
      let batchesLength = await l1ToL2TxQueue.getBatchesLength()
      let front = await l1ToL2TxQueue.front()
      let firstBatchHash = await l1ToL2TxQueue.batches(0)

      // delete the single appended batch
      await l1ToL2TxQueue.connect(canonicalTransactionChain).dequeueBatch()

      batchesLength = await l1ToL2TxQueue.getBatchesLength()
      batchesLength.should.equal(1)
      firstBatchHash = (await l1ToL2TxQueue.batches(0)).batchHeaderHash
      firstBatchHash.should.equal(
        '0x0000000000000000000000000000000000000000000000000000000000000000'
      )
      front = await l1ToL2TxQueue.front()
      front.should.equal(1)
    })
    it('should not allow dequeue from other address', async () => {
      const batch = ['0x1234']
      const cumulativePrevElements = 0
      const batchIndex = 0
      await l1ToL2TxQueue.connect(l1ToL2TransactionPasser).enqueueBatch(batch)
      await l1ToL2TxQueue
        .dequeueBatch()
        .should.be.revertedWith(
          'VM Exception while processing transaction: revert Message sender does not have permission to dequeue'
        )
    })
  })
})
