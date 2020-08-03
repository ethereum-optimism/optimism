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
  let l1ToL2TransactionPasser: Signer
  let canonicalTransactionChain: Signer
  before(async () => {
    ;[
      wallet,
      l1ToL2TransactionPasser,
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
        params: [
          resolver.addressResolver.address,
          await l1ToL2TransactionPasser.getAddress(),
        ],
      }
    )

    await resolver.addressResolver.setAddress(
      'CanonicalTransactionChain',
      await canonicalTransactionChain.getAddress()
    )
  })

  describe('enqueueBatch() ', async () => {
    it('should allow enqueue from l1ToL2TransactionPasser', async () => {
      await l1ToL2TxQueue.connect(l1ToL2TransactionPasser).enqueueTx(defaultTx) // Did not throw... success!
      const batchesLength = await l1ToL2TxQueue.getBatchHeadersLength()
      batchesLength.should.equal(1)
    })

    // TODO: Uncomment and implement when authentication mechanism is sorted out
    // it('should not allow enqueue from other address', async () => {
    //   await TestUtils.assertRevertsAsync(
    //     'Message sender does not have permission to enqueue',
    //     async () => {
    //       await l1ToL2TxQueue.enqueueTx(defaultTx)
    //     }
    //   )
    // })
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
      await TestUtils.assertRevertsAsync(
        'Message sender does not have permission to dequeue',
        async () => {
          await l1ToL2TxQueue.dequeue()
        }
      )
    })
  })
})
