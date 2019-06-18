import { should } from '../../setup'

/* External Imports */
import MemDown from 'memdown'

/* Internal Imports */
import { Wallet } from '../../../src/interfaces'
import { DefaultWallet, DefaultWalletDB, BaseDB } from '../../../src/app'

describe('DefaultWallet', () => {
  let walletdb: DefaultWalletDB
  let wallet: DefaultWallet

  beforeEach(() => {
    walletdb = new DefaultWalletDB(new BaseDB(new MemDown('') as any))
    wallet = new DefaultWallet(walletdb)
  })

  describe('createAccount', () => {
    it('should correctly create an account', async () => {
      should.not.Throw(async () => {
        await wallet.createAccount('password')
      })
    })
  })
})
