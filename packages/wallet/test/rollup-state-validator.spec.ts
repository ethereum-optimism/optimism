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
  DefaultRollupStateValidator,
  SignedTransaction,
  PIGI_TOKEN_TYPE,
  RollupStateValidator,
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
  AggregatorUnsupportedError,
  parseTransactionFromABI,
  parseTransitionFromABI,
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

describe.only('RollupStateValidator', () => {
  let rollupGuard: DefaultRollupStateValidator
  let stateDb: DB

  beforeEach(async () => {
    stateDb = new BaseDB(new MemDown('') as any, 256)
    rollupGuard = await DefaultRollupStateValidator.create(
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
        signature: ALICE_ADDRESS,
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
        signature: ALICE_ADDRESS,
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
    it.skip('should get right inclusion proof for a createAndTransfer', async () => {
      // pull initial root to compare later
      const genesisStateRootBuf: Buffer = await rollupGuard.rollupMachine.getStateRoot()
      const genesisStateRoot: string = bufToHexString(genesisStateRootBuf)
      // construct a transfer transition
      const creationTransition: CreateAndTransferTransition = {
        stateRoot: 'DOESNT_MATTER',
        senderSlotIndex: ALICE_GENESIS_STATE_INDEX,
        recipientSlotIndex: 4, // Bob hardcoded in our genesis state helper as index 3
        tokenType: UNI_TOKEN_TYPE,
        amount: 10,
        signature: ALICE_ADDRESS,
        createdAccountPubkey: BOB_ADDRESS
      }
      const snaps: StateSnapshot[] = await rollupGuard.getInputStateSnapshots(
        creationTransition
      )
      // make sure the right root was pulled
      snaps[0].stateRoot.should.equal(genesisStateRoot.replace('0x', ''))
      snaps[1].stateRoot.should.equal(genesisStateRoot.replace('0x', ''))
      // make sure the right pubkeys were pulled
      snaps[0].state.pubKey.should.equal(ALICE_ADDRESS)
      snaps[1].state.pubKey.should.equal(BOB_ADDRESS)
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
    it('should throw if accounts are not created sequentially', async () => {
      const postRoot = '0xdeadbeef1e5'

      const outOfOrderCreation: CreateAndTransferTransition = {
        stateRoot: postRoot,
        senderSlotIndex: 0,
        recipientSlotIndex: 300, // not 300th yet!
        tokenType: 0,
        amount: 100,
        signature: ALICE_ADDRESS,
        createdAccountPubkey: BOB_ADDRESS,
      }

      try {
        await rollupGuard.checkNextTransition(outOfOrderCreation)
      } catch (error) {
        error.should.be.instanceOf(AggregatorUnsupportedError)
        return
      }
      false.should.equal(true) // we should never get here!
    })
  })

  describe('checkNextBlock', () => {
    it('should throw if it recieves blocks out of order', async () => {
      const wrongOrderBlock: RollupBlock = {
        blockNumber: 5,
        transitions: undefined,
      }
      try {
        await rollupGuard.checkNextBlock(wrongOrderBlock)
      } catch (e) {
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
        blockNumber: 1,
        transitions: [transitionAliceToBob, transitionAliceSwap],
      }

      const res: FraudCheckResult = await rollupGuard.checkNextBlock(
        sendThenSwapBlock
      )
      res.should.equal('NO_FRAUD')
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
        '0xdeadbeef3b1531efd3fa80ce5698f5838e45c62efca5ecde0152f9b165ce6813'
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
        blockNumber: 1,
        transitions: [transitionAliceToBob, transitionAliceSwap],
      }

      const res: FraudCheckResult = await rollupGuard.checkNextBlock(
        sendThenSwapBlock
      )
      res.should.not.equal('NO_FRAUD')
    })
    it('should return a fraud proof for a second transition with invalid root', async () => {
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
        '0xdeadbeef3b1531efd3fa80ce5698f5838e45c62efca5ecde0152f9b165ce6813' // invalid post root
      const transitionInvalidSwap: SwapTransition = {
        stateRoot: postSwapRoot,
        senderSlotIndex: 0,
        uniswapSlotIndex: UNISWAP_GENESIS_STATE_INDEX,
        tokenType: UNI_TOKEN_TYPE,
        inputAmount: 100,
        minOutputAmount: 20,
        timeout: 10,
        signature: ALICE_ADDRESS,
      }

      const sendThenInvalidSwapBlock: RollupBlock = {
        blockNumber: 1,
        transitions: [transitionAliceToBob, transitionInvalidSwap],
      }

      const res: FraudCheckResult = await rollupGuard.checkNextBlock(
        sendThenInvalidSwapBlock
      )

      // should give first and second transitions
      res[0].inclusionProof.transitionIndex.should.equal(0)
      res[1].inclusionProof.transitionIndex.should.equal(1)
    })
  })
})
