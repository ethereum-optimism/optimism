import '../setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import { Contract } from 'ethers'

/* Internal Imports */
import { GAS_LIMIT, DEFAULT_OPCODE_WHITELIST_MASK } from '../../src/app'
import { DEFAULT_ETHNODE_GAS_LIMIT } from '../helpers'

const log = getLogger('state-manager', true)

/* Begin tests */
describe('ExecutionManager', () => {
  const provider = createMockProvider({ gasLimit: DEFAULT_ETHNODE_GAS_LIMIT })
  const [wallet1, wallet2] = getWallets(provider)
  let executionManager: Contract

  /* Deploy contracts before each test */
  beforeEach(async () => {
    // Deploy the execution manager
    executionManager = await deployContract(
      wallet1,
      ExecutionManager,
      [DEFAULT_OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), GAS_LIMIT, true],
      { gasLimit: DEFAULT_ETHNODE_GAS_LIMIT }
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
