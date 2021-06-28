import { expect } from '../../../setup'

/* External Imports */
import { ethers, waffle } from 'hardhat'
import { ContractFactory, Contract, Signer, Wallet } from 'ethers'
import { MockContract, smockit } from '@eth-optimism/smock'
import { toPlainObject } from 'lodash'

/* Internal Imports */
import { predeploys } from '../../../../src'
import { DEFAULT_EIP155_TX, LibEIP155TxStruct } from '../../../helpers'

describe('OVM_ProxyEOA', () => {
  let signer: Signer
  let wallet: Wallet
  before(async () => {
    ;[signer] = await ethers.getSigners()
    const provider = waffle.provider
    ;[wallet] = provider.getWallets()
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

  describe('upgrade()', () => {
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
      const transaction = { ...DEFAULT_EIP155_TX }
      const encodedTransaction = await wallet.signTransaction(transaction)

      const data = Mock__OVM_ECDSAContractAccount.interface.encodeFunctionData(
        'execute',
        [LibEIP155TxStruct(encodedTransaction)]
      )

      await signer.sendTransaction({
        to: OVM_ProxyEOA.address,
        data,
      })

      const call = toPlainObject(
        Mock__OVM_ECDSAContractAccount.smocked.execute.calls[0]
      )
      const _transaction = call._transaction

      expect(_transaction[0]).to.deep.equal(transaction.nonce)
      expect(_transaction.nonce).to.deep.equal(transaction.nonce)
      expect(_transaction.gasPrice).to.deep.equal(transaction.gasPrice)
      expect(_transaction.gasLimit).to.deep.equal(transaction.gasLimit)
      expect(_transaction.to).to.deep.equal(transaction.to)
      expect(_transaction.data).to.deep.equal(transaction.data)
      expect(_transaction.isCreate).to.deep.equal(false)
    })

    it.skip(`should return data from fallback`, async () => {
      // TODO: test return data from fallback
    })

    it.skip(`should revert in fallback`, async () => {
      // TODO: test reversion from fallback
    })
  })
})
