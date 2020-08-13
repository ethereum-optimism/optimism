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

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)
  })

  let SimpleProxy: ContractFactory
  before(async () => {
    SimpleProxy = await ethers.getContractFactory('SimpleProxy')
  })

  let SafetyTxQueue: ContractFactory
  beforeEach(async () => {
    SafetyTxQueue = await ethers.getContractFactory('SafetyTransactionQueue')
  })

  let safetyTxQueue: Contract
  beforeEach(async () => {
    safetyTxQueue = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'SafetyTxQueue',
      {
        factory: SafetyTxQueue,
        params: [resolver.addressResolver.address],
      }
    )

    await resolver.addressResolver.setAddress(
      'CanonicalTransactionChain',
      await canonicalTransactionChain.getAddress()
    )
  })

  describe('enqueueBatch() ', async () => {
    it('should allow enqueue from a random EOA ', async () => {
      await safetyTxQueue.connect(randomWallet).enqueueTx(defaultTx)
      const batchesLength = await safetyTxQueue.getBatchHeadersLength()
      batchesLength.should.equal(1)
    })

    it('Should disallow calls from non-EOAs', async () => {
      const simpleProxy = await SimpleProxy.deploy()

      const data = safetyTxQueue.interface.encodeFunctionData(
        'enqueueTx',
        ['0x1234123412341234']
      )

      TestUtils.assertRevertsAsync(
        'Only EOAs can enqueue rollup transactions to the safety queue.',
        async () => {
          await simpleProxy.callContractWithData(
            safetyTxQueue.address,
            data
          )
        }
      )
    })

    it('should emit the right event on enqueue', async () => {
      const tx = await safetyTxQueue.connect(randomWallet).enqueueTx(defaultTx)
      const receipt = await safetyTxQueue.provider.getTransactionReceipt(tx.hash)
      const topic = receipt.logs[0].topics[0]

      const expectedTopic = safetyTxQueue.filters['CalldataTxEnqueued()']().topics[0]

      topic.should.equal(expectedTopic, `Did not receive expected event!`)
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
        'Only the canonical transaction chain can dequeue safety queue transactions.',
        async () => {
          await safetyTxQueue.dequeue()
        }
      )
    })
  })
})
