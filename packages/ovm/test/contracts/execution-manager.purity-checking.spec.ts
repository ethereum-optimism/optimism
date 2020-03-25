import '../setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { Contract } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as AddThree from '../../build/contracts/AddThree.json'
import * as DummyContract from '../../build/contracts/DummyContract.json'

/* Internal Imports */
import { DEFAULT_OPCODE_WHITELIST_MASK, GAS_LIMIT } from '../../src/app'
import {
  DEFAULT_ETHNODE_GAS_LIMIT,
  manuallyDeployOvmContractReturnReceipt,
  didCreateSucceed,
} from '../helpers'
import { TransactionReceipt } from 'ethers/providers'

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
      [DEFAULT_OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), GAS_LIMIT, false],
      { gasLimit: DEFAULT_ETHNODE_GAS_LIMIT }
    )
  })
  describe('Purity Checking within Execution Manager', async () => {
    it('should fail when given an impure contract', async () => {
      // For transactions,
      const receipt: TransactionReceipt = await manuallyDeployOvmContractReturnReceipt(
        wallet,
        provider,
        executionManager,
        DummyContract,
        []
      )
      const createSucceeded = await didCreateSucceed(
        executionManager,
        receipt.transactionHash
      )

      createSucceeded.should.equal(
        false,
        `DummyContract.sol should not have been considered pure because it uses storage in its constructor`
      )
    })
    it('should successfully deploy a pure contract', async () => {
      const receipt = await manuallyDeployOvmContractReturnReceipt(
        wallet,
        provider,
        executionManager,
        AddThree,
        []
      )
      const createSucceeded = await didCreateSucceed(
        executionManager,
        receipt.transactionHash
      )

      createSucceeded.should.equal(
        true,
        `AddThree.sol contract should have been considered pure`
      )
    })
  })
})
