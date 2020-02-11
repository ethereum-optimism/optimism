import '../setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { Contract, ethers } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as AddThree from '../../build/contracts/AddThree.json'
import * as DummyContract from '../../build/contracts/DummyContract.json'

/* Internal Imports */
import { OPCODE_WHITELIST_MASK, GAS_LIMIT } from '../../src/app'
import {
  manuallyDeployOvmContract,
  DEFAULT_ETHNODE_GAS_LIMIT,
} from '../helpers'

const log = getLogger('execution-manager-purity-checking', true)

/*********
 * TESTS *
 *********/

describe('Execution Manager -- Purity Checking', () => {
  const provider = createMockProvider({ gasLimit: DEFAULT_ETHNODE_GAS_LIMIT })
  const [wallet] = getWallets(provider)
  // Create pointers to our execution manager & simple copier contract
  let executionManager: Contract

  beforeEach(async () => {
    // Deploy ExecutionManager with Purity Checking enabled
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), GAS_LIMIT, false],
      { gasLimit: DEFAULT_ETHNODE_GAS_LIMIT }
    )
  })
  describe('Purity Checking within Execution Manager', async () => {
    it('should revert when given an impure contract', async () => {
      let failed = false
      try {
        await manuallyDeployOvmContract(
          wallet,
          provider,
          executionManager,
          DummyContract,
          []
        )
      } catch (e) {
        if (e.message.indexOf('revert') >= 0) {
          failed = true
        }
      }
      failed.should.equal(
        true,
        `DummyContract.sol should not have been considered pure because it uses storage in its constructor`
      )
    })
    it('should successfully deploy a pure contract', async () => {
      let failed = false
      try {
        await manuallyDeployOvmContract(
          wallet,
          provider,
          executionManager,
          AddThree,
          []
        )
      } catch (e) {
        if (e.message.indexOf('revert') >= 0) {
          failed = true
        }
      }
      failed.should.equal(
        false,
        `AddThree.sol contract should have been considered pure`
      )
    })
  })
})
