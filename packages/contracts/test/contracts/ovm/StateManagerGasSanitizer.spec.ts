import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import {
  getLogger,
  numberToHexString,
  hexStrToNumber,
  ZERO_ADDRESS,
  hexStrToBuf,
} from '@eth-optimism/core-utils'
import { Contract, ContractFactory, Signer } from 'ethers'

/* Internal Imports */
import {
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
  manuallyDeployOvmContract,
  Address,
  executeTransaction,
} from '../../test-helpers'
import { Interface } from 'ethers/lib/utils'

/* Logging */
const log = getLogger('partial-state-manager', true)

// Hardcoded constants in the proxy contract
const GET_STORAGE_VIRTUAL_GAS_COST = 10000
const SET_STORAGE_VIRTUAL_GAS_COST = 20000

// Hardcoded gas overhead that the gas proxy functions take
const GET_STORAGE_GAS_COST_UPPER_BOUND = 50000
const SET_STORAGE_GAS_COST_UPPER_BOUND = 200000

const SM_GAS_TO_CONSUME = 30_000

/* Begin tests */
describe('StateManagerGasSanitizer', () => {
  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let resolver: AddressResolverMapping
  let stateManagerGasSanitizer: Contract

  let StateManagerGasSanitizer: ContractFactory
  let stateManagrGasSanitizer: Contract

  let StateManager: ContractFactory
  let stateManager: Contract

  let SimpleStorage: ContractFactory
  let simpleStorageAddress: Address
  before(async () => {
    resolver = await makeAddressResolver(wallet)
    StateManager = await ethers.getContractFactory('FullStateManager')
    StateManagerGasSanitizer = await ethers.getContractFactory(
      'StateManagerGasSanitizer'
    )

    stateManager = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'StateManager',
      {
        factory: StateManager,
        params: [],
      }
    )

    const sanitizerAddress = await resolver.addressResolver.getAddress(
      'StateManagerGasSanitizer'
    )
    stateManagerGasSanitizer = new Contract(
      sanitizerAddress,
      StateManagerGasSanitizer.interface
    ).connect(wallet)

    SimpleStorage = await ethers.getContractFactory(
      'SimpleStorageArgsFromCalldata'
    )
    simpleStorageAddress = await manuallyDeployOvmContract(
      wallet,
      resolver.contracts.executionManager.provider,
      resolver.contracts.executionManager,
      SimpleStorage,
      [resolver.addressResolver.address],
      1
    )
  })

  beforeEach(async () => {
    // reset so EM costs are same before each test
    await stateManagerGasSanitizer.resetOVMGasRefund()
  })

  const getOVMGasRefund = async (): Promise<number> => {
    const data = stateManagerGasSanitizer.interface.encodeFunctionData(
      'getOVMGasRefund',
      []
    )
    const res = await stateManagerGasSanitizer.provider.call({
      to: stateManagerGasSanitizer.address,
      data,
    })
    return hexStrToNumber(res)
  }

  const key = numberToHexString(1234, 32)
  const val = numberToHexString(5678, 32)

  // todo break out helper?
  const getGasConsumed = async (txRes: any): Promise<number> => {
    return hexStrToNumber(
      await (
        await resolver.contracts.executionManager.provider.getTransactionReceipt(
          txRes.hash
        )
      ).gasUsed._hex
    )
  }

  const PROXY_GET_STORAGE_OVERHEAD = 25631
  describe('Deterministic gas consumption and refunds', async () => {
    let GasConsumingProxy: ContractFactory
    before(async () => {
      GasConsumingProxy = await ethers.getContractFactory('GasConsumingProxy')
    })

    const setStorageParams = [ZERO_ADDRESS, key, val]
    // todo for loop these over all the constants?
    it('Correctly consumes the gas upper bound and records a refund', async () => {
      const tx = await stateManagerGasSanitizer.setStorage(...setStorageParams)
      const txGas = await getGasConsumed(tx)
      const refund = await getOVMGasRefund()

      txGas.should.be.greaterThan(SET_STORAGE_GAS_COST_UPPER_BOUND)
      refund.should.equal(
        SET_STORAGE_GAS_COST_UPPER_BOUND - SET_STORAGE_VIRTUAL_GAS_COST
      )

      // const txCalldataCost = estimateTxCalldataCost(stateManagerGasSanitizer.interface, 'setStorage', setStorageParams)
      // console.log(`tx gas: ${txGas}, ovm refund: ${refund}, tx calldata cost: ${txCalldataCost}`)

      // const externalGasConsumed = await getStateManagerExternalGasConsumed()
      // externalGasConsumed.should.equal(gasToConsume + GET_STORAGE_PROXY_GAS_COST)
      // virtualGasConsumed.should.equal(GET_STORAGE_VIRTUAL_GAS_COST)
    })
    it('Consumes the same amount of gas for two different SM implementations', async () => {
      const firstTx = await stateManagerGasSanitizer.setStorage(
        ...setStorageParams
      )
      const firstTxGas = await getGasConsumed(firstTx)

      // Deploy a proxy which forwards all calls to the SM, resolving that address at 'SMImpl'
      await deployAndRegister(
        resolver.addressResolver,
        wallet,
        'StateManager',
        {
          factory: GasConsumingProxy,
          params: [
            resolver.addressResolver.address,
            'StateManagerImplementation',
          ],
        }
      )

      // Deploy the SM implementation which is used by the proxy
      await deployAndRegister(
        resolver.addressResolver,
        wallet,
        'StateManagerImplementation',
        {
          factory: StateManager,
          params: [],
        }
      )

      // reset the OVM refund variable so that SSTORE cost is the same as it was above
      await stateManagerGasSanitizer.resetOVMGasRefund()

      const secondTx = await stateManagerGasSanitizer.setStorage(
        ...setStorageParams
      )
      const secondTxGas = await getGasConsumed(secondTx)

      firstTxGas.should.equal(secondTxGas)
    })
  })
  describe('Functions correctly as a proxy', async () => {
    it('Correctly forwards and returns data', async () => {
      const IDENTITY_PRECOMPILE_ADDRESS = numberToHexString(4, 20) // NICE
      resolver.addressResolver.setAddress(
        'StateManager',
        IDENTITY_PRECOMPILE_ADDRESS
      )
      const data: string = stateManagerGasSanitizer.interface.encodeFunctionData(
        'setStorage',
        [ZERO_ADDRESS, key, val]
      )
      const res = await stateManagerGasSanitizer.provider.call({
        to: stateManagerGasSanitizer.address,
        data,
      })
      // The identity precompile returns exactly what it's sent, so we should just get the same value we passed in.
      res.should.equal(data)
    })
  })
})
