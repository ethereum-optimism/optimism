import { expect } from '../../../setup'

/* External Imports */
import { waffle, ethers } from 'hardhat'
import { ContractFactory, Wallet, Contract, Signer } from 'ethers'
import { smockit, MockContract, unbind } from '@eth-optimism/smock'
import { toPlainObject } from 'lodash'

/* Internal Imports */
import { DEFAULT_EIP155_TX, LibEIP155TxStruct } from '../../../helpers'
import {
  getContractInterface,
  predeploys,
  getContractFactory,
} from '../../../../src'

describe('OVM_SequencerEntrypoint', () => {
  const iOVM_ECDSAContractAccount = getContractInterface(
    'OVM_ECDSAContractAccount'
  )

  let wallet: Wallet
  before(async () => {
    const provider = waffle.provider
    ;[wallet] = provider.getWallets()
  })

  let signer: Signer
  before(async () => {
    ;[signer] = await ethers.getSigners()
  })

  let Mock__OVM_ExecutionManager: MockContract
  before(async () => {
    Mock__OVM_ExecutionManager = await smockit('OVM_ExecutionManager', {
      address: predeploys.OVM_ExecutionManagerWrapper,
    })

    Mock__OVM_ExecutionManager.smocked.ovmCHAINID.will.return.with(420)
    Mock__OVM_ExecutionManager.smocked.ovmCREATEEOA.will.return()
  })

  let Factory__OVM_SequencerEntrypoint: ContractFactory
  before(async () => {
    Factory__OVM_SequencerEntrypoint = await ethers.getContractFactory(
      'OVM_SequencerEntrypoint'
    )
  })

  let OVM_SequencerEntrypoint: Contract
  beforeEach(async () => {
    OVM_SequencerEntrypoint = await Factory__OVM_SequencerEntrypoint.deploy()
  })

  describe('fallback()', async () => {
    it('should call ovmCREATEEOA when ovmEXTCODESIZE returns 0', async () => {
      const transaction = DEFAULT_EIP155_TX
      const encodedTransaction = await wallet.signTransaction(transaction)

      // Just unbind the smock in case it's there during this test for some reason.
      await unbind(await wallet.getAddress())

      await signer.sendTransaction({
        to: OVM_SequencerEntrypoint.address,
        data: encodedTransaction,
      })

      const call: any = Mock__OVM_ExecutionManager.smocked.ovmCREATEEOA.calls[0]
      const eoaAddress = ethers.utils.recoverAddress(call._messageHash, {
        v: call._v + 27,
        r: call._r,
        s: call._s,
      })

      expect(eoaAddress).to.equal(await wallet.getAddress())
    })

    it('should call EIP155', async () => {
      const transaction = DEFAULT_EIP155_TX
      const encodedTransaction = await wallet.signTransaction(transaction)

      const Mock__wallet = await smockit(iOVM_ECDSAContractAccount, {
        address: await wallet.getAddress(),
      })

      await signer.sendTransaction({
        to: OVM_SequencerEntrypoint.address,
        data: encodedTransaction,
      })

      const call = toPlainObject(Mock__wallet.smocked.execute.calls[0])
      const _transaction = call._transaction

      expect(_transaction[0]).to.deep.equal(transaction.nonce)
      expect(_transaction.nonce).to.deep.equal(transaction.nonce)
      expect(_transaction.gasPrice).to.deep.equal(transaction.gasPrice)
      expect(_transaction.gasLimit).to.deep.equal(transaction.gasLimit)
      expect(_transaction.to).to.deep.equal(transaction.to)
      expect(_transaction.data).to.deep.equal(transaction.data)
      expect(_transaction.isCreate).to.deep.equal(false)
    })

    it('should send correct calldata if tx is a create', async () => {
      const transaction = { ...DEFAULT_EIP155_TX, to: '' }
      const encodedTransaction = await wallet.signTransaction(transaction)

      const Mock__wallet = await smockit(iOVM_ECDSAContractAccount, {
        address: await wallet.getAddress(),
      })

      await signer.sendTransaction({
        to: OVM_SequencerEntrypoint.address,
        data: encodedTransaction,
      })

      const call = toPlainObject(Mock__wallet.smocked.execute.calls[0])
      const _transaction = call._transaction

      expect(_transaction[0]).to.deep.equal(transaction.nonce)
      expect(_transaction.nonce).to.deep.equal(transaction.nonce)
      expect(_transaction.gasPrice).to.deep.equal(transaction.gasPrice)
      expect(_transaction.gasLimit).to.deep.equal(transaction.gasLimit)
      expect(_transaction.to).to.deep.equal(ethers.constants.AddressZero)
      expect(_transaction.data).to.deep.equal(transaction.data)
      expect(_transaction.isCreate).to.deep.equal(true)
    })
  })
})
