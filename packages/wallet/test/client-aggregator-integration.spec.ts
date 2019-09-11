import './setup'

/* External Imports */
import { SimpleClient, BaseDB } from '@pigi/core'
import MemDown from 'memdown'

/* Internal Imports */
import { AGGREGATOR_MNEMONIC, getGenesisState } from './helpers'
import { UnipigWallet, MockAggregator, UNI_TOKEN_TYPE } from '../src'

/*********
 * TESTS *
 *********/

const timeout = 20_000

describe('Mock Client/Aggregator Integration', async () => {
  let db
  let accountAddress
  let aggregator
  let unipigWallet
  const walletPassword = 'Really great password'

  beforeEach(async function() {
    this.timeout(timeout)

    // Typings for MemDown are wrong so we need to cast to `any`.
    db = new BaseDB(new MemDown('') as any)
    unipigWallet = new UnipigWallet(db)

    // Now create a wallet account
    accountAddress = await unipigWallet.createAccount(walletPassword)

    // Initialize a mock aggregator
    await unipigWallet.unlockAccount(accountAddress, walletPassword)
    aggregator = new MockAggregator(
      getGenesisState(accountAddress),
      'localhost',
      3000,
      AGGREGATOR_MNEMONIC
    )

    await aggregator.listen()
    // Connect to the mock aggregator
    unipigWallet.rollup.connect(new SimpleClient('http://127.0.0.1:3000'))
  })

  afterEach(async () => {
    if (!!aggregator) {
      // Close the server
      await aggregator.close()
    }
  })

  describe('UnipigWallet', async () => {
    it('should be able to query the aggregators balances', async () => {
      const response = await unipigWallet.getBalances(accountAddress)
      response.should.deep.equal({ uni: 50, pigi: 50 })
    }).timeout(timeout)

    it('should return an error if the wallet tries to transfer money it doesnt have', async () => {
      try {
        const response = await unipigWallet.rollup.sendTransaction(
          {
            tokenType: UNI_TOKEN_TYPE,
            recipient: 'testing123',
            amount: 10,
          },
          accountAddress
        )
      } catch (err) {
        // Success!
      }
    }).timeout(timeout)

    it('should successfully transfer if alice sends money', async () => {
      // Set "sign" to instead sign for alice
      const response = await unipigWallet.rollup.sendTransaction(
        {
          tokenType: UNI_TOKEN_TYPE,
          recipient: 'testing123',
          amount: 10,
        },
        accountAddress
      )
      response.recipient.balances.uni.should.equal(10)
    }).timeout(timeout)

    it('should successfully transfer if first faucet is requested', async () => {
      const newPassword = 'new address password'
      const newAddress = await unipigWallet.createAccount(newPassword)
      await unipigWallet.unlockAccount(newAddress, newPassword)

      // First collect some funds from the faucet
      const faucetRes = await unipigWallet.rollup.requestFaucetFunds(
        newAddress,
        10
      )
      faucetRes.should.deep.equal({
        uni: 10,
        pigi: 10,
      })
      const transferRes = await unipigWallet.rollup.sendTransaction(
        {
          tokenType: UNI_TOKEN_TYPE,
          recipient: 'testing123',
          amount: 10,
        },
        newAddress
      )
      transferRes.recipient.balances.uni.should.equal(10)
    }).timeout(timeout)
  })
})
