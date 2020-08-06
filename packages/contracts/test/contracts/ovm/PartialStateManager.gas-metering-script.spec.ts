import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import {
  getLogger,
  remove0x,
  add0x,
  TestUtils,
  getCurrentTime,
  ZERO_ADDRESS,
  NULL_ADDRESS,
  hexStrToNumber,
  numberToHexString,
} from '@eth-optimism/core-utils'
import { Contract, ContractFactory, Signer } from 'ethers'
import { fromPairs } from 'lodash'

/* Internal Imports */
import {
  GAS_LIMIT,
  DEFAULT_OPCODE_WHITELIST_MASK,
  Address,
  manuallyDeployOvmContract,
  addressToBytes32Address,
  didCreateSucceed,
  encodeMethodId,
  encodeRawArguments,
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
} from '../../test-helpers'

/* Logging */
const log = getLogger('partial-state-manager-gas-metering', true)

/* Testing Constants */

const OVM_TX_BASE_GAS_FEE = 30_000
const OVM_TX_MAX_GAS = 1_500_000
const GAS_RATE_LIMIT_EPOCH_IN_SECONDS = 60_000
const MAX_GAS_PER_EPOCH = 10_000_000

const SEQUENCER_ORIGIN = 0
const QUEUED_ORIGIN = 1

const INITIAL_OVM_DEPLOY_TIMESTAMP = 1

const abi = new ethers.utils.AbiCoder()

// Empirically determined constant which is some extra gas the EM records due to running CALL and gasAfter - gasBefore.
// This is unfortunately not always the same--it will differ based on the size of calldata into the CALL.
// However, that size is constant for these tests, since we only call consumeGas() below.
const EXECUTE_TRANSACTION_CONSUME_GAS_OVERHEAD = 43953

/*********
 * TESTS *
 *********/

describe.only('Partial State Manager -- Gas Metering Script', () => {
  const provider = ethers.provider

  let SimpleStorage: ContractFactory
  let simpleStorageOVMAddress: Address

  let fullStateManager: Contract

  let wallet: Signer
  let walletAddress: string
  let resolver: AddressResolverMapping
  let GasConsumer: ContractFactory
  let ExecutionManager: ContractFactory
  let FullStateManager: ContractFactory
  let PartialStateManager: ContractFactory
  before(async () => {
    ;[wallet] = await ethers.getSigners()
    walletAddress = await wallet.getAddress()
    resolver = await makeAddressResolver(wallet)
    GasConsumer = await ethers.getContractFactory('GasConsumer')
    ExecutionManager = await ethers.getContractFactory('ExecutionManager')
    FullStateManager = await ethers.getContractFactory('FullStateManager')
    PartialStateManager = await ethers.getContractFactory('PartialStateManager')

    SimpleStorage = await ethers.getContractFactory('SimpleStorage')

    executionManager = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'ExecutionManager',
      {
        factory: ExecutionManager,
        params: [
          resolver.addressResolver.address,
          NULL_ADDRESS,
          [
            OVM_TX_BASE_GAS_FEE,
            OVM_TX_MAX_GAS,
            GAS_RATE_LIMIT_EPOCH_IN_SECONDS,
            MAX_GAS_PER_EPOCH,
            MAX_GAS_PER_EPOCH,
          ],
        ],
      }
    )
  })

  let executionManager: Contract

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

  const getSimpleStorageCallCallback = (methodName: string, params: any[]) => {
    const internalCallBytes = SimpleStorage.interface.encodeFunctionData(
      methodName,
      params
    )

    const EMCallBytes = ExecutionManager.interface.encodeFunctionData(
      'executeTransaction',
      [
        1_000_000,
        0,
        simpleStorageOVMAddress,
        internalCallBytes,
        walletAddress,
        ZERO_ADDRESS,
        OVM_TX_MAX_GAS,
        false,
      ]
    )

    return async () => {
      return wallet.sendTransaction({
        to: executionManager.address,
        data: EMCallBytes,
        gasLimit: GAS_LIMIT,
      })
    }
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
    callbackConsumingGas: () => Promise<any>
  ): Promise<{ internalToOVM: number; additionalExecuteTransactionOverhead: number }> => {
    // record value before
    const sequencedBefore: number = await getCumulativeSequencedGas()
    const tx = await callbackConsumingGas()
    const sequencedAfter: number = await getCumulativeSequencedGas()

    const receipt = await executionManager.provider.getTransactionReceipt(
      tx.hash
    )

    const change = sequencedAfter - sequencedBefore

    return {
      internalToOVM: change,
      additionalExecuteTransactionOverhead: hexStrToNumber(receipt.gasUsed._hex) - change,
    }
  }

  describe('Simplestorage gas meter -- OVM vs EVM -- full state manager', async () => {
    before(async () => {
      fullStateManager = await deployAndRegister(
        resolver.addressResolver,
        wallet,
        'StateManager',
        {
          factory: FullStateManager,
          params: [],
        }
      )
    })
    beforeEach(async () => {
      simpleStorageOVMAddress = await manuallyDeployOvmContract(
        wallet,
        provider,
        executionManager,
        SimpleStorage,
        [],
        INITIAL_OVM_DEPLOY_TIMESTAMP
      )
    })
    const key = '0x' + '12'.repeat(32)
    const val = '0x' + '23'.repeat(32)
    it('setStorage', async () => {
      let doCall = getSimpleStorageCallCallback('setStorage', [key, val])
      log.debug(JSON.stringify(await getChangeInCumulativeGas(doCall)))
    })
    it('getStorage', async () => {
      let doCall = getSimpleStorageCallCallback('getStorage', [key])
      log.debug(JSON.stringify(await getChangeInCumulativeGas(doCall)))
    })
  })
  describe.only('Simplestorage gas meter -- OVM vs EVM -- partial state manager', async () => {
    const key = '0x' + '12'.repeat(32)
    const startIndexForMultipleKeys = 1
    const multipleSequentialKeys = (new Array<string>(20)).fill('lol').map((val, i) => {
      return numberToHexString(startIndexForMultipleKeys + i, 32)
    })
    const val = '0x' + '23'.repeat(32)

    let stateManager: Contract

    let simpleStorageNative: Contract
    before(async () => {
      simpleStorageOVMAddress = '0x' + '45'.repeat(20)
      stateManager = (
        await deployAndRegister(
          resolver.addressResolver,
          wallet,
          'StateManager',
          {
            factory: PartialStateManager,
            params: [resolver.addressResolver.address, walletAddress],
          }
        )
      ).connect(wallet)

      simpleStorageNative = await SimpleStorage.deploy()

      await stateManager.insertVerifiedContract(walletAddress, walletAddress, 0)

      await stateManager.insertVerifiedContract(
        simpleStorageOVMAddress,
        simpleStorageNative.address,
        0
      )
    })

    beforeEach(async () => {
      // reset the single key value so that sstore costs are the same between tests
      await stateManager.insertVerifiedStorage(
        simpleStorageOVMAddress,
        key,
        '0x' + '00'.repeat(32)
      )
      // reset the sequential keys we set so sstore costs are the same between tests (new set vs update)
      for (let aKey of multipleSequentialKeys) {
        await stateManager.insertVerifiedStorage(
          simpleStorageOVMAddress,
          aKey,
          '0x' + '00'.repeat(32)
        )
      }
    })

    it('setStorage', async () => {
      let doCall = getSimpleStorageCallCallback('setStorage', [key, val])
      log.debug(JSON.stringify(await getChangeInCumulativeGas(doCall)))
    })
    it('getstorage', async () => {
      let doCall = getSimpleStorageCallCallback('getStorage', [key])
      log.debug(JSON.stringify(await getChangeInCumulativeGas(doCall)))
    })
    it('getstorages (20x)', async () => {
      let doCall = getSimpleStorageCallCallback('getStorages', [key])
      log.debug(JSON.stringify(await getChangeInCumulativeGas(doCall)))
    })
    
    it('setSameSlotRepeated (20x sstore, same key)', async () => {
      let doCall = getSimpleStorageCallCallback('setSameSlotRepeated', [key, val])
      log.debug(JSON.stringify(await getChangeInCumulativeGas(doCall)))
    })
    it('setStorages (20x sequential unset keys)', async () => {
      let doCall = getSimpleStorageCallCallback('setSequentialSlots', [startIndexForMultipleKeys, val])
      log.debug(JSON.stringify(await getChangeInCumulativeGas(doCall)))
    })

  })
})
