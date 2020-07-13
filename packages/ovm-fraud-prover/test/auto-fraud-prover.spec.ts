import './setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, ContractFactory, Signer } from 'ethers'

/* Internal Imports */
import {
  getContractFactory,
  DEFAULT_OPCODE_WHITELIST_MASK,
  NULL_ADDRESS,
  GAS_LIMIT
} from './test-helpers'

describe('AutoFraudProver', () => {
  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let ExecutionManager: ContractFactory
  let RollupMerkleUtils: ContractFactory
  let CanonicalTransactionChain: ContractFactory
  let StateCommitmentChain: ContractFactory
  let FraudVerifier: ContractFactory
  before(async () => {
    ExecutionManager = getContractFactory('ExecutionManager', wallet)
    RollupMerkleUtils = getContractFactory('RollupMerkleUtils', wallet)
    StateCommitmentChain = getContractFactory('StateCommitmentChain', wallet)
    CanonicalTransactionChain = getContractFactory('CanonicalTransactionChain', wallet)
    FraudVerifier = getContractFactory('FraudVerifier', wallet)
  })

  let executionManager: Contract
  let rollupMerkleUtils: Contract
  before(async () => {
    executionManager = await ExecutionManager.deploy(
      DEFAULT_OPCODE_WHITELIST_MASK,
      NULL_ADDRESS,
      GAS_LIMIT,
      true
    )
    rollupMerkleUtils = await RollupMerkleUtils.deploy()
  })

  describe('prove', () => {
    it('should handle the complete fraud proof process', async () => {

    })
  })
})