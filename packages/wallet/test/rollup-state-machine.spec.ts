import './setup'

/* External Imports */

/* Internal Imports */
import {
  calculateSwapWithFees,
  getGenesisState,
  getGenesisStateLargeEnoughForFees,
} from './helpers'
import {
  UNI_TOKEN_TYPE,
  MockRollupStateMachine,
  UNISWAP_ADDRESS,
  InsufficientBalanceError,
} from '../src'

/*********
 * TESTS *
 *********/

describe('RollupStateMachine', async () => {
  let rollupState
  beforeEach(() => {
    rollupState = new MockRollupStateMachine(getGenesisState(), 0)
  })

  describe('getBalances', async () => {
    it('should not throw even if the account doesnt exist', () => {
      const response = rollupState.getBalances('this is not an address!')
      response.should.deep.equal({
        uni: 0,
        pigi: 0,
      })
    })
  })

  describe('applyTransfer', async () => {
    const txAliceToBob = {
      signature: 'alice',
      transaction: {
        tokenType: UNI_TOKEN_TYPE,
        recipient: 'bob',
        amount: 5,
      },
    }

    it('should not throw when alice sends 5 uni from genesis', () => {
      rollupState
        .getBalances('alice')
        .should.deep.equal(getGenesisState().alice.balances)
      const result = rollupState.applyTransaction(txAliceToBob)
    })

    it('should update balances after transfer', () => {
      const result = rollupState.applyTransaction(txAliceToBob)
      rollupState
        .getBalances('alice')
        .uni.should.equal(getGenesisState().alice.balances.uni - 5)
      rollupState.getBalances('bob').uni.should.deep.equal(5)
    })

    it('should throw if transfering too much money', () => {
      const invalidTxApply = () =>
        rollupState.applyTransaction({
          signature: 'alice',
          transaction: {
            tokenType: UNI_TOKEN_TYPE,
            recipient: 'bob',
            amount: 500,
          },
        })
      invalidTxApply.should.throw(InsufficientBalanceError)
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

    it('should not throw when alice swaps 5 uni from genesis', () => {
      const result = rollupState.applyTransaction(txAliceSwapUni)
    })

    it('should update balances after swap', () => {
      const result = rollupState.applyTransaction(txAliceSwapUni)
      rollupState
        .getBalances('alice')
        .uni.should.equal(getGenesisState().alice.balances.uni - uniInput)
      rollupState
        .getBalances('alice')
        .pigi.should.equal(
          getGenesisState().alice.balances.pigi + expectedPigiAfterFees
        )
      // And we should have the opposite balances for uniswap
      rollupState
        .getBalances(UNISWAP_ADDRESS)
        .uni.should.equal(
          getGenesisState()[UNISWAP_ADDRESS].balances.uni + uniInput
        )
      rollupState
        .getBalances(UNISWAP_ADDRESS)
        .pigi.should.equal(
          getGenesisState()[UNISWAP_ADDRESS].balances.pigi -
            expectedPigiAfterFees
        )
    })

    it('should update balances after swap including fee', () => {
      const feeBasisPoints = 30
      rollupState = new MockRollupStateMachine(
        getGenesisStateLargeEnoughForFees(),
        feeBasisPoints
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

      rollupState.applyTransaction(txAliceSwapUni)

      rollupState
        .getBalances('alice')
        .uni.should.equal(
          getGenesisStateLargeEnoughForFees().alice.balances.uni - uniInput
        )
      rollupState
        .getBalances('alice')
        .pigi.should.equal(
          getGenesisStateLargeEnoughForFees().alice.balances.pigi +
            expectedPigiAfterFees
        )
      // And we should have the opposite balances for uniswap
      rollupState
        .getBalances(UNISWAP_ADDRESS)
        .uni.should.equal(
          getGenesisStateLargeEnoughForFees()[UNISWAP_ADDRESS].balances.uni +
            uniInput
        )
      rollupState
        .getBalances(UNISWAP_ADDRESS)
        .pigi.should.equal(
          getGenesisStateLargeEnoughForFees()[UNISWAP_ADDRESS].balances.pigi -
            expectedPigiAfterFees
        )
    })
  })
})
