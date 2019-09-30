import MemDown from 'memdown'
import './setup'
import {
  DB,
  BaseDB,
  IdentityVerifier,
  hexStrToBuf,
  bufToHexString,
  SignatureVerifier,
  ForAllSuchThatDecider,
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
  RollupBlock,
  ValidationOutOfOrderError,
} from '../src'
import { resolve } from 'dns'
import { Transaction } from 'ethers/utils'
import { DH_CHECK_P_NOT_SAFE_PRIME } from 'constants'

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
    it('should get right inclusion proof for a swap', async () => {
      // pull initial root to compare later
      const genesisStateRootBuf: Buffer = await rollupGuard.rollupMachine.getStateRoot()
      const genesisStateRoot: string = bufToHexString(genesisStateRootBuf)
      // construct a swap transition
      const swapTransition: SwapTransition = {
        stateRoot: 'DOESNT_MATTER',
        senderSlotIndex: ALICE_GENESIS_STATE_INDEX,
        uniswapSlotIndex: UNISWAP_GENESIS_STATE_INDEX,
        tokenType: UNI_TOKEN_TYPE,
        inputAmount: 100,
        minOutputAmount: 20,
        timeout: 10,
        signature: ALICE_ADDRESS
      }
      const snaps: StateSnapshot[] = await rollupGuard.getInputStateSnapshots(
        swapTransition
      )
      // make sure the right root was pulled
      snaps[0].stateRoot.should.equal(genesisStateRoot.replace('0x', ''))
      snaps[1].stateRoot.should.equal(genesisStateRoot.replace('0x', ''))
      // make sure the right pubkeys were pulled
      snaps[0].state.pubKey.should.equal(ALICE_ADDRESS)
      snaps[1].state.pubKey.should.equal(UNISWAP_ADDRESS)
    })
    it('should get right inclusion proof for a non creation transfer', async () => {
      // pull initial root to compare later
      const genesisStateRootBuf: Buffer = await rollupGuard.rollupMachine.getStateRoot()
      const genesisStateRoot: string = bufToHexString(genesisStateRootBuf)
      // construct a transfer transition
      const transferTransition: TransferTransition = {
        stateRoot: 'DOESNT_MATTER',
        senderSlotIndex: ALICE_GENESIS_STATE_INDEX,
        recipientSlotIndex: 3, // Bob hardcoded in our genesis state helper
        tokenType: UNI_TOKEN_TYPE,
        amount: 10,
        signature: ALICE_ADDRESS
      }
      const snaps: StateSnapshot[] = await rollupGuard.getInputStateSnapshots(
        transferTransition
      )
      // make sure the right root was pulled
      snaps[0].stateRoot.should.equal(genesisStateRoot.replace('0x', ''))
      snaps[1].stateRoot.should.equal(genesisStateRoot.replace('0x', ''))
      // make sure the right pubkeys were pulled
      snaps[0].state.pubKey.should.equal(ALICE_ADDRESS)
      snaps[1].state.pubKey.should.equal(BOB_ADDRESS)
    })
  })

  describe.skip('getTransactionFromTransition', async () => {
    it('should get a transfer from the transition', async () => {
      // let resTx: SignedTransaction = await rollupGuard.getTransactionFromTransition(transitionAliceToBob)
      // console.log('converted transition to tx: ')
      // console.log(resTx)
    })
  })

  describe('checkNextTransition', () => {

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

      const res: FraudCheckResult = await rollupGuard.checkNextTransition(
        transitionAliceToBob
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

      const res: FraudCheckResult = await rollupGuard.checkNextTransition(
        transitionAliceSwap
      )
      res.should.equal('NO_FRAUD')
    })
    it('should return positive for fraud if transition has invalid root', async () => {
      const wrongPostRoot: string =
      '0xdeadbeefb833c9e1086ded944c9fbe011248203e586d81f9fe0922434632dcde'

    const transitionAliceSwap: SwapTransition = {
      stateRoot: wrongPostRoot,
      senderSlotIndex: 0,
      uniswapSlotIndex: UNISWAP_GENESIS_STATE_INDEX,
      tokenType: UNI_TOKEN_TYPE,
      inputAmount: 100,
      minOutputAmount: 20,
      timeout: 10,
      signature: ALICE_ADDRESS,
    }

    const res: FraudCheckResult = await rollupGuard.checkNextTransition(
      transitionAliceSwap
    )
    res.should.not.equal('NO_FRAUD')
    })
  })

  describe('checkNextBlock', () => {
    it('should throw if it recieves blocks out of order', async () => {
      const wrongOrderBlock: RollupBlock = {
        number: 5,
        transitions: undefined
      }
      try {
        await rollupGuard.checkNextBlock(wrongOrderBlock)
      } catch(e) {
        e.should.be.an.instanceOf(ValidationOutOfOrderError)
      }
    })
    it('should successfully validate a send followed by a swap', async () => {
      const postTransferRoot: string =
        '0x8bb6f1bd59e26928f8f1531af52224d59d76d6951db31c403bf1e215c99372e6'
      const transitionAliceToBob: TransferTransition = {
        stateRoot: postTransferRoot,
        senderSlotIndex: 0,
        recipientSlotIndex: 3,
        tokenType: 0,
        amount: 100,
        signature: ALICE_ADDRESS,
      }
      
      const postSwapRoot: string =
        '0x3b1537dac24e21efd3fa80ce5698f5838e45c62efca5ecde0152f9b165ce6813'
      const transitionAliceSwap: SwapTransition = {
        stateRoot: postSwapRoot,
        senderSlotIndex: 0,
        uniswapSlotIndex: UNISWAP_GENESIS_STATE_INDEX,
        tokenType: UNI_TOKEN_TYPE,
        inputAmount: 100,
        minOutputAmount: 20,
        timeout: 10,
        signature: ALICE_ADDRESS,
      }

      const sendThenSwapBlock: RollupBlock = {
        number: 1,
        transitions: [transitionAliceToBob, transitionAliceSwap]
      }

      const res: FraudCheckResult = await rollupGuard.checkNextBlock(sendThenSwapBlock)
      res.should.equal('NO_FRAUD')
    })
  })
})
