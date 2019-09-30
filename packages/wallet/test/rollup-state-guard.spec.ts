import MemDown from 'memdown'
import './setup'
import { DB, BaseDB, IdentityVerifier, hexStrToBuf, bufToHexString } from '@pigi/core'

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
  FraudCheckResult,
  CreateAndTransferTransition,
  StateSnapshot,
  RollupTransition,
  TransferTransition,
  abiEncodeTransition,
} from '../src'
import { resolve } from 'dns'

/* External Imports */

// import { generateNTransitions, RollupBlock } from '../../contracts/build/test/helpers'

/* Internal Imports */

/*********
 * TESTS *
 *********/

describe.only('RollupStateMachine', () => {
  let rollupGuard: DefaultRollupStateGuard
  let stateDb: DB

  // beforeEach(async () => {})

  before(async () => {
    stateDb = new BaseDB(new MemDown('') as any, 256)
    rollupGuard = await DefaultRollupStateGuard.create(
      getGenesisStateLargeEnoughForFees(),
      stateDb
    )
  })

  afterEach(async () => {
    await stateDb.close()
  })

  describe('initialization', () => {
    it('should create Guarder with a rollup machine', async () => {
      rollupGuard.rollupMachine.should.not.be.undefined
    })
  })
  
  describe('getInputStateSnapshots', () => {
      it('should get right inclusion proof for an account which is about to be created', async () => {
          const stateRootBuf: Buffer =  await rollupGuard.rollupMachine.getStateRoot()
          const stateRootStr: string = bufToHexString(stateRootBuf)
          const creationTransition: CreateAndTransferTransition = {
            stateRoot: stateRootStr,
            senderSlotIndex: 0,
            recipientSlotIndex: 10,
            tokenType: 0,
            amount: 1,
            signature: 'FAKE_SIG',
            createdAccountPubkey: 'FAKE_ADDR',
          }
          const res: StateSnapshot[] = await rollupGuard.getInputStateSnapshots(creationTransition)
          console.log('snapshot result is: ')
          console.log(res)
      })
  })

  describe.only('checkNextTransition', () => {
    // const txAliceToBob: SignedTransaction = {
    //   signature: ALICE_ADDRESS,
    //   transaction: {
    //     sender: ALICE_ADDRESS,
    //     recipient: BOB_ADDRESS,
    //     tokenType: UNI_TOKEN_TYPE,
    //     amount: 5,
    //   },
    // }
    it('should return no fraud if correct root', async () => {

        const postRoot: string = '0x0000000000000000000000000000000000000000000000000000000000000000'

        const transitionAliceToBob: TransferTransition = {
            stateRoot: postRoot,
            senderSlotIndex: 0,
            recipientSlotIndex: 3,
            tokenType: 0,
            amount: 100,
            signature: '0x0a0a0a0a',
        }

        const transAliceToBobEncoded: string = abiEncodeTransition(transitionAliceToBob)

      const res: FraudCheckResult = await rollupGuard.checkNextEncodedTransition(
        transAliceToBobEncoded,
        hexStrToBuf(postRoot)
      )
      res.should.equal('NO_FRAUD')
    })
  })
})
