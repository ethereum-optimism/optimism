import '../setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

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
  const defaultTx = '0x1234'
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
      await l1ToL2TxQueue.connect(l1ToL2TransactionPasser).enqueueTx(defaultTx) // Did not throw... success!
      const batchesLength = await l1ToL2TxQueue.getBatchHeadersLength()
      batchesLength.should.equal(1)
    })
    it('should not allow enqueue from other address', async () => {
      await l1ToL2TxQueue
        .enqueueTx(defaultTx)
        .should.be.revertedWith(
          'VM Exception while processing transaction: revert Message sender does not have permission to enqueue'
        )
    })
  })

  describe('dequeue() ', async () => {
    it('should allow dequeue from canonicalTransactionChain', async () => {
      await l1ToL2TxQueue.connect(l1ToL2TransactionPasser).enqueueTx(defaultTx)
      await l1ToL2TxQueue.connect(canonicalTransactionChain).dequeue()
      const batchesLength = await l1ToL2TxQueue.getBatchHeadersLength()
      batchesLength.should.equal(1)
      const { txHash, timestamp } = await l1ToL2TxQueue.batchHeaders(0)
      txHash.should.equal(
        '0x0000000000000000000000000000000000000000000000000000000000000000'
      )
      timestamp.should.equal(0)
      const front = await l1ToL2TxQueue.front()
      front.should.equal(1)
    })
    it('should not allow dequeue from other address', async () => {
      await l1ToL2TxQueue.connect(l1ToL2TransactionPasser).enqueueTx(defaultTx)
      await l1ToL2TxQueue
        .dequeue()
        .should.be.revertedWith(
          'VM Exception while processing transaction: revert Message sender does not have permission to dequeue'
        )
    })
  })
})
