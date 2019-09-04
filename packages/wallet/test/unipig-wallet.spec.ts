import './setup'

/* External Imports */
import { BaseDB, SimpleServer, SimpleClient } from '@pigi/core'
import MemDown from 'memdown'

/* Internal Imports */
import { UnipigWallet, Address, SignedTransaction } from '../src'

/***********
 * HELPERS *
 ***********/

// A mocked getBalances api
const getBalances = (address: Address) => {
  return {
    uni: 5,
    pigi: 10,
  }
}

// A mocked applyTransaction function
const applyTransaction = (transaction: SignedTransaction) => {
  // TODO
}

/*********
 * TESTS *
 *********/

describe('UnipigWallet', async () => {
  let db
  let unipigWallet
  let accountAddress
  let aggregator

  beforeEach(async () => {
    // Typings for MemDown are wrong so we need to cast to `any`.
    db = new BaseDB(new MemDown('') as any)
    unipigWallet = new UnipigWallet(db)
    // Now create a wallet account
    accountAddress = unipigWallet.createAccount('')
    // Initialize a mock aggregator
    aggregator = new SimpleServer(
      {
        getBalances,
      },
      'localhost',
      3000
    )
    await aggregator.listen()
    // Connect to the mock aggregator
    unipigWallet.rollup.connect(new SimpleClient('http://127.0.0.1:3000'))
  })

  afterEach(async () => {
    // Close the server
    await aggregator.close()
  })

  describe('getBalance()', async () => {
    it('should return an empty balance after initialized', async () => {
      const balances = await unipigWallet.getBalances(accountAddress)
      balances.should.deep.equal(getBalances(''))
    })
  })
})
