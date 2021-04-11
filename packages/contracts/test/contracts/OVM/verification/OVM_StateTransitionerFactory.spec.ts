import { expect } from '../../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { ContractFactory, Contract, constants, Signer } from 'ethers'
import { MockContract, smockit } from '@eth-optimism/smock'

/* Internal Imports */
import {
  makeAddressManager,
  DUMMY_OVM_TRANSACTIONS,
  hashTransaction,
} from '../../../helpers'

const DUMMY_HASH = hashTransaction(DUMMY_OVM_TRANSACTIONS[0])

describe('OVM_StateTransitionerFactory', () => {
  let signer1: Signer
  before(async () => {
    ;[signer1] = await ethers.getSigners()
  })

  let AddressManager: Contract
  before(async () => {
    AddressManager = await makeAddressManager()
  })

  let Factory__OVM_StateTransitionerFactory: ContractFactory
  before(async () => {
    Factory__OVM_StateTransitionerFactory = await ethers.getContractFactory(
      'OVM_StateTransitionerFactory'
    )
  })

  let OVM_StateTransitionerFactory: Contract
  let Mock__OVM_StateManagerFactory: MockContract
  beforeEach(async () => {
    OVM_StateTransitionerFactory = await Factory__OVM_StateTransitionerFactory.deploy(
      AddressManager.address
    )

    Mock__OVM_StateManagerFactory = await smockit('OVM_StateManagerFactory')
    Mock__OVM_StateManagerFactory.smocked.create.will.return.with(
      ethers.constants.AddressZero
    )

    await AddressManager.setAddress(
      'OVM_StateManagerFactory',
      Mock__OVM_StateManagerFactory.address
    )
  })

  describe('create', () => {
    describe('when the sender is not the OVM_FraudVerifier', () => {
      beforeEach(async () => {
        await AddressManager.setAddress(
          'OVM_FraudVerifier',
          constants.AddressZero
        )
      })

      it('should revert', async () => {
        await expect(
          OVM_StateTransitionerFactory.create(
            AddressManager.address,
            ethers.constants.HashZero,
            ethers.constants.HashZero,
            DUMMY_HASH
          )
        ).to.be.revertedWith(
          'Create can only be done by the OVM_FraudVerifier.'
        )
      })
    })

    describe('when the sender is the OVM_FraudVerifier', () => {
      beforeEach(async () => {
        await AddressManager.setAddress(
          'OVM_FraudVerifier',
          await signer1.getAddress()
        )
      })

      it('should not revert', async () => {
        await expect(
          OVM_StateTransitionerFactory.connect(signer1).create(
            AddressManager.address,
            ethers.constants.HashZero,
            ethers.constants.HashZero,
            DUMMY_HASH
          )
        ).to.not.be.reverted
      })
    })
  })
})
