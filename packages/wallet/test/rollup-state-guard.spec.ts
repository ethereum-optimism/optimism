import MemDown from 'memdown'
import './setup'
import { DB, BaseDB, IdentityVerifier } from '@pigi/core'

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
  DefaultRollupStateMachine,
  DefaultRollupStateGuard,
  SignedTransaction,
  PIGI_TOKEN_TYPE,
  RollupStateGuard,
} from '../src'

/* External Imports */

/* Internal Imports */

/*********
 * TESTS *
 *********/

describe.only('RollupStateMachine', () => {
  let rollupGuard: RollupStateGuard
  let stateDb: DB

  // beforeEach(async () => {})

  afterEach(async () => {
    await stateDb.close()
  })

  describe('DefaultRollupStateGuard', () => {
    it('should create Guarder successfully', async () => {
      stateDb = new BaseDB(new MemDown('') as any, 256)
      rollupGuard = await DefaultRollupStateGuard.create(
        getGenesisState(),
        stateDb
      )
    })
  })
})
