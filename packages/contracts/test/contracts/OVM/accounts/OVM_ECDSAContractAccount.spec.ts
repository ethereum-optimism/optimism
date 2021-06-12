import { expect } from '../../../setup'

/* External Imports */
import { ethers, waffle } from 'hardhat'
import { ContractFactory, Contract, Wallet, BigNumber, utils } from 'ethers'
import { MockContract, smockit } from '@eth-optimism/smock'
import { toPlainObject } from 'lodash'

/* Internal Imports */
import { LibEIP155TxStruct, DEFAULT_EIP155_TX } from '../../../helpers'
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

  describe('fallback', async () => {
    it('should successfully accept value sent to it', async () => {
      await expect(
        wallet.sendTransaction({
          to: OVM_ECDSAContractAccount.address,
          value: 1,
        })
      ).to.not.be.reverted
    })
  })

  describe('execute()', () => {
    it(`should successfully execute an EIP155Transaction`, async () => {
      const transaction = DEFAULT_EIP155_TX
      const encodedTransaction = await wallet.signTransaction(transaction)

      await OVM_ECDSAContractAccount.execute(
        LibEIP155TxStruct(encodedTransaction)
      )
    })

    it(`should ovmCREATE if EIP155Transaction.to is zero address`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, to: '' }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await OVM_ECDSAContractAccount.execute(
        LibEIP155TxStruct(encodedTransaction)
      )

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
        OVM_ECDSAContractAccount.execute(LibEIP155TxStruct(encodedTransaction))
      ).to.be.revertedWith(
        'Signature provided for EOA transaction execution is invalid.'
      )
    })

    it(`should revert on incorrect nonce`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, nonce: 99 }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await expect(
        OVM_ECDSAContractAccount.execute(LibEIP155TxStruct(encodedTransaction))
      ).to.be.revertedWith(
        'Transaction nonce does not match the expected nonce.'
      )
    })

    it(`should revert on incorrect chainId`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, chainId: 421 }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await expect(
        OVM_ECDSAContractAccount.execute(LibEIP155TxStruct(encodedTransaction))
      ).to.be.revertedWith('Transaction signed with wrong chain ID')
    })

    // TEMPORARY: Skip gas checks for mainnet.
    it.skip(`should revert on insufficient gas`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, gasLimit: 200000000 }
      const encodedTransaction = await wallet.signTransaction(transaction)

      const tx = LibEIP155TxStruct(encodedTransaction)
      await expect(
        OVM_ECDSAContractAccount.execute(tx, {
          gasLimit: 40000000,
        })
      ).to.be.revertedWith('Gas is not sufficient to execute the transaction.')
    })

    it(`should revert if fee is not transferred to the relayer`, async () => {
      const transaction = DEFAULT_EIP155_TX
      const encodedTransaction = await wallet.signTransaction(transaction)

      Mock__OVM_ETH.smocked.transfer.will.return.with(false)

      const tx = LibEIP155TxStruct(encodedTransaction)
      await expect(OVM_ECDSAContractAccount.execute(tx)).to.be.revertedWith(
        'Fee was not transferred to relayer.'
      )
    })

    it(`should transfer value if value is greater than 0`, async () => {
      const value = 100
      const valueRecipient = '0x' + '34'.repeat(20)
      const transaction = {
        ...DEFAULT_EIP155_TX,
        to: valueRecipient,
        value,
        data: '0x',
      }
      const encodedTransaction = await wallet.signTransaction(transaction)

      // fund the contract account
      await wallet.sendTransaction({
        to: OVM_ECDSAContractAccount.address,
        value: value * 10,
        gasLimit: 1_000_000,
      })

      const receipientBalanceBefore = await wallet.provider.getBalance(
        valueRecipient
      )
      await OVM_ECDSAContractAccount.execute(
        LibEIP155TxStruct(encodedTransaction)
      )
      const recipientBalanceAfter = await wallet.provider.getBalance(
        valueRecipient
      )

      expect(
        recipientBalanceAfter.sub(receipientBalanceBefore).toNumber()
      ).to.eq(value)
    })

    it(`should revert if trying to send value with a contract creation`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, value: 1234, to: '' }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await expect(
        OVM_ECDSAContractAccount.execute(LibEIP155TxStruct(encodedTransaction))
      ).to.be.revertedWith('Value transfer in contract creation not supported.')
    })

    // NOTE: Upgrades are disabled for now but will be re-enabled at a later point in time. See
    // comment in OVM_ECDSAContractAccount.sol for additional information.
    it(`should revert if trying call itself`, async () => {
      const transaction = {
        ...DEFAULT_EIP155_TX,
        to: wallet.address,
      }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await expect(
        OVM_ECDSAContractAccount.execute(LibEIP155TxStruct(encodedTransaction))
      ).to.be.revertedWith(
        'Calls to self are disabled until upgradability is re-enabled.'
      )
    })
  })

  describe('isValidSignature()', () => {
    const message = '0x42'
    const messageHash = ethers.utils.hashMessage(message)
    it(`should revert for a malformed signature`, async () => {
      await expect(
        OVM_ECDSAContractAccount.isValidSignature(messageHash, '0xdeadbeef')
      ).to.be.revertedWith('ECDSA: invalid signature length')
    })

    it(`should return 0 for an invalid signature`, async () => {
      const signature = await wallet.signMessage(message)
      const bytes = await OVM_ECDSAContractAccount.isValidSignature(
        messageHash,
        signature
      )
      expect(bytes).to.equal('0x00000000')
    })
    // NOTE: There is no good way to unit test verifying a valid signature
    // An integration test exists testing this instead
  })
})
