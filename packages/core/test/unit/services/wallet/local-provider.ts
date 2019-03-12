import '../../../setup'

/* External Imports */
import { account as accountlib } from 'eth-lib'
import * as web3Utils from 'web3-utils'

/* Internal Imports */
import { LocalWalletProvider } from '../../../../src/services'
import { createApp, eth, walletdb } from '../../../mock'

describe('LocalWalletProvider', () => {
  const { app } = createApp({ walletdb, eth })

  const wallet = new LocalWalletProvider({ app, name: 'wallet' })

  let address: string

  it('should have dependencies', () => {
    const dependencies = ['eth', 'walletdb']
    wallet.dependencies.should.deep.equal(dependencies)
  })

  it('should have a name', () => {
    wallet.name.should.equal('wallet')
  })

  it('should a user to create an account', async () => {
    address = await wallet.createAccount()
    web3Utils.isAddress(address).should.be.true
  })

  it('should get the accounts in the wallet', async () => {
    const accounts = await wallet.getAccounts()

    accounts.should.have.lengthOf(1)
    accounts.should.deep.equal([address])
  })

  it('should get a single account', async () => {
    const account = await wallet.getAccount(address)
    const recovered = accountlib.fromPrivate(account.privateKey).address

    recovered.should.equal(address)
  })

  it('should allow a user to sign some data', async () => {
    const data = 'hello'
    const hash = web3Utils.sha3(data)
    const sig = await wallet.sign(address, data)
    const recovered = accountlib.recover(hash, sig)

    recovered.should.equal(address)
  })
})
