import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import {
  getLogger,
  TestUtils,
  numberToHexString,
  remove0x,
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
const log = getLogger('safety-tx-queue', true)

/* Tests */
describe('SafetyTransactionQueue', () => {
  const GET_TX_WITH_OVM_GAS_LIMIT = (gasLimit: number) => {
    return (
      '0x' +
      '00'.repeat(40) +
      remove0x(numberToHexString(gasLimit, 32)) +
      '12'.repeat(40)
    )
  }
  const defaultGasLimit = 30_000
  const defaultTx = GET_TX_WITH_OVM_GAS_LIMIT(defaultGasLimit)

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

      const data = safetyTxQueue.interface.encodeFunctionData('enqueueTx', [
        '0x1234123412341234',
      ])

      TestUtils.assertRevertsAsync(
        'Only EOAs can enqueue rollup transactions to the safety queue.',
        async () => {
          await simpleProxy.callContractWithData(safetyTxQueue.address, data)
        }
      )
    })

    it('should emit the right event on enqueue', async () => {
      const tx = await safetyTxQueue.connect(randomWallet).enqueueTx(defaultTx)
      const receipt = await safetyTxQueue.provider.getTransactionReceipt(
        tx.hash
      )
      const topic = receipt.logs[0].topics[0]

      const expectedTopic = safetyTxQueue.filters['CalldataTxEnqueued()']()
        .topics[0]

      topic.should.equal(expectedTopic, `Did not receive expected event!`)
    })

    it('Should burn _ovmGasLimit/L2_GAS_DISCOUNT_DIVISOR gas to enqueue', async () => {
      // do an initial enqueue to make subsequent SSTORES equivalently priced
      await safetyTxQueue.enqueueTx(defaultTx)
      // specify as hex string to ensure EOA calldata cost is the same
      const gasLimits: number[] = ['0x22000', '0x33000'].map((num) => {
        return hexStrToNumber(num)
      })
      const [lowerGasLimitTx, higherGasLimitTx] = gasLimits.map((num) => {
        return GET_TX_WITH_OVM_GAS_LIMIT(num)
      })

      const lowerLimitEnqueue = await safetyTxQueue.enqueueTx(lowerGasLimitTx)
      const higherLimitEnqueue = await safetyTxQueue.enqueueTx(higherGasLimitTx)

      const lowerLimitL1GasConsumed = await getGasConsumed(
        lowerLimitEnqueue,
        safetyTxQueue.provider
      )
      const higherLimitL1GasConsumed = await getGasConsumed(
        higherLimitEnqueue,
        safetyTxQueue.provider
      )
      const l1GasDiff = higherLimitL1GasConsumed - lowerLimitL1GasConsumed

      const expectedDiff = Math.floor((gasLimits[1] - gasLimits[0]) / 10)

      l1GasDiff.should.equal(expectedDiff)
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
