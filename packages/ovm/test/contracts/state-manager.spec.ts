import '../setup'

/* External Imports */
import { getLogger } from '@pigi/core-utils'
import { newInMemoryDB, SparseMerkleTreeImpl } from '@pigi/core-db'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

const log = getLogger('state-manager', true)

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as PurityChecker from '../../build/contracts/PurityChecker.json'
import { Contract, ContractFactory, Wallet, utils } from 'ethers'

/* Begin tests */
describe('ExecutionManager', () => {
  const provider = createMockProvider()
  const [wallet1, wallet2] = getWallets(provider)
  let executionManager: Contract
  let purityChecker: Contract
  // Useful constants
  const ONE_FILLED_BYTES_32 = '0x' + '11'.repeat(32)

  /* Link libraries before tests */
  before(async () => {
    purityChecker = await deployContract(
      wallet1,
      PurityChecker,
      [ONE_FILLED_BYTES_32],
      { gasLimit: 6700000 }
    )
  })

  /* Deploy contracts before each test */
  beforeEach(async () => {
    // Deploy the execution manager
    executionManager = await deployContract(
      wallet1,
      ExecutionManager,
      [purityChecker.address, '0x' + '00'.repeat(20)],
      {
        gasLimit: 6700000,
      }
    )
  })

  /*
   * Test hello world!
   */
  describe('Hello World!', async () => {
    it('Hello World!', async () => {
      log.info('Hello World!')
    })
  })
})
