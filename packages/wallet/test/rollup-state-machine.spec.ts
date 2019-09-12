import MemDown from 'memdown'
import './setup'

import {
  assertThrowsAsync,
  calculateSwapWithFees,
  getGenesisState,
  getGenesisStateLargeEnoughForFees,
} from './helpers'
import {
  UNI_TOKEN_TYPE,
  UNISWAP_ADDRESS,
  InsufficientBalanceError,
  IdentityVerifier,
  DefaultRollupStateMachine,
  SignedTransaction,
} from '../src'
import { DB, BaseDB } from '@pigi/core'

/* External Imports */

/* Internal Imports */

/*********
 * TESTS *
 *********/

describe('RollupStateMachine', async () => {
  let rollupState
  let db: DB

  beforeEach(async () => {
    db = new BaseDB(new MemDown('') as any, 256)
    rollupState = await DefaultRollupStateMachine.create(
      getGenesisState(),
      db,
      IdentityVerifier.instance()
    )
  })

  afterEach(async () => {
    await db.close()
  })

  describe('getBalances', async () => {
    it('should not throw even if the account doesnt exist', async () => {
      const response = await rollupState.getBalances('this is not an address!')
      response.should.deep.equal({
        uni: 0,
        pigi: 0,
      })
    })
  })

  describe('applyTransfer', async () => {
    const txAliceToBob: SignedTransaction = {
      signature: 'alice',
      transaction: {
        tokenType: UNI_TOKEN_TYPE,
        recipient: 'bob',
        amount: 5,
      },
    }

    it('should not throw when alice sends 5 uni from genesis', async () => {
      const aliceBalance = await rollupState.getBalances('alice')
      aliceBalance.should.deep.equal(getGenesisState().alice.balances)
      const result = await rollupState.applyTransaction(txAliceToBob)
    })

    it('should update balances after transfer', async () => {
      const result = await rollupState.applyTransaction(txAliceToBob)

      const aliceBalance = await rollupState.getBalances('alice')
      aliceBalance.uni.should.equal(getGenesisState().alice.balances.uni - 5)

      const bobBalance = await rollupState.getBalances('bob')
      bobBalance.uni.should.deep.equal(5)
    })

    it('should throw if transfering too much money', async () => {
      const invalidTxApply = async () =>
        rollupState.applyTransaction({
          signature: 'alice',
          transaction: {
            tokenType: UNI_TOKEN_TYPE,
            recipient: 'bob',
            amount: 500,
          },
        })
      await assertThrowsAsync(invalidTxApply, InsufficientBalanceError)
    })
  })

  describe('applySwap', async () => {
    let uniInput
    let expectedPigiAfterFees
    let txAliceSwapUni

    beforeEach(() => {
      uniInput = 25
      expectedPigiAfterFees = calculateSwapWithFees(
        uniInput,
        getGenesisState()[UNISWAP_ADDRESS].balances.uni,
        getGenesisState()[UNISWAP_ADDRESS].balances.pigi,
        0
      )

      txAliceSwapUni = {
        signature: 'alice',
        transaction: {
          tokenType: UNI_TOKEN_TYPE,
          inputAmount: uniInput,
          minOutputAmount: expectedPigiAfterFees,
          timeout: +new Date() + 1000,
        },
      }
    })

    it('should not throw when alice swaps 5 uni from genesis', async () => {
      const result = await rollupState.applyTransaction(txAliceSwapUni)
    })

    it('should update balances after swap', async () => {
      const result = await rollupState.applyTransaction(txAliceSwapUni)

      const aliceBalances = await rollupState.getBalances('alice')
      aliceBalances.uni.should.equal(
        getGenesisState().alice.balances.uni - uniInput
      )
      aliceBalances.pigi.should.equal(
        getGenesisState().alice.balances.pigi + expectedPigiAfterFees
      )

      // And we should have the opposite balances for uniswap
      const uniswapBalances = await rollupState.getBalances(UNISWAP_ADDRESS)
      uniswapBalances.uni.should.equal(
        getGenesisState()[UNISWAP_ADDRESS].balances.uni + uniInput
      )
      uniswapBalances.pigi.should.equal(
        getGenesisState()[UNISWAP_ADDRESS].balances.pigi - expectedPigiAfterFees
      )
    })

    it('should update balances after swap including fee', async () => {
      const feeBasisPoints = 30
      rollupState = await DefaultRollupStateMachine.create(
        getGenesisStateLargeEnoughForFees(),
        db,
        IdentityVerifier.instance()
      )

      uniInput = 2500
      expectedPigiAfterFees = calculateSwapWithFees(
        uniInput,
        getGenesisStateLargeEnoughForFees()[UNISWAP_ADDRESS].balances.uni,
        getGenesisStateLargeEnoughForFees()[UNISWAP_ADDRESS].balances.pigi,
        feeBasisPoints
      )

      txAliceSwapUni = {
        signature: 'alice',
        transaction: {
          tokenType: UNI_TOKEN_TYPE,
          inputAmount: uniInput,
          minOutputAmount: expectedPigiAfterFees,
          timeout: +new Date() + 1000,
        },
      }

      await rollupState.applyTransaction(txAliceSwapUni)

      const aliceBalances = await rollupState.getBalances('alice')
      aliceBalances.uni.should.equal(
        getGenesisStateLargeEnoughForFees().alice.balances.uni - uniInput
      )
      aliceBalances.pigi.should.equal(
        getGenesisStateLargeEnoughForFees().alice.balances.pigi +
          expectedPigiAfterFees
      )
      // And we should have the opposite balances for uniswap

      const uniswapBalances = await rollupState.getBalances(UNISWAP_ADDRESS)
      uniswapBalances.uni.should.equal(
        getGenesisStateLargeEnoughForFees()[UNISWAP_ADDRESS].balances.uni +
          uniInput
      )
      uniswapBalances.pigi.should.equal(
        getGenesisStateLargeEnoughForFees()[UNISWAP_ADDRESS].balances.pigi -
          expectedPigiAfterFees
      )
    })
  })
})
