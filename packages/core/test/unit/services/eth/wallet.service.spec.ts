import '../../../setup'

/* External Imports */
import { account as accountlib } from 'eth-lib'
import { isAddress, sha3 } from 'web3-utils'

/* Internal Imports */
import { WalletService } from '../../../../src/services'
import { logs, web3Service, walletdb } from '../../../mock'

describe('LocalWalletProvider', () => {
  const wallet = new WalletService(logs, web3Service, walletdb)
  let address: string

  it('should a user to create an account', async () => {
    address = await wallet.createAccount()
    isAddress(address).should.be.true
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
    const hash = sha3(data)
    const sig = await wallet.sign(address, data)
    const recovered = accountlib.recover(hash, sig)

    recovered.should.equal(address)
  })
})
