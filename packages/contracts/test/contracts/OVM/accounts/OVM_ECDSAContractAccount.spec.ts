import { expect } from '../../../setup'

/* External Imports */
import { ethers, waffle } from 'hardhat'
import { ContractFactory, Contract, Wallet, BigNumber } from 'ethers'
import { MockContract, smockit } from '@eth-optimism/smock'
import { toPlainObject } from 'lodash'

/* Internal Imports */
import { DEFAULT_EIP155_TX } from '../../../helpers'
import { predeploys } from '../../../../src'

describe('OVM_ECDSAContractAccount', () => {
  let wallet: Wallet
  before(async () => {
    const provider = waffle.provider
    ;[wallet] = provider.getWallets()
  })

  let Mock__OVM_ExecutionManager: MockContract
  let Mock__OVM_ETH: MockContract
  before(async () => {
    Mock__OVM_ExecutionManager = await smockit('OVM_ExecutionManager', {
      address: predeploys.OVM_ExecutionManagerWrapper,
    })
    Mock__OVM_ETH = await smockit('OVM_ETH', {
      address: predeploys.OVM_ETH,
    })
  })

  let Factory__OVM_ECDSAContractAccount: ContractFactory
  before(async () => {
    Factory__OVM_ECDSAContractAccount = await ethers.getContractFactory(
      'OVM_ECDSAContractAccount'
    )
  })

  let OVM_ECDSAContractAccount: Contract
  beforeEach(async () => {
    OVM_ECDSAContractAccount = await Factory__OVM_ECDSAContractAccount.deploy()
  })

  beforeEach(async () => {
    Mock__OVM_ExecutionManager.smocked.ovmCHAINID.will.return.with(420)
    Mock__OVM_ExecutionManager.smocked.ovmGETNONCE.will.return.with(100)
    Mock__OVM_ExecutionManager.smocked.ovmADDRESS.will.return.with(
      await wallet.getAddress()
    )
    Mock__OVM_ETH.smocked.transfer.will.return.with(true)
  })

  describe('fallback()', () => {
    it(`should successfully execute an EIP155Transaction`, async () => {
      const transaction = DEFAULT_EIP155_TX
      const encodedTransaction = await wallet.signTransaction(transaction)

      await OVM_ECDSAContractAccount.execute(encodedTransaction)
    })

    it(`should ovmCREATE if EIP155Transaction.to is zero address`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, to: '' }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await OVM_ECDSAContractAccount.execute(encodedTransaction)

      const ovmCREATE: any =
        Mock__OVM_ExecutionManager.smocked.ovmCREATE.calls[0]
      expect(ovmCREATE._bytecode).to.equal(transaction.data)
    })

    it(`should revert on invalid signature`, async () => {
      const transaction = DEFAULT_EIP155_TX
      const encodedTransaction = ethers.utils.serializeTransaction(
        transaction,
        '0x' + '00'.repeat(65)
      )

      await expect(
        OVM_ECDSAContractAccount.execute(encodedTransaction)
      ).to.be.revertedWith(
        'Signature provided for EOA transaction execution is invalid.'
      )
    })

    it(`should revert on incorrect nonce`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, nonce: 99 }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await expect(
        OVM_ECDSAContractAccount.execute(encodedTransaction)
      ).to.be.revertedWith(
        'Transaction nonce does not match the expected nonce.'
      )
    })

    it(`should revert on incorrect chainId`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, chainId: 421 }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await expect(
        OVM_ECDSAContractAccount.execute(encodedTransaction)
      ).to.be.revertedWith(
        'Lib_EIP155Tx: Transaction signed with wrong chain ID'
      )
    })

    // TEMPORARY: Skip gas checks for minnet.
    it.skip(`should revert on insufficient gas`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, gasLimit: 200000000 }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await expect(
        OVM_ECDSAContractAccount.execute(encodedTransaction, {
          gasLimit: 40000000,
        })
      ).to.be.revertedWith('Gas is not sufficient to execute the transaction.')
    })

    it(`should revert if fee is not transferred to the relayer`, async () => {
      const transaction = DEFAULT_EIP155_TX
      const encodedTransaction = await wallet.signTransaction(transaction)

      Mock__OVM_ETH.smocked.transfer.will.return.with(false)

      await expect(
        OVM_ECDSAContractAccount.execute(encodedTransaction)
      ).to.be.revertedWith('Fee was not transferred to relayer.')
    })

    it(`should revert if trying to send value with a contract creation`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, value: 1234, to: '' }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await expect(
        OVM_ECDSAContractAccount.execute(encodedTransaction)
      ).to.be.revertedWith('Value transfer in contract creation not supported.')
    })

    // NOTE: Upgrades are disabled for now but will be re-enabled at a later point in time. See
    // comment in OVM_ECDSAContractAccount.sol for additional information.
    it(`should revert if trying call itself`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, to: wallet.address }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await expect(
        OVM_ECDSAContractAccount.execute(encodedTransaction)
      ).to.be.revertedWith(
        'Calls to self are disabled until upgradability is re-enabled.'
      )
    })
  })
})
