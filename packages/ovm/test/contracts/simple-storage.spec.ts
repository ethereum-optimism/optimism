import '../setup'

/* External Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { getLogger } from '@pigi/core-utils'
import { newInMemoryDB, SparseMerkleTreeImpl } from '@pigi/core-db'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as SimpleStorage from '../../build/contracts/SimpleStorage.json'
import { Contract, ContractFactory, Wallet, utils } from 'ethers'

const log = getLogger('simple-storage', true)

/*********
 * TESTS *
 *********/

describe('SimpleStorage', () => {
  const provider = createMockProvider()
  const [wallet1, wallet2] = getWallets(provider)
  // Create pointers to our execution manager & simple storage contract
  let executionManager
  let simpleStorage
  // Generate some bytes32 values used in our tests
  const ZERO_FILLED_BYTES32 = '0x' + '00'.repeat(32)
  const ONE_FILLED_BYTES32 = '0x' + '11'.repeat(32)
  const TWO_FILLED_BYTES32 = '0x' + '22'.repeat(32)

  /* Deploy contracts before each test */
  beforeEach(async () => {
    // First deploy the execution manager
    executionManager = await deployContract(
      wallet1,
      ExecutionManager,
      new Array(2).fill('0x' + '00'.repeat(20)),
      {
        gasLimit: 6700000,
      }
    )
    // Next, deploy a simpleStorage contract which uses the execution manager for storage
    simpleStorage = await deployContract(
      wallet1,
      SimpleStorage,
      [executionManager.address],
      {
        gasLimit: 6700000,
      }
    )
  })

  describe('setStorage', async () => {
    it('does not throw', async () => {
      await simpleStorage.setStorage(ONE_FILLED_BYTES32, TWO_FILLED_BYTES32)
    })

    it('stores the value & is query-able after the fact in the executionManager', async () => {
      await simpleStorage.setStorage(ONE_FILLED_BYTES32, TWO_FILLED_BYTES32)
      // Check the value stored in the execution manager
      const two = await executionManager.ovmSLOAD(ONE_FILLED_BYTES32)
      // It should load the value which we just set
      two.should.equal(TWO_FILLED_BYTES32)
    })
  })

  describe('getStorage', async () => {
    it('loads zero bytes32 if accessing something that has not been set', async () => {
      const emptyStorage = await simpleStorage.getStorage(ONE_FILLED_BYTES32)
      emptyStorage.should.equal(ZERO_FILLED_BYTES32)
    })

    it('correctly loads a value after we store it', async () => {
      await simpleStorage.setStorage(ONE_FILLED_BYTES32, TWO_FILLED_BYTES32)
      const two = await simpleStorage.getStorage(ONE_FILLED_BYTES32)
      // It should load the value which we just set
      two.should.equal(TWO_FILLED_BYTES32)
    })
  })
})
