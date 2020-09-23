import '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, NULL_ADDRESS } from '@eth-optimism/core-utils'
import { Contract, Signer, ContractFactory } from 'ethers'
import { TransactionReceipt } from 'ethers/providers'

/* Internal Imports */
import {
  GAS_LIMIT,
  manuallyDeployOvmContractReturnReceipt,
  didCreateSucceed,
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
  getDefaultGasMeterParams,
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

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)
  })

  let ExecutionManager: ContractFactory
  let SafetyChecker: ContractFactory
  let DummyContract: ContractFactory
  let AddThree: ContractFactory
  before(async () => {
    ExecutionManager = await ethers.getContractFactory('ExecutionManager')
    SafetyChecker = await ethers.getContractFactory('SafetyChecker')
    DummyContract = await ethers.getContractFactory('DummyContract')
    AddThree = await ethers.getContractFactory('AddThree')
  })

  let safetyChecker: Contract
  before(async () => {
    safetyChecker = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'SafetyChecker',
      {
        factory: SafetyChecker,
        params: [resolver.addressResolver.address],
      }
    )
  })

  let executionManager: Contract
  beforeEach(async () => {
    executionManager = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'ExecutionManager',
      {
        factory: ExecutionManager,
        params: [
          resolver.addressResolver.address,
          NULL_ADDRESS,
          getDefaultGasMeterParams(),
        ],
      }
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

    it.skip('should successfully deploy a safe contract', async () => {
      // Skipping because this uses events to verify success

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
