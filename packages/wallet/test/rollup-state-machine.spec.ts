import './setup'

/* External Imports */
import {
  DB,
  IdentityVerifier,
  newInMemoryDB,
  SparseMerkleTreeImpl,
  ZERO,
} from '@pigi/core'
import * as assert from 'assert'

/* Internal Imports */
import {
  AGGREGATOR_ADDRESS,
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
  DefaultRollupStateMachine,
  SignedTransaction,
  PIGI_TOKEN_TYPE,
  NON_EXISTENT_SLOT_INDEX,
  StateSnapshot,
} from '../src'

/*********
 * TESTS *
 *********/

describe('RollupStateMachine', () => {
  let rollupState: DefaultRollupStateMachine
  let db: DB

  beforeEach(async () => {
    db = newInMemoryDB()
    rollupState = (await DefaultRollupStateMachine.create(
      getGenesisState(),
      db,
      AGGREGATOR_ADDRESS,
      IdentityVerifier.instance()
    )) as DefaultRollupStateMachine
  })

  describe('getState', () => {
    it('should not throw even if the account doesnt exist', async () => {
      const response = await rollupState.getState('this is not an address!')
      response.slotIndex.should.equal(NON_EXISTENT_SLOT_INDEX)
      assert(
        response.state === undefined,
        'State should be undefined for non-existent address'
      )
      assert(
        response.inclusionProof === undefined,
        'Inclusion proof should be undefined for non-existent address'
      )
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
      const aliceState: StateSnapshot = await rollupState.getState(
        ALICE_ADDRESS
      )
      aliceState.state.balances.should.deep.equal(
        getGenesisState()[ALICE_GENESIS_STATE_INDEX].balances
      )
      await rollupState.applyTransaction(txAliceToBob)
    })

    it('should update balances after transfer', async () => {
      await rollupState.applyTransaction(txAliceToBob)

      const aliceState: StateSnapshot = await rollupState.getState(
        ALICE_ADDRESS
      )
      aliceState.state.balances[UNI_TOKEN_TYPE].should.equal(
        getGenesisState()[ALICE_GENESIS_STATE_INDEX].balances[UNI_TOKEN_TYPE] -
          5
      )

      const bobState: StateSnapshot = await rollupState.getState(BOB_ADDRESS)
      bobState.state.balances[UNI_TOKEN_TYPE].should.deep.equal(5)
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

      const aliceState: StateSnapshot = await rollupState.getState(
        ALICE_ADDRESS
      )
      aliceState.state.balances[UNI_TOKEN_TYPE].should.equal(
        getGenesisState()[ALICE_GENESIS_STATE_INDEX].balances[UNI_TOKEN_TYPE] -
          uniInput
      )
      aliceState.state.balances[PIGI_TOKEN_TYPE].should.equal(
        getGenesisState()[ALICE_GENESIS_STATE_INDEX].balances[PIGI_TOKEN_TYPE] +
          expectedPigiAfterFees
      )

      // And we should have the opposite balances for uniswap
      const uniswapState: StateSnapshot = await rollupState.getState(
        UNISWAP_ADDRESS
      )
      uniswapState.state.balances[UNI_TOKEN_TYPE].should.equal(
        getGenesisState()[UNISWAP_GENESIS_STATE_INDEX].balances[
          UNI_TOKEN_TYPE
        ] + uniInput
      )
      uniswapState.state.balances[PIGI_TOKEN_TYPE].should.equal(
        getGenesisState()[UNISWAP_GENESIS_STATE_INDEX].balances[
          PIGI_TOKEN_TYPE
        ] - expectedPigiAfterFees
      )
    })

    it('should update balances after swap including fee', async () => {
      const feeBasisPoints = 30
      rollupState = (await DefaultRollupStateMachine.create(
        getGenesisStateLargeEnoughForFees(),
        db,
        AGGREGATOR_ADDRESS,
        IdentityVerifier.instance()
      )) as DefaultRollupStateMachine

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

      const aliceState: StateSnapshot = await rollupState.getState(
        ALICE_ADDRESS
      )
      aliceState.state.balances[UNI_TOKEN_TYPE].should.equal(
        getGenesisStateLargeEnoughForFees()[ALICE_GENESIS_STATE_INDEX].balances[
          UNI_TOKEN_TYPE
        ] - uniInput
      )
      aliceState.state.balances[PIGI_TOKEN_TYPE].should.equal(
        getGenesisStateLargeEnoughForFees()[ALICE_GENESIS_STATE_INDEX].balances[
          PIGI_TOKEN_TYPE
        ] + expectedPigiAfterFees
      )
      // And we should have the opposite balances for uniswap

      const uniswapState = await rollupState.getState(UNISWAP_ADDRESS)
      uniswapState.state.balances[UNI_TOKEN_TYPE].should.equal(
        getGenesisStateLargeEnoughForFees()[UNISWAP_GENESIS_STATE_INDEX]
          .balances[UNI_TOKEN_TYPE] + uniInput
      )
      uniswapState.state.balances[PIGI_TOKEN_TYPE].should.equal(
        getGenesisStateLargeEnoughForFees()[UNISWAP_GENESIS_STATE_INDEX]
          .balances[PIGI_TOKEN_TYPE] - expectedPigiAfterFees
      )
    })
  })

  describe('test recovery', () => {
    before(async () => {
      db = newInMemoryDB()
    })

    it('should initialize with no previous state', async () => {
      rollupState = (await DefaultRollupStateMachine.create(
        getGenesisState(),
        db,
        AGGREGATOR_ADDRESS,
        IdentityVerifier.instance()
      )) as DefaultRollupStateMachine
    })

    it('should initialize with previous state and ignore genesis state', async () => {
      const tree: SparseMerkleTreeImpl = await SparseMerkleTreeImpl.create(
        db,
        undefined,
        32
      )

      await Promise.all([
        db.put(
          DefaultRollupStateMachine.ADDRESS_TO_KEYS_COUNT_KEY,
          Buffer.from('1')
        ),
        db.put(DefaultRollupStateMachine.LAST_OPEN_KEY, ZERO.toBuffer()),
        db.put(
          DefaultRollupStateMachine.getAddressMapDBKey(0),
          DefaultRollupStateMachine.serializeAddressToKeyForDB(
            AGGREGATOR_ADDRESS,
            ZERO
          )
        ),
        tree.update(
          ZERO,
          DefaultRollupStateMachine.serializeBalances(AGGREGATOR_ADDRESS, {
            [UNI_TOKEN_TYPE]: 999,
            [PIGI_TOKEN_TYPE]: 9_999,
          })
        ),
      ])

      await db.put(DefaultRollupStateMachine.ROOT_KEY, await tree.getRootHash())

      rollupState = (await DefaultRollupStateMachine.create(
        getGenesisState(),
        db,
        AGGREGATOR_ADDRESS,
        IdentityVerifier.instance()
      )) as DefaultRollupStateMachine

      rollupState.getUsedKeys().size.should.equal(1)
      rollupState
        .getUsedKeys()
        .has(ZERO.toString())
        .should.equal(true)
      assert(
        rollupState.getLastOpenKey().equals(ZERO),
        'Last open key should be 0'
      )
      rollupState.getAddressesToKeys().size.should.equal(1)
      assert(
        rollupState
          .getAddressesToKeys()
          .get(AGGREGATOR_ADDRESS)
          .equals(ZERO),
        'aggregator address key should be 0'
      )

      const aggregatorState: StateSnapshot = await rollupState.getState(
        AGGREGATOR_ADDRESS
      )
      aggregatorState.state.balances[UNI_TOKEN_TYPE].should.equal(999)
      aggregatorState.state.balances[PIGI_TOKEN_TYPE].should.equal(9_999)
    })
  })
})
