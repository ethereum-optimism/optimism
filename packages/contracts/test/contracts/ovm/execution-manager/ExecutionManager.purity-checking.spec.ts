import '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger } from '@eth-optimism/core-utils'
import { Contract, Signer, ContractFactory } from 'ethers'
import { TransactionReceipt } from 'ethers/providers'

/* Internal Imports */
import {
  DEFAULT_OPCODE_WHITELIST_MASK,
  GAS_LIMIT,
} from '../../../test-helpers/core-helpers'
import {
  manuallyDeployOvmContractReturnReceipt,
  didCreateSucceed,
} from '../../../test-helpers'

/* Logging */
const log = getLogger('execution-manager-safety-checking', true)

/* Tests */
describe('Execution Manager -- Safety Checking', () => {
  const provider = ethers.provider

  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let ExecutionManager: ContractFactory
  let DummyContract: ContractFactory
  let AddThree: ContractFactory
  before(async () => {
    ExecutionManager = await ethers.getContractFactory('ExecutionManager')
    DummyContract = await ethers.getContractFactory('DummyContract')
    AddThree = await ethers.getContractFactory('AddThree')
  })

  let executionManager: Contract
  beforeEach(async () => {
    executionManager = await ExecutionManager.deploy(
      DEFAULT_OPCODE_WHITELIST_MASK,
      '0x' + '00'.repeat(20),
      GAS_LIMIT,
      false
    )
  })

  describe('Safety Checking within Execution Manager', async () => {
    it('should fail when given an unsafe contract', async () => {
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
        `DummyContract.sol should not have been considered safe because it uses storage in its constructor`
      )
    })

    it('should successfully deploy a safe contract', async () => {
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
        `AddThree.sol contract should have been considered safe`
      )
    })
  })
})
