import './setup'

/* External Imports */
import { BaseDB, SimpleServer, SimpleClient } from '@pigi/core'
import MemDown from 'memdown'

/* Internal Imports */
import {
  UnipigWallet,
  Address,
  SignedTransaction,
  SignedStateReceipt,
} from '../src'

/***********
 * HELPERS *
 ***********/

const balances = {
  uni: 5,
  pigi: 10,
}

// A mocked getState api
const getState = (address: Address): SignedStateReceipt => {
  return {
    signature: 'mocked',
    stateReceipt: {
      address,
      stateRoot: 'mocked',
      inclusionProof: [],
      blockNumber: 1,
      transitionIndex: 0,
      state: {
        [address]: {
          balances,
        },
      },
    },
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

  const timeout = 20_000
  beforeEach(async () => {
    // Typings for MemDown are wrong so we need to cast to `any`.
    db = new BaseDB(new MemDown('') as any)
    unipigWallet = new UnipigWallet(db)
    // Now create a wallet account
    accountAddress = await unipigWallet.createAccount('')
    // Initialize a mock aggregator
    aggregator = new SimpleServer(
      {
        getState,
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

  describe('getBalance()', () => {
    it('should return an empty balance after initialized', async () => {
      const result = await unipigWallet.getBalances(accountAddress)
      result.should.deep.equal(balances)
    }).timeout(timeout)
  })
})
