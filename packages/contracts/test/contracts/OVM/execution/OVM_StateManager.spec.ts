import { expect } from '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, ContractFactory } from 'ethers'

/* Internal Imports */
import {
  DUMMY_ACCOUNTS,
  DUMMY_BYTES32,
  toOVMAccount
} from '../../../helpers'

describe('OVM_StateManager', () => {
  let Factory__OVM_StateManager: ContractFactory
  before(async () => {
    Factory__OVM_StateManager = await ethers.getContractFactory(
      'OVM_StateManager'
    )
  })

  let OVM_StateManager: Contract
  beforeEach(async () => {
    OVM_StateManager = await Factory__OVM_StateManager.deploy()
  })

  describe('putAccount', () => {
    it('should be able to store an OVM account', async () => {
      await expect(
        OVM_StateManager.putAccount(
          DUMMY_ACCOUNTS[0].address,
          DUMMY_ACCOUNTS[0].data
        )
      ).to.not.be.reverted
    })

    it('should be able to overwrite an OVM account', async () => {
      await OVM_StateManager.putAccount(
        DUMMY_ACCOUNTS[0].address,
        DUMMY_ACCOUNTS[0].data
      )

      await expect(
        OVM_StateManager.putAccount(
          DUMMY_ACCOUNTS[0].address,
          DUMMY_ACCOUNTS[1].data
        )
      ).to.not.be.reverted
    })
  })

  describe('getAccount', () => {
    it('should be able to retrieve an OVM account', async () => {
      await OVM_StateManager.putAccount(
        DUMMY_ACCOUNTS[0].address,
        DUMMY_ACCOUNTS[0].data
      )

      expect(
        toOVMAccount(
          await OVM_StateManager.callStatic.getAccount(DUMMY_ACCOUNTS[0].address)
        )
      ).to.deep.equal(DUMMY_ACCOUNTS[0].data)
    })

    it('should be able to retrieve an overwritten OVM account', async () => {
      await OVM_StateManager.putAccount(
        DUMMY_ACCOUNTS[0].address,
        DUMMY_ACCOUNTS[0].data
      )

      await OVM_StateManager.putAccount(
        DUMMY_ACCOUNTS[0].address,
        DUMMY_ACCOUNTS[1].data
      )

      expect(
        toOVMAccount(
          await OVM_StateManager.callStatic.getAccount(DUMMY_ACCOUNTS[0].address)
        )
      ).to.deep.equal(DUMMY_ACCOUNTS[1].data)
    })
  })

  describe('hasAccount', () => {
    it('should return true if an account exists', async () => {
      await OVM_StateManager.putAccount(
        DUMMY_ACCOUNTS[0].address,
        DUMMY_ACCOUNTS[0].data
      )

      expect(
        await OVM_StateManager.callStatic.hasAccount(DUMMY_ACCOUNTS[0].address)
      ).to.equal(true)
    })

    it('should return false if the account does not exist', async () => {
      expect(
        await OVM_StateManager.callStatic.hasAccount(DUMMY_ACCOUNTS[0].address)
      ).to.equal(false)
    })
  })

  describe('putContractStorage', () => {
    it('should be able to insert a storage slot for a given contract', async () => {
      await expect(
        OVM_StateManager.putContractStorage(
          DUMMY_ACCOUNTS[0].address,
          DUMMY_BYTES32[0],
          DUMMY_BYTES32[1],
        )
      ).to.not.be.reverted
    })

    it('should be able to overwrite a storage slot for a given contract', async () => {
      await OVM_StateManager.putContractStorage(
        DUMMY_ACCOUNTS[0].address,
        DUMMY_BYTES32[0],
        DUMMY_BYTES32[1],
      )

      await expect(
        OVM_StateManager.putContractStorage(
          DUMMY_ACCOUNTS[0].address,
          DUMMY_BYTES32[0],
          DUMMY_BYTES32[2],
        )
      ).to.not.be.reverted
    })
  })

  describe('getContractStorage', () => {
    it('should be able to retrieve a storage slot for a given contract', async () => {
      await OVM_StateManager.putContractStorage(
        DUMMY_ACCOUNTS[0].address,
        DUMMY_BYTES32[0],
        DUMMY_BYTES32[1],
      )

      expect(
        await OVM_StateManager.callStatic.getContractStorage(
          DUMMY_ACCOUNTS[0].address,
          DUMMY_BYTES32[0]
        )
      ).to.equal(DUMMY_BYTES32[1])
    })

    it('should be able to retrieve an overwritten storage slot for a given contract', async () => {
      await OVM_StateManager.putContractStorage(
        DUMMY_ACCOUNTS[0].address,
        DUMMY_BYTES32[0],
        DUMMY_BYTES32[1],
      )

      await OVM_StateManager.putContractStorage(
        DUMMY_ACCOUNTS[0].address,
        DUMMY_BYTES32[0],
        DUMMY_BYTES32[2],
      )

      expect(
        await OVM_StateManager.callStatic.getContractStorage(
          DUMMY_ACCOUNTS[0].address,
          DUMMY_BYTES32[0]
        )
      ).to.equal(DUMMY_BYTES32[2])
    })
  })
})
