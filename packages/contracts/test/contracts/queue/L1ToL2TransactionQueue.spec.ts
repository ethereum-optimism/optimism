import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, TestUtils } from '@eth-optimism/core-utils'
import { Signer, ContractFactory, Contract } from 'ethers'

/* Internal Imports */
import {
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
} from '../../test-helpers'

/* Logging */
const log = getLogger('l1-to-l2-tx-queue', true)

/* Tests */
describe('L1ToL2TransactionQueue', () => {
  const defaultTx = '0x1234'

  let wallet: Signer
  let otherWallet: Signer
  let canonicalTransactionChain: Signer
  before(async () => {
    ;[
      wallet,
      otherWallet,
      canonicalTransactionChain,
    ] = await ethers.getSigners()
  })

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)
  })

  let L1toL2TxQueue: ContractFactory
  before(async () => {
    L1toL2TxQueue = await ethers.getContractFactory('L1ToL2TransactionQueue')
  })

  let l1ToL2TxQueue: Contract
  beforeEach(async () => {
    l1ToL2TxQueue = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'L1toL2TxQueue',
      {
        factory: L1toL2TxQueue,
        params: [resolver.addressResolver.address],
      }
    )

    await resolver.addressResolver.setAddress(
      'CanonicalTransactionChain',
      await canonicalTransactionChain.getAddress()
    )
  })

  describe('enqueueBatch() ', async () => {
    it('should allow enqueue from a random address', async () => {
      await l1ToL2TxQueue.connect(otherWallet).enqueueTx(defaultTx) // Did not throw... success!
      const batchesLength = await l1ToL2TxQueue.getBatchHeadersLength()
      batchesLength.should.equal(1)
    })

    it('should emit the right event on enqueue', async () => {
      const tx = await l1ToL2TxQueue.connect(wallet).enqueueTx(defaultTx)
      const receipt = await l1ToL2TxQueue.provider.getTransactionReceipt(tx.hash)
      const topic = receipt.logs[0].topics[0]
      
      const expectedTopic = l1ToL2TxQueue.filters['L1ToL2TxEnqueued(bytes)']().topics[0]

      topic.should.equal(expectedTopic, `Did not receive expected event!`)
    })
  })

  describe('dequeue() ', async () => {
    it('should allow dequeue from canonicalTransactionChain', async () => {
      await l1ToL2TxQueue.connect(otherWallet).enqueueTx(defaultTx)
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
      await l1ToL2TxQueue.connect(otherWallet).enqueueTx(defaultTx)
      await TestUtils.assertRevertsAsync(
        'Only the canonical transaction chain can dequeue L1->L2 queue transactions.',
        async () => {
          await l1ToL2TxQueue.dequeue()
        }
      )
    })
  })
})
