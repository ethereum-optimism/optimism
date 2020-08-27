import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import {
  getLogger,
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
  Address,
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
} from '../../test-helpers'

/* Logging */
const log = getLogger('partial-state-manager-gas-metering', true)

/* Testing Constants */

const OVM_TX_BASE_GAS_FEE = 30_000
const OVM_TX_MAX_GAS = 100_000_000
const GAS_RATE_LIMIT_EPOCH_IN_SECONDS = 60_000
const MAX_GAS_PER_EPOCH = 2_000_000_000

const key = '0x' + '12'.repeat(32)
const numIterationsToDo = 30
const startIndexForMultipleKeys = 1
const multipleSequentialKeys = new Array<string>(numIterationsToDo)
  .fill('lol')
  .map((v, i) => {
    return numberToHexString(startIndexForMultipleKeys + i, 32)
  })
const val = '0x' + '23'.repeat(32)

/*********
 * TESTS *
 *********/

describe.skip('Partial State Manager -- Storage Performance Testing Script', () => {
  let wallet: Signer
  let walletAddress: string
  let resolver: AddressResolverMapping
  let ExecutionManager: ContractFactory
  let PartialStateManager: ContractFactory

  let executionManager: Contract
  let stateManager: Contract

  let SimpleStorage: ContractFactory
  let simpleStorageNative: Contract
  const simpleStorageOVMAddress: Address = '0x' + '45'.repeat(20)

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

  const getCumulativeSequencedGas = async (): Promise<number> => {
    return hexStrToNumber(
      (await executionManager.getCumulativeSequencedGas())._hex
    )
  }

  const getChangeInCumulativeGas = async (
    callbackConsumingGas: () => Promise<any>
  ): Promise<{
    internalToOVM: number
    additionalExecuteTransactionOverhead: number
  }> => {
    const before: number = await getCumulativeSequencedGas()
    const tx = await callbackConsumingGas()
    const after: number = await getCumulativeSequencedGas()

    const receipt = await executionManager.provider.getTransactionReceipt(
      tx.hash
    )

    const change = after - before

    return {
      internalToOVM: change,
      additionalExecuteTransactionOverhead:
        hexStrToNumber(receipt.gasUsed._hex) - change,
    }
  }

  before(async () => {
    ;[wallet] = await ethers.getSigners()
    walletAddress = await wallet.getAddress()
    resolver = await makeAddressResolver(wallet)
    ExecutionManager = await ethers.getContractFactory('ExecutionManager')
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
    for (const aKey of multipleSequentialKeys) {
      await stateManager.insertVerifiedStorage(
        simpleStorageOVMAddress,
        aKey,
        '0x' + '00'.repeat(32)
      )
    }
  })

  it('setStorage', async () => {
    const doCall = getSimpleStorageCallCallback('setStorage', [key, val])
    const change = await getChangeInCumulativeGas(doCall)
    log.debug(
      `OVM gas cost of a single setStorage is ${change.internalToOVM}, with + ${change.additionalExecuteTransactionOverhead} EVM overhead from executeTransaction().`
    )
  })

  it('getstorage', async () => {
    const doCall = getSimpleStorageCallCallback('getStorage', [key])
    const change = await getChangeInCumulativeGas(doCall)
    log.debug(
      `OVM gas cost of a single getStorage is ${change.internalToOVM}, with + ${change.additionalExecuteTransactionOverhead} EVM overhead from executeTransaction().`
    )
  })

  it(`getstorages (${numIterationsToDo}x)`, async () => {
    const doCall = getSimpleStorageCallCallback('getStorages', [
      key,
      numIterationsToDo,
    ])
    const change = await getChangeInCumulativeGas(doCall)
    log.debug(
      `OVM gas cost of ${numIterationsToDo} getStorages is ${change.internalToOVM}, with + ${change.additionalExecuteTransactionOverhead} EVM overhead from executeTransaction().`
    )
    log.debug(
      `This corresponds to an OVM cost of ${change.internalToOVM /
        numIterationsToDo} per iteration.`
    )
  })

  it(`setSameSlotRepeated (${numIterationsToDo}x set storage, same key)`, async () => {
    const doCall = getSimpleStorageCallCallback('setSameSlotRepeated', [
      key,
      val,
      numIterationsToDo,
    ])
    const change = await getChangeInCumulativeGas(doCall)
    log.debug(
      `OVM gas cost of ${numIterationsToDo} setStorages (repeated) is ${change.internalToOVM}, with + ${change.additionalExecuteTransactionOverhead} EVM overhead from executeTransaction().`
    )
    log.debug(
      `This corresponds to an OVM cost of ${change.internalToOVM /
        numIterationsToDo} per iteration.`
    )
  })

  it(`setStorages (${numIterationsToDo}x, sequential unset keys)`, async () => {
    const doCall = getSimpleStorageCallCallback('setSequentialSlots', [
      startIndexForMultipleKeys,
      val,
      numIterationsToDo,
    ])
    const change = await getChangeInCumulativeGas(doCall)
    log.debug(
      `OVM gas cost of ${numIterationsToDo} setStorages (unique) is ${change.internalToOVM}, with + ${change.additionalExecuteTransactionOverhead} EVM overhead from executeTransaction().`
    )
    log.debug(
      `This corresponds to an OVM cost of ${change.internalToOVM /
        numIterationsToDo} per iteration.`
    )
  })
})
