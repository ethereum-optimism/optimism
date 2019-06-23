import { should } from '../../setup'

/* External Imports */
import MemDown from 'memdown'
import { ethers } from 'ethers'

/* Internal Imports */
import { Wallet } from '../../../src/types'
import { DefaultWallet, DefaultWalletDB, BaseDB } from 'src/app'

describe('DefaultWallet', () => {
  let walletdb: DefaultWalletDB
  let wallet: DefaultWallet

  beforeEach(() => {
    // Typings for MemDown are wrong so we need to cast to `any`.
    walletdb = new DefaultWalletDB(new BaseDB(new MemDown('') as any))
    wallet = new DefaultWallet(walletdb)
  })

  describe('createAccount', () => {
    it('should correctly create an account', async () => {
      await wallet.createAccount('password').should.be.fulfilled
    }).timeout(5000)
  })

  describe('listAccounts', () => {
    it('should return an empty list if there are no accounts', async () => {
      const accounts = await wallet.listAccounts()

      accounts.should.deep.equal([])
    })

    it('should return the account if an account exists', async () => {
      const account = await wallet.createAccount('password')

      const accounts = await wallet.listAccounts()

      accounts.should.deep.equal([account])
    }).timeout(5000)

    it('should return multiple accounts if more than one exists', async () => {
      const account1 = await wallet.createAccount('password')
      const account2 = await wallet.createAccount('password')

      const accounts = (await wallet.listAccounts()).sort()

      accounts.should.deep.equal([account1, account2].sort())
    }).timeout(10000)
  })

  describe('unlockAccount', () => {
    it('should unlock an account if given the right password', async () => {
      const account = await wallet.createAccount('password')

      await wallet.unlockAccount(account, 'password').should.be.fulfilled
    }).timeout(10000)

    it('should throw if trying to unlock with the wrong password', async () => {
      const account = await wallet.createAccount('password')

      await wallet
        .unlockAccount(account, 'wrongpassword')
        .should.be.rejectedWith('Invalid account password.')
    }).timeout(10000)

    it('should throw if the account does not exist', async () => {
      const account = '0x0000000000000000000000000000000000000000'

      await wallet
        .unlockAccount(account, 'password')
        .should.be.rejectedWith('Account does not exist.')
    })
  })

  describe('lockAccount', () => {
    it('should lock an account if unlocked', async () => {
      const account = await wallet.createAccount('password')

      await wallet.unlockAccount(account, 'password')

      await wallet.lockAccount(account).should.be.fulfilled
    }).timeout(5000)

    it('should lock an account even if not unlocked', async () => {
      const account = await wallet.createAccount('password')

      await wallet.lockAccount(account).should.be.fulfilled
    }).timeout(5000)
  })

  describe('sign', () => {
    it('should correctly sign some data', async () => {
      const account = await wallet.createAccount('password')

      await wallet.unlockAccount(account, 'password')
      const signature = await wallet.sign(account, 'hello world')

      ethers.utils.verifyMessage('hello world', signature).should.equal(account)
    }).timeout(5000)

    it('should throw if the account is not unlocked', async () => {
      const account = await wallet.createAccount('password')

      await wallet
        .sign(account, 'hello world')
        .should.be.rejectedWith('Account is not unlocked.')
    }).timeout(5000)

    it('should throw if the account does not exist', async () => {
      const account = '0x0000000000000000000000000000000000000000'

      await wallet
        .sign(account, 'hello world')
        .should.be.rejectedWith('Account does not exist.')
    })
  })
})
