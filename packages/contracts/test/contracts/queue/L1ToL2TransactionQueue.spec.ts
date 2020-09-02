import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import {
  getLogger,
  TestUtils,
  ZERO_ADDRESS,
  hexStrToNumber,
} from '@eth-optimism/core-utils'
import { Signer, ContractFactory, Contract } from 'ethers'

/* Internal Imports */
import {
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
  getGasConsumed,
} from '../../test-helpers'

/* Logging */
const log = getLogger('l1-to-l2-tx-queue', true)

/* Tests */
describe('L1ToL2TransactionQueue', () => {
  const L2_GAS_DISCOUNT_DIVISOR = 10
  const GET_DUMMY_L1_L2_ARGS = (ovmGasLimit: number) => {
    return [ZERO_ADDRESS, ovmGasLimit, '0x1234123412341234']
  }
  const defaultL1ToL2Params = GET_DUMMY_L1_L2_ARGS(30_000)

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

  describe('enqueueL1ToL2Message() ', async () => {
    it('should allow enqueue from a random address', async () => {
      await l1ToL2TxQueue
        .connect(otherWallet)
        .enqueueL1ToL2Message(...defaultL1ToL2Params) // Did not throw... success!
      const batchesLength = await l1ToL2TxQueue.getBatchHeadersLength()
      batchesLength.should.equal(1)
    })

    it('should emit the right event on enqueue', async () => {
      const tx = await l1ToL2TxQueue
        .connect(wallet)
        .enqueueL1ToL2Message(...defaultL1ToL2Params)
      const receipt = await l1ToL2TxQueue.provider.getTransactionReceipt(
        tx.hash
      )
      const topic = receipt.logs[0].topics[0]

      const expectedTopic = l1ToL2TxQueue.filters[
        'L1ToL2TxEnqueued(address,address,uint32,bytes)'
      ]().topics[0]

      topic.should.equal(expectedTopic, `Did not receive expected event!`)
    })

    it('Should charge/burn _ovmGasLimit/L2_GAS_DISCOUNT_DIVISOR gas to enqueue', async () => {
      // do an initial enqueue to make subsequent SSTORES equivalently priced
      await l1ToL2TxQueue.enqueueL1ToL2Message(...defaultL1ToL2Params)
      // specify as hex string to ensure EOA calldata cost is the same
      const gasLimits: number[] = ['0x22000', '0x33000'].map((num) => {
        return hexStrToNumber(num)
      })
      const [lowerGasLimtArgs, higherGasLimitArgs] = gasLimits.map((num) => {
        return GET_DUMMY_L1_L2_ARGS(num)
      })

      const lowerLimitEnqueue = await l1ToL2TxQueue.enqueueL1ToL2Message(
        ...lowerGasLimtArgs
      )
      const higherLimitEnqueue = await l1ToL2TxQueue.enqueueL1ToL2Message(
        ...higherGasLimitArgs
      )

      const lowerLimitL1GasConsumed = await getGasConsumed(
        lowerLimitEnqueue,
        l1ToL2TxQueue.provider
      )
      const higherLimitL1GasConsumed = await getGasConsumed(
        higherLimitEnqueue,
        l1ToL2TxQueue.provider
      )
      const l1GasDiff = higherLimitL1GasConsumed - lowerLimitL1GasConsumed

      const expectedDiff = Math.floor((gasLimits[1] - gasLimits[0]) / 10)

      l1GasDiff.should.equal(expectedDiff)
    })
  })

  describe('dequeue() ', async () => {
    it('should allow dequeue from canonicalTransactionChain', async () => {
      await l1ToL2TxQueue
        .connect(otherWallet)
        .enqueueL1ToL2Message(...defaultL1ToL2Params)
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
      await l1ToL2TxQueue
        .connect(otherWallet)
        .enqueueL1ToL2Message(...defaultL1ToL2Params)
      await TestUtils.assertRevertsAsync(async () => {
        await l1ToL2TxQueue.dequeue()
      }, 'Only the canonical transaction chain can dequeue L1->L2 queue transactions.')
    })
  })
})
