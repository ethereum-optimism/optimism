import MemDown from 'memdown'
import './setup'
import { DB, BaseDB } from '@pigi/core'

import {
  ALICE_ADDRESS,
  ALICE_GENESIS_STATE_INDEX,
  assertThrowsAsync,
  BOB_ADDRESS,
  calculateSwapWithFees,
  getGenesisState,
  getGenesisStateLargeEnoughForFees,
  UNISWAP_GENESIS_STATE_INDEX,
} from './helpers'
import {
  UNI_TOKEN_TYPE,
  UNISWAP_ADDRESS,
  InsufficientBalanceError,
  IdentityVerifier,
  DefaultRollupStateMachine,
  SignedTransaction,
  PIGI_TOKEN_TYPE,
} from '../src'

/* External Imports */

/* Internal Imports */

/*********
 * TESTS *
 *********/

describe('RollupStateMachine', () => {
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

  describe('getBalances', () => {
    it('should not throw even if the account doesnt exist', async () => {
      const response = await rollupState.getBalances('this is not an address!')
      response.should.deep.equal({
        [UNI_TOKEN_TYPE]: 0,
        [PIGI_TOKEN_TYPE]: 0,
      })
    })
  })

  describe('applyTransfer', () => {
    const txAliceToBob: SignedTransaction = {
      signature: ALICE_ADDRESS,
      transaction: {
        sender: ALICE_ADDRESS,
        recipient: BOB_ADDRESS,
        tokenType: UNI_TOKEN_TYPE,
        amount: 5,
      },
    }

    it('should not throw when alice sends 5 uni from genesis', async () => {
      const aliceBalance = await rollupState.getBalances(ALICE_ADDRESS)
      aliceBalance.should.deep.equal(
        getGenesisState()[ALICE_GENESIS_STATE_INDEX].balances
      )
      await rollupState.applyTransaction(txAliceToBob)
    })

    it('should update balances after transfer', async () => {
      await rollupState.applyTransaction(txAliceToBob)

      const aliceBalance = await rollupState.getBalances(ALICE_ADDRESS)
      aliceBalance[UNI_TOKEN_TYPE].should.equal(
        getGenesisState()[ALICE_GENESIS_STATE_INDEX].balances[UNI_TOKEN_TYPE] -
          5
      )

      const bobBalance = await rollupState.getBalances(BOB_ADDRESS)
      bobBalance[UNI_TOKEN_TYPE].should.deep.equal(5)
    })

    it('should throw if transfering too much money', async () => {
      const invalidTxApply = async () =>
        rollupState.applyTransaction({
          signature: ALICE_ADDRESS,
          transaction: {
            sender: ALICE_ADDRESS,
            tokenType: UNI_TOKEN_TYPE,
            recipient: BOB_ADDRESS,
            amount: 500,
          },
        })
      await assertThrowsAsync(invalidTxApply, InsufficientBalanceError)
    })
  })

  describe('applySwap', () => {
    let uniInput
    let expectedPigiAfterFees
    let txAliceSwapUni

    beforeEach(() => {
      uniInput = 25
      expectedPigiAfterFees = calculateSwapWithFees(
        uniInput,
        getGenesisState()[UNISWAP_GENESIS_STATE_INDEX].balances[UNI_TOKEN_TYPE],
        getGenesisState()[UNISWAP_GENESIS_STATE_INDEX].balances[
          PIGI_TOKEN_TYPE
        ],
        0
      )

      txAliceSwapUni = {
        signature: ALICE_ADDRESS,
        transaction: {
          sender: ALICE_ADDRESS,
          tokenType: UNI_TOKEN_TYPE,
          inputAmount: uniInput,
          minOutputAmount: expectedPigiAfterFees,
          timeout: +new Date() + 1000,
        },
      }
    })

    it('should not throw when alice swaps 5 uni from genesis', async () => {
      await rollupState.applyTransaction(txAliceSwapUni)
    })

    it('should update balances after swap', async () => {
      await rollupState.applyTransaction(txAliceSwapUni)

      const aliceBalances = await rollupState.getBalances(ALICE_ADDRESS)
      aliceBalances[UNI_TOKEN_TYPE].should.equal(
        getGenesisState()[ALICE_GENESIS_STATE_INDEX].balances[UNI_TOKEN_TYPE] -
          uniInput
      )
      aliceBalances[PIGI_TOKEN_TYPE].should.equal(
        getGenesisState()[ALICE_GENESIS_STATE_INDEX].balances[PIGI_TOKEN_TYPE] +
          expectedPigiAfterFees
      )

      // And we should have the opposite balances for uniswap
      const uniswapBalances = await rollupState.getBalances(UNISWAP_ADDRESS)
      uniswapBalances[UNI_TOKEN_TYPE].should.equal(
        getGenesisState()[UNISWAP_GENESIS_STATE_INDEX].balances[
          UNI_TOKEN_TYPE
        ] + uniInput
      )
      uniswapBalances[PIGI_TOKEN_TYPE].should.equal(
        getGenesisState()[UNISWAP_GENESIS_STATE_INDEX].balances[
          PIGI_TOKEN_TYPE
        ] - expectedPigiAfterFees
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
        getGenesisStateLargeEnoughForFees()[UNISWAP_GENESIS_STATE_INDEX]
          .balances[UNI_TOKEN_TYPE],
        getGenesisStateLargeEnoughForFees()[UNISWAP_GENESIS_STATE_INDEX]
          .balances[PIGI_TOKEN_TYPE],
        feeBasisPoints
      )

      txAliceSwapUni = {
        signature: ALICE_ADDRESS,
        transaction: {
          sender: ALICE_ADDRESS,
          tokenType: UNI_TOKEN_TYPE,
          inputAmount: uniInput,
          minOutputAmount: expectedPigiAfterFees,
          timeout: +new Date() + 1000,
        },
      }

      await rollupState.applyTransaction(txAliceSwapUni)

      const aliceBalances = await rollupState.getBalances(ALICE_ADDRESS)
      aliceBalances[UNI_TOKEN_TYPE].should.equal(
        getGenesisStateLargeEnoughForFees()[ALICE_GENESIS_STATE_INDEX].balances[
          UNI_TOKEN_TYPE
        ] - uniInput
      )
      aliceBalances[PIGI_TOKEN_TYPE].should.equal(
        getGenesisStateLargeEnoughForFees()[ALICE_GENESIS_STATE_INDEX].balances[
          PIGI_TOKEN_TYPE
        ] + expectedPigiAfterFees
      )
      // And we should have the opposite balances for uniswap

      const uniswapBalances = await rollupState.getBalances(UNISWAP_ADDRESS)
      uniswapBalances[UNI_TOKEN_TYPE].should.equal(
        getGenesisStateLargeEnoughForFees()[UNISWAP_GENESIS_STATE_INDEX]
          .balances[UNI_TOKEN_TYPE] + uniInput
      )
      uniswapBalances[PIGI_TOKEN_TYPE].should.equal(
        getGenesisStateLargeEnoughForFees()[UNISWAP_GENESIS_STATE_INDEX]
          .balances[PIGI_TOKEN_TYPE] - expectedPigiAfterFees
      )
    })
  })
})
