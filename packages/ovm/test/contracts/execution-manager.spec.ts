import '../setup'

/* External Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { getLogger } from '@pigi/core-utils'
import { newInMemoryDB, SparseMerkleTreeImpl } from '@pigi/core-db'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as SimpleStorage from '../../build/contracts/SimpleStorage.json'
import { Contract, ContractFactory, Wallet, utils } from 'ethers'

const log = getLogger('execution-manager', true)

/*********
 * TESTS *
 *********/

describe('ExecutionManager', () => {
  const provider = createMockProvider()
  const [wallet1, wallet2] = getWallets(provider)
  let executionManager
  // Useful constants
  const ONE_FILLED_BYTES_32 = '0x' + '11'.repeat(32)
  const TWO_FILLED_BYTES_32 = '0x' + '22'.repeat(32)

  /* Deploy contracts before each test */
  beforeEach(async () => {
    // Deploy the execution manager
    executionManager = await deployContract(
      wallet1,
      ExecutionManager,
      new Array(2).fill('0x' + '00'.repeat(20)),
      {
        gasLimit: 6700000,
      }
    )
  })

  /*
   * Test SSTORE opcode
   */
  describe('ovmSSTORE', async () => {
    it('successfully stores without throwing', async () => {
      await executionManager.ovmSSTORE(ONE_FILLED_BYTES_32, TWO_FILLED_BYTES_32)
    })
  })

  /*
   * Test SLOAD opcode
   */
  describe('ovmSLOAD', async () => {
    it('loads a value immediately after it is stored', async () => {
      await executionManager.ovmSSTORE(ONE_FILLED_BYTES_32, TWO_FILLED_BYTES_32)
      const two = await executionManager.ovmSLOAD(ONE_FILLED_BYTES_32)
      // It should load the value which we just set
      two.should.equal(TWO_FILLED_BYTES_32)
    })
  })
})
