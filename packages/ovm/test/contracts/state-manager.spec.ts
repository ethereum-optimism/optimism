/* Internal Imports */
import '../setup'
import { OPCODE_WHITELIST_MASK } from '../../src/app'

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

  /* Deploy contracts before each test */
  beforeEach(async () => {
    // Deploy the execution manager
    executionManager = await deployContract(
      wallet1,
      ExecutionManager,
      [OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), true],
      { gasLimit: 6700000 }
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
