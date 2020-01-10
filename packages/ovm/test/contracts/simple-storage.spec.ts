import '../setup'

/* External Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { getLogger, add0x, abi } from '@pigi/core-utils'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as SimpleStorage from '../../build/contracts/SimpleStorage.json'
import { Contract, ContractFactory, Wallet, utils } from 'ethers'

/* Internal Imports */
import {
  manuallyDeployOvmContract,
  getUnsignedTransactionCalldata,
} from '../helpers'

const log = getLogger('simple-storage', true)

/*********
 * TESTS *
 *********/

describe.skip('SimpleStorage', () => {
  const provider = createMockProvider()
  const [wallet1, wallet2] = getWallets(provider)
  // Create pointers to our execution manager & simple storage contract
  let executionManager
  let simpleStorage
  let simpleStorageOvmAddress
  // Generate some bytes32 values used in our tests
  const ZERO_FILLED_BYTES32 = '0x' + '00'.repeat(32)
  const ONE_FILLED_BYTES32 = '0x' + '11'.repeat(32)
  const TWO_FILLED_BYTES32 = '0x' + '22'.repeat(32)

  /* Deploy contracts before each test */
  beforeEach(async () => {
    // Before each test let's deploy a fresh ExecutionManager and SimpleStorage

    // Set the ABI to consider `executeCall()` to be a "constant" function sothat we can use web3.call(executeCall(...))
    // not just web3.applyTransaction(...) -- TODO: Figure out a less hacky way to do this
    const executeCallAbiIndex = ExecutionManager.abi.reduce(
      (accumulator, method, index) => {
        if (method.name === 'executeCall') {
          // Change the method to constant so it defaults to web3.call
          return index
        }
        return accumulator
      },
      -1
    )
    ExecutionManager.abi[executeCallAbiIndex].constant = true
    // Deploy ExecutionManager the normal way
    executionManager = await deployContract(
      wallet1,
      ExecutionManager,
      new Array(2).fill('0x' + '00'.repeat(20)),
      {
        gasLimit: 6700000,
      }
    )

    // Deploy SimpleStorage with the ExecutionManager
    simpleStorageOvmAddress = await manuallyDeployOvmContract(
      provider,
      executionManager,
      SimpleStorage,
      [executionManager.address]
    )
    // Also set our simple storage ethers contract so we can generate unsigned transactions
    simpleStorage = new ContractFactory(
      SimpleStorage.abi as any, // For some reason the ABI type definition is not accepted
      SimpleStorage.bytecode
    )
  })

  describe('setStorage', async () => {
    it('properly sets storage for the contract we expect', async () => {
      // Create the variables we will use for setStorage
      const slot = add0x('99'.repeat(32))
      const value = add0x('01'.repeat(32))
      // Generate our tx calldata
      const calldata = getUnsignedTransactionCalldata(
        simpleStorage,
        'setStorage',
        [slot, value]
      )
      // Now actually apply it to our execution manager
      const tx = await executionManager.executeTransaction(
        {
          ovmEntrypoint: simpleStorageOvmAddress,
          ovmCalldata: calldata,
        },
        0,
        0
      )
      const reciept = await provider.getTransactionReceipt(tx.hash)
      // Now make sure the SetStorage event was emitted
      const rawSetStorageEvent = reciept.logs[0].data
      const decodedSetStorageEvent = abi.decode(
        ['address', 'bytes32', 'bytes32'],
        rawSetStorageEvent
      )
      // Make sure we got back what we expect
      decodedSetStorageEvent.should.deep.equal([
        simpleStorageOvmAddress,
        slot,
        value,
      ])
    })
  })

  describe('getStorage', async () => {
    it('correctly loads a value after we store it', async () => {
      // Create the variables we will use for set & get storage
      const slot = add0x('99'.repeat(32))
      const value = add0x('01'.repeat(32))

      //
      // SET STORAGE
      // Generate our tx calldata
      const setStorageCalldata = getUnsignedTransactionCalldata(
        simpleStorage,
        'setStorage',
        [slot, value]
      )
      // Apply it to our execution manager
      await executionManager.executeTransaction(
        {
          ovmEntrypoint: simpleStorageOvmAddress,
          ovmCalldata: setStorageCalldata,
        },
        0,
        0
      )

      //
      // GET STORAGE
      // Generate our tx calldata
      const calldata = getUnsignedTransactionCalldata(
        simpleStorage,
        'getStorage',
        [slot]
      )
      // Call our execution manager
      const result = await executionManager.executeCall(
        {
          ovmEntrypoint: simpleStorageOvmAddress,
          ovmCalldata: calldata,
        },
        0,
        0
      )
      // Check the result is what we expected
      result.should.equal(value)
    })
  })
})
