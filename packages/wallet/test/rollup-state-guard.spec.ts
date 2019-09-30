import MemDown from 'memdown'
import './setup'
import {
  DB,
  BaseDB,
  IdentityVerifier,
  hexStrToBuf,
  bufToHexString,
  SignatureVerifier,
} from '@pigi/core'

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
  AGGREGATOR_ADDRESS,
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
  State,
  SwapTransition,
} from '../src'
import { resolve } from 'dns'
import { Transaction } from 'ethers/utils'

/* External Imports */

// import { generateNTransitions, RollupBlock } from '../../contracts/build/test/helpers'

/* Internal Imports */

/*********
 * HELPERS *
 *********/

function getMultiBalanceGenesis(
  aliceAddress: string = ALICE_ADDRESS,
  bobAddress: string = BOB_ADDRESS
): State[] {
  return [
    {
      pubKey: aliceAddress,
      balances: {
        [UNI_TOKEN_TYPE]: 5_000,
        [PIGI_TOKEN_TYPE]: 5_000,
      },
    },
    {
      pubKey: UNISWAP_ADDRESS,
      balances: {
        [UNI_TOKEN_TYPE]: 650_000,
        [PIGI_TOKEN_TYPE]: 650_000,
      },
    },
    {
      pubKey: AGGREGATOR_ADDRESS,
      balances: {
        [UNI_TOKEN_TYPE]: 1_000_000,
        [PIGI_TOKEN_TYPE]: 1_000_000,
      },
    },
    {
      pubKey: bobAddress,
      balances: {
        [UNI_TOKEN_TYPE]: 5_000,
        [PIGI_TOKEN_TYPE]: 5_000,
      },
    },
  ]
}

class MockSignatureVerifier implements SignatureVerifier {
  public verifyMessage(message: string, signature: string): string {
    return ALICE_ADDRESS
  }
}

/*********
 * TESTS *
 *********/

describe.only('RollupStateMachine', () => {
  let rollupGuard: DefaultRollupStateGuard
  let stateDb: DB

  beforeEach(async () => {
    stateDb = new BaseDB(new MemDown('') as any, 256)
    rollupGuard = await DefaultRollupStateGuard.create(
      getMultiBalanceGenesis(),
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
      const stateRootBuf: Buffer = await rollupGuard.rollupMachine.getStateRoot()
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
      const res: StateSnapshot[] = await rollupGuard.getInputStateSnapshots(
        creationTransition
      )
      console.log('snapshot result is: ')
      console.log(res)
    })
  })

  describe.skip('getTransactionFromTransition', async () => {
    it('should get a transfer from the transition', async () => {
      // let resTx: SignedTransaction = await rollupGuard.getTransactionFromTransition(transitionAliceToBob)
      // console.log('converted transition to tx: ')
      // console.log(resTx)
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
    it('should return no fraud if correct root for transfer', async () => {
      const postRoot: string =
        '0x8bb6f1bd59e26928f8f1531af52224d59d76d6951db31c403bf1e215c99372e6'

      const transitionAliceToBob: TransferTransition = {
        stateRoot: postRoot,
        senderSlotIndex: 0,
        recipientSlotIndex: 3,
        tokenType: 0,
        amount: 100,
        signature: ALICE_ADDRESS,
      }
      const transAliceToBobEncoded: string = abiEncodeTransition(
        transitionAliceToBob
      )

      const res: FraudCheckResult = await rollupGuard.checkNextEncodedTransition(
        transAliceToBobEncoded
      )
      res.should.equal('NO_FRAUD')
    })

    it('should return no fraud if correct root for swap', async () => {
      const postRoot: string =
        '0x773015e9b833c9e1086ded944c9fbe011248203e586d81f9fe0922434632dcde'

      const transitionAliceSwap: SwapTransition = {
        stateRoot: postRoot,
        senderSlotIndex: 0,
        uniswapSlotIndex: UNISWAP_GENESIS_STATE_INDEX,
        tokenType: UNI_TOKEN_TYPE,
        inputAmount: 100,
        minOutputAmount: 20,
        timeout: 10,
        signature: ALICE_ADDRESS,
      }
      const transAliceSwapEncoded: string = abiEncodeTransition(
        transitionAliceSwap
      )

      const res: FraudCheckResult = await rollupGuard.checkNextEncodedTransition(
        transAliceSwapEncoded
      )
      res.should.equal('NO_FRAUD')
    })
  })
})
