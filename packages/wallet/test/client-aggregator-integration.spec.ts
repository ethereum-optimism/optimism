import './setup'

/* External Imports */
import { SimpleClient, BaseDB, DB } from '@pigi/core'
import MemDown from 'memdown'

/* Internal Imports */
import { AGGREGATOR_MNEMONIC, getGenesisState } from './helpers'
import { UnipigWallet, MockAggregator, UNI_TOKEN_TYPE } from '../src'
import { RollupStateMachine } from '../src/types'
import { DefaultRollupStateMachine } from '../src/rollup-state-machine'

/*********
 * TESTS *
 *********/

const timeout = 20_000

describe('Mock Client/Aggregator Integration', async () => {
  let db: DB
  let accountAddress: string
  let aggregator: MockAggregator
  let unipigWallet: UnipigWallet
  let memdown: any
  const walletPassword = 'Really great password'

  beforeEach(async function() {
    this.timeout(timeout)

    memdown = new MemDown('') as any
    db = new BaseDB(memdown)
    unipigWallet = new UnipigWallet(db)

    // Now create a wallet account
    accountAddress = await unipigWallet.createAccount(walletPassword)

    const rollupStateMachine: RollupStateMachine = await DefaultRollupStateMachine.create(
      getGenesisState(accountAddress),
      db
    )

    // Initialize a mock aggregator
    await unipigWallet.unlockAccount(accountAddress, walletPassword)
    aggregator = new MockAggregator(
      rollupStateMachine,
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
    await db.close()
    memdown = undefined
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
      const recipient = 'testing123'
      const response = await unipigWallet.rollup.sendTransaction(
        {
          tokenType: UNI_TOKEN_TYPE,
          recipient,
          amount: 10,
        },
        accountAddress
      )
      response[recipient].balances.uni.should.equal(10)
    }).timeout(timeout)

    it('should successfully transfer if first faucet is requested', async () => {
      const recipient = 'testing123'
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
          recipient,
          amount: 10,
        },
        newAddress
      )
      transferRes[recipient].balances.uni.should.equal(10)
    }).timeout(timeout)
  })
})
