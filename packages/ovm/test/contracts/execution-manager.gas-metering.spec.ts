import '../setup'

/* External Imports */
import {
  Address,
  DEFAULT_OPCODE_WHITELIST_MASK,
  DEFAULT_ETHNODE_GAS_LIMIT,
  getUnsignedTransactionCalldata,
} from '@eth-optimism/rollup-core'
import {
  getLogger,
  ZERO_ADDRESS,
  TestUtils,
  hexStrToNumber,
} from '@eth-optimism/core-utils'

import {
  ExecutionManagerContractDefinition as ExecutionManager,
  FullStateManagerContractDefinition as StateManager,
  TestDummyContractDefinition as DummyContract,
  TestSimpleConsumeGasConractDefinition as SimpleGas,
} from '@eth-optimism/rollup-contracts'

import { Contract, ContractFactory, ethers } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Internal Imports */
import { manuallyDeployOvmContract, ZERO_UINT, numberToBuf } from '../helpers'

export const abi = new ethers.utils.AbiCoder()

const log = getLogger('execution-manager-gas-metering', true)

/*********************
 * Testing Constants *
 *********************/

const OVM_TX_FLAT_GAS_FEE = 30_000
const OVM_TX_MAX_GAS = 1_500_000
const GAS_RATE_LIMIT_EPOCH_LENGTH = 60_000
const MAX_GAS_PER_EPOCH = 2_000_000

const SEQUENCER_ORIGIN = 0
const QUEUED_ORIGIN = 1

/*********
 * TESTS *
 *********/

describe('Execution Manager -- Gas Metering', () => {
  const provider = createMockProvider({ gasLimit: DEFAULT_ETHNODE_GAS_LIMIT })
  const [wallet] = getWallets(provider)
  // Create pointers to our execution manager & simple copier contract
  let executionManager: Contract
  let stateManager: Contract
  let gasConsumerContract: ContractFactory
  let gasConsumerAddress: Address

  const assertOvmTxRevertedWithMessage = async (
    tx: any,
    msg: string,
    _wallet: any
  ) => {
    const reciept = await _wallet.provider.getTransactionReceipt(tx.hash)
    const revertTopic = ethers.utils.id('EOACallRevert(bytes)')
    const revertEvent = reciept.logs.find((logged) => {
      return logged.topics.includes(revertTopic)
    })
    revertEvent.should.not.equal(undefined)
    revertEvent.data.should.equal(abi.encode(['bytes'], [Buffer.from(msg)]))
    return
  }

  const assertOvmTxDidNotRevert = async (tx: any, _wallet: any) => {
    const reciept = await _wallet.provider.getTransactionReceipt(tx.hash)
    const revertTopic = ethers.utils.id('EOACallRevert(bytes)')
    const revertEvent = reciept.logs.find((logged) => {
      return logged.topics.includes(revertTopic)
    })
    const didNotRevert: boolean = !revertEvent
    const msg = didNotRevert
      ? ''
      : `Expected not to find an EOACallRevert but one was found with data: ${revertEvent.data}`
    didNotRevert.should.eq(true, msg)
  }

  const getConsumeGasTx = (
    timestamp: number,
    queueOrigin: number,
    gasToConsume: number
  ): Promise<any> => {
    const internalCalldata = getUnsignedTransactionCalldata(
      gasConsumerContract,
      'consumeGasExceeding',
      [gasToConsume]
    )
    // overall tx gas padding to account for executeTransaction and SimpleGas return overhead
    const gasPad: number = 50_000
    const ovmTxGasLimit: number = gasToConsume + OVM_TX_FLAT_GAS_FEE + gasPad
    return executionManager.executeTransaction(
      timestamp,
      queueOrigin,
      gasConsumerAddress,
      internalCalldata,
      wallet.address,
      ZERO_ADDRESS,
      ovmTxGasLimit,
      false
    )
  }

  const getCumulativeQueuedGas = async (): Promise<number> => {
    return hexStrToNumber(
      (await executionManager.getCumulativeQueuedGas())._hex
    )
  }

  const getCumulativeSequencedGas = async (): Promise<number> => {
    return hexStrToNumber(
      (await executionManager.getCumulativeSequencedGas())._hex
    )
  }

  const getChangeInCumulativeGas = async (
    call: Promise<any>
  ): Promise<{ sequenced: number; queued: number }> => {
    // record value before
    const queuedBefore: number = await getCumulativeQueuedGas()
    const sequencedBefore: number = await getCumulativeSequencedGas()
    await call
    const queuedAfter: number = await getCumulativeQueuedGas()
    const sequencedAfter: number = await getCumulativeSequencedGas()

    return {
      sequenced: sequencedAfter - sequencedBefore,
      queued: queuedAfter - queuedBefore,
    }
  }

  beforeEach(async () => {
    // Before each test let's deploy a fresh ExecutionManager and GasConsumer
    // Deploy ExecutionManager the normal way
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [
        DEFAULT_OPCODE_WHITELIST_MASK,
        '0x' + '00'.repeat(20),
        [
          OVM_TX_FLAT_GAS_FEE,
          OVM_TX_MAX_GAS,
          GAS_RATE_LIMIT_EPOCH_LENGTH,
          MAX_GAS_PER_EPOCH,
          MAX_GAS_PER_EPOCH,
        ],
        true,
      ],
      { gasLimit: DEFAULT_ETHNODE_GAS_LIMIT }
    )
    // Set the state manager as well
    stateManager = new Contract(
      await executionManager.getStateManagerAddress(),
      StateManager.abi,
      wallet
    )
    // Deploy SimpleCopier with the ExecutionManager
    gasConsumerAddress = await manuallyDeployOvmContract(
      wallet,
      provider,
      executionManager,
      SimpleGas,
      [],
      1
    )
    log.debug(`Gas consumer contract address: [${gasConsumerAddress}]`)

    // Also set our simple copier Ethers contract so we can generate unsigned transactions
    gasConsumerContract = new ContractFactory(
      SimpleGas.abi as any,
      SimpleGas.bytecode
    )
  })

  const dummyCalldata = '0x123412341234'
  describe('Per-transaction gas limit', async () => {
    it('Should emit EOACallRevert event if the gas limit is higher than the max allowed', async () => {
      const gasToConsume = OVM_TX_MAX_GAS + 1
      const timestamp = 1

      const doTx = await getConsumeGasTx(
        timestamp,
        SEQUENCER_ORIGIN,
        gasToConsume
      )
      const tx = await doTx
      await assertOvmTxRevertedWithMessage(
        tx,
        'Transaction gas limit exceeds max OVM tx gas limit',
        wallet
      )
    })
  })
  describe('Cumulative gas tracking', async () => {
    const gasToConsume: number = 500_000
    const timestamp = 1
    it('Should properly track sequenced consumed gas', async () => {
      const consumeTx = getConsumeGasTx(
        timestamp,
        SEQUENCER_ORIGIN,
        gasToConsume
      )
      const change = await getChangeInCumulativeGas(consumeTx)

      change.queued.should.equal(0)
      // TODO get the SimpleGas consuming the exact gas amount input so we can check an equality
      change.sequenced.should.be.gt(gasToConsume)
    })
    it('Should properly track queued consumed gas', async () => {
      const consumeTx = getConsumeGasTx(timestamp, QUEUED_ORIGIN, gasToConsume)
      const change = await getChangeInCumulativeGas(consumeTx)

      change.sequenced.should.equal(0)
      // TODO get the SimpleGas consuming the exact gas amount input so we can check an equality
      change.queued.should.be.gt(gasToConsume)
    })
    it('Should properly track both queue and sequencer consumed gas', async () => {
      const queuedBefore: number = await getCumulativeQueuedGas()
      const sequencedBefore: number = await getCumulativeSequencedGas()

      const sequencerGasToConsume = 100_000
      const queueGasToConsume = 200_000

      const consumeQueueGasTx = await getConsumeGasTx(
        timestamp,
        QUEUED_ORIGIN,
        queueGasToConsume
      )
      await consumeQueueGasTx
      const consumeSequencerGasTx = await getConsumeGasTx(
        timestamp,
        SEQUENCER_ORIGIN,
        sequencerGasToConsume
      )
      await consumeSequencerGasTx

      const queuedAfter: number = await getCumulativeQueuedGas()
      const sequencedAfter: number = await getCumulativeSequencedGas()
      const change = {
        sequenced: sequencedAfter - sequencedBefore,
        queued: queuedAfter - queuedBefore,
      }

      // TODO get the SimpleGas consuming the exact gas amount input so we can check an equality
      change.sequenced.should.not.equal(0)
      change.queued.should.not.equal(0)
      change.queued.should.be.gt(change.sequenced)
    })
    describe('Gas rate limiting over multiple transactions', async () => {
      // start in a new epoch since the deployment takes some gas
      const startTimestamp = 1 + GAS_RATE_LIMIT_EPOCH_LENGTH
      const moreThanHalfGas: number = MAX_GAS_PER_EPOCH / 2 + 1000
      for (const [queueToFill, otherQueue] of [
        [QUEUED_ORIGIN, SEQUENCER_ORIGIN],
        [SEQUENCER_ORIGIN, QUEUED_ORIGIN],
      ]) {
        it('Should revert like-kind transactions in a full epoch, still allowing gas through the other queue', async () => {
          // Get us close to the limit
          const firstTx = await getConsumeGasTx(
            startTimestamp,
            queueToFill,
            moreThanHalfGas
          )
          await firstTx
          // Now try a tx which goes over the limit
          const secondTx = await getConsumeGasTx(
            startTimestamp,
            queueToFill,
            moreThanHalfGas
          )
          const failedTx = await secondTx
          await assertOvmTxRevertedWithMessage(
            failedTx,
            'Transaction gas limit exceeds remaining gas for this epoch and queue origin.',
            wallet
          )
          const otherQueueTx = await getConsumeGasTx(
            startTimestamp,
            otherQueue,
            moreThanHalfGas
          )
          const successTx = await otherQueueTx
          await assertOvmTxDidNotRevert(successTx, wallet)
        }).timeout(30000)
        it('Should allow gas back in at the start of a new epoch', async () => {
          // Get us close to the limit
          const firstTx = await getConsumeGasTx(
            startTimestamp,
            queueToFill,
            moreThanHalfGas
          )
          await firstTx

          // Now consume more than half gas again, but in the next epoch
          const nextEpochTimestamp =
            startTimestamp + GAS_RATE_LIMIT_EPOCH_LENGTH + 1
          const secondEpochTx = await getConsumeGasTx(
            nextEpochTimestamp,
            queueToFill,
            moreThanHalfGas
          )
          const successTx = await secondEpochTx
          await assertOvmTxDidNotRevert(successTx, wallet)
        }).timeout(30000)
      }
    })
  })
})
