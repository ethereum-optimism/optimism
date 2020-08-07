import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, numberToHexString, hexStrToNumber, ZERO_ADDRESS } from '@eth-optimism/core-utils'
import { Contract, ContractFactory, Signer } from 'ethers'

/* Internal Imports */
import {
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
} from '../../test-helpers'

/* Logging */
const log = getLogger('partial-state-manager', true)

// Hardcoded constants in the proxy contract
const GET_STORAGE_VIRTUAL_GAS_COST = 10000
const SET_STORAGE_VIRTUAL_GAS_COST = 30000

// Hardcoded gas overhead that the gas proxy functions take
const GET_STORAGE_PROXY_GAS_COST = 7217
const SET_STORAGE_PROXY_GAS_COST = 7220

/* Begin tests */
describe.only('StateManagerGasProxy', () => {
  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let resolver: AddressResolverMapping
  let DummyGasConsumer: ContractFactory
  let dummyGasConsumer: Contract
  let StateManagerGasProxy: ContractFactory
  let stateManagerGasProxy: Contract
  before(async () => {
    resolver = await makeAddressResolver(wallet)
    DummyGasConsumer = await ethers.getContractFactory('DummyGasConsumer')
    dummyGasConsumer = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'StateManager',
      {
        factory: DummyGasConsumer,
        params: [],
      }
    )
    StateManagerGasProxy = await ethers.getContractFactory('StateManagerGasProxy')
    stateManagerGasProxy = await StateManagerGasProxy.deploy(resolver.addressResolver.address)
  })

  beforeEach(async () => {
    await stateManagerGasProxy.inializeGasConsumedValues()
  })

  const getStateManagerExternalGasConsumed = async (): Promise<number> => {
    const data = stateManagerGasProxy.interface.encodeFunctionData(
      'getStateManagerExternalGasConsumed', []
    )
    const res = await stateManagerGasProxy.provider.call(
      {
        to: stateManagerGasProxy.address,
        data
      }
    )
    return hexStrToNumber(res)
  }

  const getStateManagerVirtualGasConsumed = async (): Promise<number> => {
    const data = stateManagerGasProxy.interface.encodeFunctionData(
      'getStateManagerVirtualGasConsumed', []
    )
    const res = await stateManagerGasProxy.provider.call(
      {
        to: stateManagerGasProxy.address,
        data
      }
    )
    return hexStrToNumber(res)
  }

  const key = numberToHexString(1234, 32)
  const val = numberToHexString(5678, 32)
  describe('Gas Tracking', async () => {
    it('Correctly tracks the external and virtual gas after proxying a single call', async () => {
      const gasToConsume = 100_000

      await dummyGasConsumer.setAmountGasToConsume(gasToConsume)
      await stateManagerGasProxy.getStorage(ZERO_ADDRESS, key)

      const externalGasConsumed = await getStateManagerExternalGasConsumed()
      const virtualGasConsumed = await getStateManagerVirtualGasConsumed()
      externalGasConsumed.should.equal(gasToConsume + GET_STORAGE_PROXY_GAS_COST)
      virtualGasConsumed.should.equal(GET_STORAGE_VIRTUAL_GAS_COST)
    })
    it('Correctly tracks the external and virtual gas after proxying two different calls', async () => {
      const gasToConsumeFirst = 100_000
      const gasToConsumeSecond = 200_000
      
      await dummyGasConsumer.setAmountGasToConsume(gasToConsumeFirst)
      await stateManagerGasProxy.getStorage(ZERO_ADDRESS, key)
      await dummyGasConsumer.setAmountGasToConsume(gasToConsumeSecond)
      await stateManagerGasProxy.setStorage(ZERO_ADDRESS, key, val)

      const externalGasConsumed = await getStateManagerExternalGasConsumed()
      const virtualGasConsumed = await getStateManagerVirtualGasConsumed()
      externalGasConsumed.should.equal(
        gasToConsumeFirst + GET_STORAGE_PROXY_GAS_COST + gasToConsumeSecond + SET_STORAGE_PROXY_GAS_COST
      )
      virtualGasConsumed.should.equal(
        GET_STORAGE_VIRTUAL_GAS_COST + SET_STORAGE_VIRTUAL_GAS_COST
      )
    })
  })
  describe('Functions correctly as a proxy', async () => {
    it('Correctly forwards and returns data', async () => {
      const IDENTITY_PRECOMPILE_ADDRESS = numberToHexString(4, 20) // NICE
      resolver.addressResolver.setAddress(
        'StateManager',
        IDENTITY_PRECOMPILE_ADDRESS
      )
      // The identity precompile returns exactly what it's sent, so we will get and return the same value.
      const data: string = stateManagerGasProxy.interface.encodeFunctionData(
        'setStorage',
        [
          ZERO_ADDRESS,
          key,
          val
        ]
      )
      const res = await stateManagerGasProxy.provider.call(
        {
          to: stateManagerGasProxy.address,
          data
        }
      )
      res.should.equal(data)
    })
  })
})
