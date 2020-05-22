import '../setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Logging */
const log = getLogger('safety-tx-queue', true)

/* Contract Imports */
import * as SafetyTransactionQueue from '../../build/SafetyTransactionQueue.json'
import * as RollupMerkleUtils from '../../build/RollupMerkleUtils.json'

describe('SafetyTransactionQueue', () => {
  const provider = createMockProvider()
  const [wallet, canonicalTransactionChain, randomWallet] = getWallets(provider)
  const defaultTx = '0x1234'
  let safetyTxQueue
  let rollupMerkleUtils

  /* Link libraries before tests */
  before(async () => {
    rollupMerkleUtils = await deployContract(wallet, RollupMerkleUtils, [], {
      gasLimit: 6700000,
    })
  })

  beforeEach(async () => {
    safetyTxQueue = await deployContract(
      wallet,
      SafetyTransactionQueue,
      [rollupMerkleUtils.address, canonicalTransactionChain.address],
      {
        gasLimit: 6700000,
      }
    )
  })

  describe('enqueueBatch() ', async () => {
    it('should allow enqueue from any address', async () => {
      await safetyTxQueue.connect(randomWallet).enqueueTx(defaultTx)
      const batchesLength = await safetyTxQueue.getBatchHeadersLength()
      batchesLength.should.equal(1)
    })
  })

  describe('dequeue() ', async () => {
    it('should allow dequeue from canonicalTransactionChain', async () => {
      await safetyTxQueue.enqueueTx(defaultTx)
      await safetyTxQueue.connect(canonicalTransactionChain).dequeue()
      const batchesLength = await safetyTxQueue.getBatchHeadersLength()
      batchesLength.should.equal(1)
      const { txHash, timestamp } = await safetyTxQueue.batchHeaders(0)
      txHash.should.equal(
        '0x0000000000000000000000000000000000000000000000000000000000000000'
      )
      timestamp.should.equal(0)
      const front = await safetyTxQueue.front()
      front.should.equal(1)
    })
    it('should not allow dequeue from other address', async () => {
      await safetyTxQueue.enqueueTx(defaultTx)
      await safetyTxQueue
        .dequeue()
        .should.be.revertedWith(
          'VM Exception while processing transaction: revert Message sender does not have permission to dequeue'
        )
    })
  })
})
