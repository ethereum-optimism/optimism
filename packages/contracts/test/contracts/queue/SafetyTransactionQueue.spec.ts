import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, TestUtils } from '@eth-optimism/core-utils'
import { Signer, ContractFactory, Contract } from 'ethers'

/* Logging */
const log = getLogger('safety-tx-queue', true)

/* Tests */
describe('SafetyTransactionQueue', () => {
  const defaultTx = '0x1234'

  let wallet: Signer
  let canonicalTransactionChain: Signer
  let randomWallet: Signer
  before(async () => {
    ;[
      wallet,
      canonicalTransactionChain,
      randomWallet,
    ] = await ethers.getSigners()
  })

  let SafetyTxQueue: ContractFactory
  beforeEach(async () => {
    SafetyTxQueue = await ethers.getContractFactory('SafetyTransactionQueue')
  })

  let safetyTxQueue: Contract
  beforeEach(async () => {
    safetyTxQueue = await SafetyTxQueue.deploy(
      await canonicalTransactionChain.getAddress()
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
      await TestUtils.assertRevertsAsync(
        'Message sender does not have permission to dequeue',
        async () => {
          await safetyTxQueue.dequeue()
        }
      )
    })
  })
})
