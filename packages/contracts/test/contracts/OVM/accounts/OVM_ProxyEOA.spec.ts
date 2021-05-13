import { expect } from '../../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { ContractFactory, Contract, Signer } from 'ethers'
import { MockContract, smockit } from '@eth-optimism/smock'
import { toPlainObject } from 'lodash'

/* Internal Imports */
import { getContractInterface, predeploys } from '../../../../src'

describe('OVM_ProxyEOA', () => {
  let signer: Signer
  before(async () => {
    ;[signer] = await ethers.getSigners()
  })

  let Mock__OVM_ExecutionManager: MockContract
  let Mock__OVM_ECDSAContractAccount: MockContract
  before(async () => {
    Mock__OVM_ExecutionManager = await smockit('OVM_ExecutionManager', {
      address: predeploys.OVM_ExecutionManagerWrapper,
    })
    Mock__OVM_ECDSAContractAccount = await smockit('OVM_ECDSAContractAccount', {
      address: predeploys.OVM_ECDSAContractAccount,
    })
  })

  let Factory__OVM_ProxyEOA: ContractFactory
  before(async () => {
    Factory__OVM_ProxyEOA = await ethers.getContractFactory('OVM_ProxyEOA')
  })

  let OVM_ProxyEOA: Contract
  beforeEach(async () => {
    OVM_ProxyEOA = await Factory__OVM_ProxyEOA.deploy()
  })

  describe('getImplementation()', () => {
    it(`should be created with implementation at predeploy address`, async () => {
      expect(await OVM_ProxyEOA.getImplementation()).to.equal(
        predeploys.OVM_ECDSAContractAccount
      )
    })
  })

  // NOTE: Upgrades are disabled for now but will be re-enabled at a later point in time. See
  // comment in OVM_ProxyEOA.sol for additional information.
  describe.skip('upgrade()', () => {
    it(`should upgrade the proxy implementation`, async () => {
      const newImpl = `0x${'81'.repeat(20)}`
      Mock__OVM_ExecutionManager.smocked.ovmADDRESS.will.return.with(
        await signer.getAddress()
      )
      await expect(OVM_ProxyEOA.upgrade(newImpl)).to.not.be.reverted
      expect(await OVM_ProxyEOA.getImplementation()).to.equal(newImpl)
    })

    it(`should not allow upgrade of the proxy implementation by another account`, async () => {
      const newImpl = `0x${'81'.repeat(20)}`
      Mock__OVM_ExecutionManager.smocked.ovmADDRESS.will.return.with(
        ethers.constants.AddressZero
      )
      await expect(OVM_ProxyEOA.upgrade(newImpl)).to.be.revertedWith(
        'EOAs can only upgrade their own EOA implementation'
      )
    })
  })

  describe('fallback()', () => {
    it(`should call delegateCall with right calldata`, async () => {
      const data = Mock__OVM_ECDSAContractAccount.interface.encodeFunctionData(
        'execute',
        ['0x12341234']
      )

      await signer.sendTransaction({
        to: OVM_ProxyEOA.address,
        data,
      })

      expect(
        toPlainObject(Mock__OVM_ECDSAContractAccount.smocked.execute.calls[0])
      ).to.deep.include({
        _encodedTransaction: '0x12341234',
      })
    })

    it.skip(`should return data from fallback`, async () => {
      // TODO: test return data from fallback
    })

    it.skip(`should revert in fallback`, async () => {
      // TODO: test reversion from fallback
    })
  })
})
